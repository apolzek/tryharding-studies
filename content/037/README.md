# 037 — k0s POC em containers (Ubuntu 24.04)

POC de cluster Kubernetes com **k0s** (distribuição leve do k8s), usando
**1 control plane + 2 workers**, cada nó é um container Docker rodando
uma imagem Ubuntu 24.04 customizada. O objetivo é validar a automação
antes de subir em uma VM.

## Estrutura

```
037/
├── Dockerfile                 # ubuntu:24.04 + k0s + hardening
├── docker-compose.yml         # 1 controller + 2 workers, mesma rede
├── Makefile
├── config/
│   ├── k0s.yaml               # config do cluster (hardening no kube-apiserver)
│   └── audit-policy.yaml      # política de auditoria do apiserver
├── scripts/
│   ├── install-k0s.sh         # build-time: baixa o binário k0s
│   ├── entrypoint.sh          # runtime: sobe controller ou worker
│   └── test-cluster.sh        # host-side: valida nodes e sobe um app
└── manifests/
    └── sample-app.yaml        # namespace PSA + Deployment + PVC + NetworkPolicy
```

## Como os containers se enxergam

```mermaid
flowchart LR
    HOST["host :6443"]
    subgraph NET["docker network: k0snet (bridge)"]
        CONTROLLER["controller<br/>(k0s API)<br/>:6443 :8132 :9443"]
        WORKER1["worker-1<br/>(kubelet)"]
        WORKER2["worker-2"]
        SHARED["shared volume<br/>/shared/worker.token"]
    end

    HOST -->|kubeconfig de fora do docker| CONTROLLER
    WORKER1 -->|6443| CONTROLLER
    WORKER2 -->|6443| CONTROLLER
    CONTROLLER -->|writes token| SHARED
    SHARED -->|reads token| WORKER1
    SHARED -->|reads token| WORKER2
```

- **Troca de token**: o controller emite o join token para worker quando o
  apiserver fica pronto e escreve em `/shared/worker.token` (named volume
  compartilhado, montado **read-only** nos workers). Os workers ficam em
  poll até o arquivo aparecer.
- **CNI**: `kube-router` (default do k0s) em modo iptables.
- **Datastore**: `kine` sobre SQLite — single-node CP, leve. Para HA use
  etcd (mudar `spec.storage.type: etcd` em `config/k0s.yaml`).

## Hardening aplicado

**Imagem**
- Base `ubuntu:24.04`, apenas pacotes necessários (`iproute2`, `iptables`,
  `kmod`, `conntrack`, `ethtool`, `socat`, `jq`, `tini`, `ca-certificates`).
- Sem SSH, sem docs, caches do apt removidos, `/usr/share/{doc,man}` limpos.
- `tini` como PID 1 para propagação correta de sinais / reap de zumbis.
- `k0s.yaml` e `audit-policy.yaml` com `0640`.

**kube-apiserver** (via `spec.api.extraArgs` em `config/k0s.yaml`)
- `anonymous-auth=false`
- `authorization-mode=Node,RBAC`
- `profiling=false`
- `tls-min-version=VersionTLS12`
- Audit log habilitado em `/var/log/k0s/audit.log` com rotação.

**kube-controller-manager / scheduler**
- `bind-address=127.0.0.1` (métricas/debug só locais)
- `profiling=false`

**Rede**
- CNI kube-router com NetworkPolicy ativo.

**Cargas de trabalho (sample-app)**
- Namespace com labels de **Pod Security Admission** (`enforce=baseline`,
  `warn/audit=restricted`).
- Container rodando **não-root** (uid/gid 101, `runAsNonRoot: true`).
- `readOnlyRootFilesystem: true`, `allowPrivilegeEscalation: false`,
  `capabilities: drop: ["ALL"]`, `seccompProfile: RuntimeDefault`.
- `resources.requests/limits` definidos (evita noisy neighbour / OOM geral).
- Imagem `nginxinc/nginx-unprivileged` (bind em :8080, não precisa de root).
- `NetworkPolicy` default-deny de ingress + `allow-web` abrindo apenas a
  porta 8080 do pod `app=web`.

## Uso

```bash
make build      # builda a imagem ubuntu+k0s
make up         # sobe controller + 2 workers
make nodes      # kubectl get nodes -o wide
make test       # aplica sample-app, valida rollout e faz smoke curl
make kubeconfig # exporta kubeconfig para usar kubectl de fora
make down       # para tudo (volumes ficam)
make clean      # down -v + remove kubeconfig
```

Acesso via host:

```bash
make kubeconfig
export KUBECONFIG=$PWD/kubeconfig
kubectl get nodes
```

## Disco / storage

### Como fica nos containers (POC)

Cada nó tem volumes nomeados:

| Volume              | Mount            | Função                                    |
|---------------------|------------------|-------------------------------------------|
| `controller-data`   | `/var/lib/k0s`   | Estado do CP (kine/SQLite, PKI, manifests)|
| `controller-log`    | `/var/log/k0s`   | Audit log do apiserver                    |
| `worker-N-data`     | `/var/lib/k0s`   | kubelet state, CNI, logs                  |
| `worker-N-disk`     | `/mnt/data`      | "Disco de dados" para PVs hostPath/local  |
| `shared`            | `/shared`        | Troca do join token                       |

### PV / PVC no POC

O `manifests/sample-app.yaml` cria um `PersistentVolume` do tipo `local`
apontando para `/mnt/data/demo` no **worker-1**, com `nodeAffinity` forçando
o pod a ser agendado nesse nó. Storage class `local-hostpath` (sem
provisioner — volume criado manualmente, modo estático).

Fluxo:
1. `test-cluster.sh` cria `/mnt/data/demo` no container do worker-1 (via
   `docker exec`). Em uma VM, seria `mkdir` direto no disco.
2. PV `demo-pv-worker-1` declara `nodeAffinity: worker-1` + `local.path: /mnt/data/demo`.
3. PVC `demo-pvc` (RWO, 1Gi) bind ao PV quando o pod é agendado.
4. Deployment `web` (1 réplica, `nodeSelector: worker-1`) monta o PVC em
   `/usr/share/nginx/html`. Um `initContainer` semeia `index.html` no PV.
5. Nginx serve o conteúdo lido do disco local do worker-1.

Por que `local` e não `hostPath`?  `hostPath` não expressa afinidade de nó
e quebra se o scheduler colocar o pod em outro worker. `local` PV é a
forma recomendada de expor disco local no k8s.

### Em produção (VM)

Na VM, use um **disco dedicado** para os dados do k0s/containerd:

```bash
# exemplo: /dev/vdb dedicado, XFS, montado com noatime
mkfs.xfs -f -L k0s /dev/vdb
mkdir -p /var/lib/k0s
echo 'LABEL=k0s /var/lib/k0s xfs defaults,noatime,nodiratime 0 2' >> /etc/fstab
mount -a
```

Por quê: `/var/lib/k0s` guarda imagens de container, overlayfs layers,
logs dos pods e o datastore (kine/etcd). Em single volume com o root,
uma Pod barulhenta pode encher o disco e derrubar o SO. Mount separado
é isolamento barato.

Para storage dinâmico dentro do cluster na VM, opções:

- **local-path-provisioner** (Rancher) — cria PVs automaticamente em um
  diretório do nó. Simples, bom para single-node/edge.
- **Longhorn** / **OpenEBS** — replicação entre nós, snapshots.
- **CSI** do provedor (EBS, Ceph-CSI, NFS-CSI) em clusters maiores.

Para o POC em containers a escolha foi PV estático para deixar explícito
o mapeamento `worker-1-disk` ↔ `/mnt/data` ↔ PV.

## Ajustes específicos do "k8s em Docker" (NÃO usar na VM)

Para o cluster subir dentro do Docker, o `entrypoint.sh` passa para o
kubelet flags que **só fazem sentido em container**:

- `--cgroups-per-qos=false --enforce-node-allocatable=` — desliga o
  gerenciamento da hierarquia QoS (Burstable/Guaranteed/BestEffort) do
  kubelet, que colide com a delegação de cgroup v2 em container.
- `--resolv-conf=/etc/resolv.conf.k0s` — evita o loop do CoreDNS quando o
  `/etc/resolv.conf` do container aponta para o DNS interno do Docker
  (127.0.0.11), que o CoreDNS detecta como loop e morre.
- `cgroup: host` no compose — compartilha cgroupns com o host (kubelet
  consegue criar `/sys/fs/cgroup/k8s.io/...` com todos os controllers).

Em VM real, **remova** essas flags: o kubelet faz QoS enforcement
correto, o resolv.conf do host não tem loops e cgroupns não é problema.
A variável `K0S_KUBELET_EXTRA_ARGS` permite sobrescrever sem editar o
entrypoint.

## Migração para VM

1. Subir VM com Ubuntu 24.04, 2+ vCPU, 4+ GiB RAM, disco de dados separado
   em `/var/lib/k0s` (ver seção de disco).
2. Copiar `config/k0s.yaml` + `config/audit-policy.yaml` para `/etc/k0s/`.
3. `curl -sSfL https://get.k0s.sh | sudo sh` (ou copiar o binário).
4. Controller: `sudo k0s install controller --config /etc/k0s/k0s.yaml && sudo k0s start`.
5. Token para worker: `sudo k0s token create --role=worker --expiry=24h`.
6. Em cada worker: `sudo k0s install worker --token-file <(echo "<TOKEN>") && sudo k0s start`.
7. Validar: `sudo k0s kubectl get nodes`.

Nada das flags extras do kubelet é necessário.

## Validação (executada nesta POC)

```
$ make up && make test
...
worker-1   Ready    v1.31.2+k0s
worker-2   Ready    v1.31.2+k0s
deployment "web" successfully rolled out
persistentvolumeclaim/demo-pvc   Bound    demo-pv-worker-1   1Gi
attempt 1 -> HTTP 200
cluster OK
```

E o conteúdo servido vem do disco do worker-1:

```
$ docker exec k0s-worker-1 cat /mnt/data/demo/index.html
hello from k0s on worker-1 PV
```

## Troubleshooting

- `kubectl get pods -n kube-system` mostrando `CrashLoopBackOff` no
  coredns → é o loop de DNS. Confirme que o kubelet está com
  `--resolv-conf=/etc/resolv.conf.k0s` (veja `ps -ef | grep kubelet`
  dentro do worker).
- Workers `NotReady` por muito tempo → `docker logs k0s-worker-1` e
  procurar por "cgroup" ou "cni". kube-router baixa a imagem na primeira
  vez; pode levar alguns minutos.
- Token expirado → recriar: `docker exec k0s-controller k0s token create --role=worker > /tmp/t && docker cp /tmp/t k0s-worker-1:/shared/worker.token`.

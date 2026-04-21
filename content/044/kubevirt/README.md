---
title: KubeVirt — VMs clássicas rodando como pods em Kubernetes
tags: [cncf, incubating, kubevirt, vm, virtualization, kvm]
status: stable
---

## KubeVirt (CNCF Incubating)

**O que é:** extensão do Kubernetes que roda **VMs KVM como pods**. CRDs `VirtualMachine`, `VirtualMachineInstance`, `DataVolume`. Por que? Legacy apps que não containerizam (proprietárias, Windows, appliances de rede) convivem no mesmo cluster das apps cloud-native.

**Quando usar (SRE day-to-day):**

- Consolidar VM + container no mesmo plano de controle — bye OpenStack separado.
- Rodar Windows Server ou apps legadas sem refactor.
- Platform engineering para DBs pesados (SAP, Oracle) que ops prefere VM.
- Sandboxes de segurança (VM isola melhor que container).

**Quando NÃO usar:**

- Workload cloud-native — use container. KubeVirt é para o "mundo antigo" que não vai migrar.
- Nós sem `/dev/kvm` (nested virt) — KubeVirt cai em modo software-emulated, lento demais para prod.

### Cenário real

*"Minha empresa tem 50 VMs legacy (Windows, CentOS) + 200 microsserviços. Quero um único plano (k8s + GitOps) ao invés de 2 stacks (k8s + VMware)."*

### Reproducing

> ⚠️ **Requer `/dev/kvm` no host**. Kind + nested virt pode exigir configuração extra do host Linux; em macOS/Windows não roda.

```bash
cd content/044/kubevirt

# 1. Cluster
kind create cluster --config kind.yaml

# Permite nested virt no node (se host suporta)
docker exec -it kubevirt-poc-control-plane bash -c "modprobe kvm_intel || modprobe kvm_amd || true"

# 2. Instala KubeVirt operator + CR
export VERSION=$(curl -s https://api.github.com/repos/kubevirt/kubevirt/releases/latest | grep tag_name | cut -d '"' -f 4)
kubectl apply -f "https://github.com/kubevirt/kubevirt/releases/download/${VERSION}/kubevirt-operator.yaml"
kubectl apply -f "https://github.com/kubevirt/kubevirt/releases/download/${VERSION}/kubevirt-cr.yaml"

# 2a. Se não tem /dev/kvm — habilite software emulation (lento mas funciona p/ POC)
kubectl -n kubevirt patch kubevirt kubevirt --type=merge \
  -p '{"spec":{"configuration":{"developerConfiguration":{"useEmulation":true}}}}'

# 3. Espera
kubectl -n kubevirt wait kv kubevirt --for=condition=Available --timeout=15m

# 4. VM demo (Cirros — Linux minúsculo)
kubectl apply -f vm.yaml
kubectl wait vm/cirros-vm --for=condition=Ready --timeout=5m

# 5. virtctl para console
curl -L -o virtctl "https://github.com/kubevirt/kubevirt/releases/download/${VERSION}/virtctl-${VERSION}-linux-amd64"
chmod +x virtctl

./virtctl console cirros-vm  # Ctrl+] para sair
# login: cirros / password: gocubsgo
```

### Cleanup

```bash
kind delete cluster --name kubevirt-poc
rm -f virtctl
```

### Tips de SRE

- **CDI** (Containerized Data Importer) para importar disco qcow2/vmdk/ova direto para PVC.
- **Live Migration**: VM move de nó sem downtime (precisa de storage shared/RWX).
- **Hotplug** de disco/rede/CPU/memória — feature essential em DB que não pode reiniciar.
- **Multus** CNI para VM ter múltiplas NICs (mundo de VM adora isso).
- **Scheduling**: use `nodeSelector` para pin em nós com hardware específico (GPU passthrough, SR-IOV).
- Observe `ksmtuning` e `cpuManager: static` para performance.

### References

- https://kubevirt.io/user-guide/
- https://github.com/kubevirt/kubevirt

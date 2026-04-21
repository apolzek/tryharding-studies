# Kernel Development — Teoria e Prática

Guia para entender **onde** você pode injetar código/observação no kernel
Linux, **como** o kernel expõe suas interfaces para userspace, e **como**
o eBPF se encaixa nesse ecossistema. Os exemplos deste repo (`01-..08-`)
todos são aplicações concretas dos conceitos aqui.

---

## 1. Visão geral — as duas metades do sistema

```
┌──────────────────────────────────────── userspace (ring 3) ──────────────────────
│  bash, nginx, python, curl, ...
│         │  int 0x80 / syscall / sysenter       ioctl()  read()  write()
│         ▼
├──────────────────────────────────────── kernel (ring 0) ─────────────────────────
│  syscall table → VFS → drivers → netstack → scheduler → memory mm → block layer
│                    ▲
│                    │ ganchos onde eBPF pluga:
│                    │  kprobe  kretprobe  tracepoint  fentry/fexit
│                    │  XDP     tc-bpf     netfilter-bpf  sockops  cgroup-bpf
│                    │  uprobe (na verdade dispara em userspace)
│                    │  LSM hooks (bpf-lsm)
│                    │  perf events (sampling)
└──────────────────────────────────────── hardware ────────────────────────────────
```

Kernel development = qualquer atividade que cruze essa fronteira:
- escrever **drivers** (bloco, char, rede, USB…)
- escrever **módulos** (carregáveis via `insmod`/`modprobe`)
- modificar subsistemas (scheduler, mm, VFS)
- escrever **programas eBPF** (quase sempre a maneira *correta* hoje)

---

## 2. Interfaces principais kernel ⇄ userspace

### 2.1 Syscalls

Contrato estável mais antigo. A tabela em `arch/x86/entry/syscalls/syscall_64.tbl`
mapeia número → função (`__x64_sys_openat`). Para investigar:

```bash
ausyscall --dump | head        # nome → número
cat /proc/kallsyms | grep __x64_sys_openat
```

Cada syscall vira um **tracepoint estável**:
`tracepoint:syscalls:sys_enter_<name>` e `sys_exit_<name>` — é o que os
exemplos 07, 10, 11, 12, 40, 41, 42, 44 usam.

### 2.2 /proc (procfs)

Filesystem virtual que expõe estado do kernel como texto:
- `/proc/<pid>/{status,maps,fd/}` — info por processo.
- `/proc/kallsyms` — todos os símbolos do kernel (útil para achar alvos
  de kprobe).
- `/proc/net/tcp`, `/proc/meminfo`, `/proc/interrupts`.

Drivers podem criar entradas com `proc_create()`.

### 2.3 /sys (sysfs)

Um arquivo por atributo de objeto do kernel (device, module, bus).
Convenção: **leitura mostra estado**, **escrita altera**. Ex:

```bash
echo 0 | sudo tee /sys/class/net/eth0/carrier     # baixa link (não faça)
cat /sys/kernel/mm/transparent_hugepage/enabled   # [always] madvise never
```

### 2.4 Netlink (AF_NETLINK)

Socket bidirecional para configuração: `ip` usa rtnetlink, `ethtool` usa
GENL, iptables nftables usa nfnetlink. Alternativa moderna ao ioctl.

### 2.5 ioctl(2)

“Syscall curinga” — cada driver define sua própria linguagem. Usado
para: controlar terminal (`TIOCGWINSZ`), PTY, bloco (`BLKGETSIZE`),
tun/tap. Defeito: não autodocumentado → errno soluções obscuras.

### 2.6 mmap(2) + character devices

Compartilhamento direto de memória: `/dev/mem`, `/dev/kvm` ou memórias
de device drivers. O kernel reserva páginas físicas e mapeia no espaço
do processo — latência zero de cópia.

### 2.7 Tracefs / debugfs

- `/sys/kernel/tracing/` (tracefs) — `ftrace`, tracepoints.
- `/sys/kernel/debug/` (debugfs) — endpoints de depuração (inclui
  `/sys/kernel/debug/tracing/trace_pipe` que os exemplos BCC usam via
  `bpf_trace_printk`).

### 2.8 eventfd / signalfd / timerfd / io_uring

Mecanismos modernos que integram eventos ao `epoll`. `io_uring`
especialmente é o substituto de `aio` — duas ring buffers
(submission/completion) compartilhadas com o kernel.

### 2.9 BPF(2)

Syscall única (`bpf(cmd, attr, size)`) que centraliza todas as operações
de eBPF: criar map, carregar programa, anexar a um hook, query, pin em
bpffs. `strace -e bpf …` mostra essas chamadas.

---

## 3. O que dá pra fazer (mapa de capacidades)

| Eu quero … | Interface | Exemplo/Exemplo no repo |
|---|---|---|
| Ver **cada syscall** do sistema | tracepoint raw_syscalls | 12, 54 |
| Contar I/O por processo | kprobe `vfs_read`/`vfs_write` | 14, 55 |
| Traçar **latência** ponta a ponta de X | kprobe + kretprobe + BPF_HASH | 13, 16, 48 |
| Filtrar/descartar pacotes **antes** da stack | XDP (+ driver offload se suportar) | 30, 31, 53 |
| Modelar QoS (classificar ingress/egress) | tc-bpf clsact | (ampliar 30) |
| Observar TCP (connect, rtt, retrans) | kprobes em tcp_* / tracepoints sock | 08, 25, 26, 27 |
| Interceptar **connect() e redirecionar** | cgroup-bpf (`cgroup_sock_addr`) | Cilium faz isso |
| Captar TLS plaintext | uprobe em `libssl.SSL_write/read` | 46 |
| Seguir **funções Python/Java/Node** | USDT probes | 47 |
| Profiling on-CPU / off-CPU | perf_event sampler + stack trace | 24, 36, 49 |
| Auditar segurança (LSM) | bpf-lsm (kernel ≥ 5.7) | ver §7 |
| Criar driver novo de hardware | módulo `.ko` | ver §5 |
| Implementar um filesystem | módulo VFS ou FUSE | FUSE = userspace |
| Passar dados pro userspace com alta vazão | BPF_RINGBUF | 50, minimal.c |
| Rodar **código arbitrário** no kernel | módulo (não eBPF!) | raramente correto |

Regra prática moderna: **se existe um hook eBPF para o que você quer,
não escreva um módulo** — eBPF é verificado, hot-pluggable, seguro e
portável via CO-RE.

---

## 4. Hooks eBPF em detalhe (o que observar, onde)

| Hook | Disparo | Arg | Útil para |
|---|---|---|---|
| `kprobe/<fn>` | entrada de função do kernel | `pt_regs` | qualquer função não-inlined |
| `kretprobe/<fn>` | retorno | `pt_regs` (RC) | medir latência; ler retorno |
| `tracepoint/<grp>/<name>` | ponto estático anotado | struct gerada | **preferir** sobre kprobe |
| `fentry/fexit` (BTF) | equivalentes ao kprobe mais rápidos, args tipados | func args | substituir kprobe em kernels novos |
| `xdp` | pacote chegando na NIC | `xdp_md` | firewall L2-L4, DDoS drop |
| `tc` (clsact) | ingress/egress | `__sk_buff` | QoS, NAT, redirect |
| `cgroup_sock{,_addr}` | `socket`/`connect`/`bind` | sockops | policy por cgroup (K8s) |
| `sock_ops` | eventos TCP | socket | congestion control |
| `sk_msg` | dados enviados via sendmsg | msg | sockmap (redirect sem copy) |
| `perf_event` | amostragem periódica ou evento PMU | ctx | profiler |
| `uprobe/<bin>:<sym>` | função em userspace | `pt_regs` | TLS, libs, interpretadores |
| `USDT` | probe estática anotada no binário | args nomeados | Python/Node/OpenJDK |
| `lsm/<hook>` | decisão de segurança | hook-specific | **bloquear** (retornar -EACCES) |
| `iter/...` | iteradores | obj | dump de estado de maps/procs |
| `struct_ops` | implementar interface do kernel em BPF | — | TCP CC custom |

### Diferença kprobe vs tracepoint

- **kprobe**: atacha em *qualquer* símbolo exportado. Quebra se a função
  for inlinada ou renomeada entre kernels.
- **tracepoint**: é uma âncora estável declarada no kernel com
  `TRACE_EVENT`. Os campos de args são garantidos por versão.

**Sempre prefira tracepoint** quando existir.

---

## 5. Módulo de kernel (o jeito "clássico")

eBPF resolve 90% dos casos. Os outros 10% ainda pedem módulo: **drivers
de hardware**, filesystem de alto throughput, subsistema novo.

Esqueleto mínimo (não compilar aqui — é didático):

```c
// hello_mod.c
#include <linux/init.h>
#include <linux/module.h>
#include <linux/kernel.h>

static int __init hello_init(void) {
    pr_info("hello from kernel\n");
    return 0;
}
static void __exit hello_exit(void) {
    pr_info("bye\n");
}

module_init(hello_init);
module_exit(hello_exit);
MODULE_LICENSE("GPL");
MODULE_AUTHOR("you");
MODULE_DESCRIPTION("hello");
```

Makefile:

```make
obj-m += hello_mod.o
KDIR := /lib/modules/$(shell uname -r)/build
all:
	$(MAKE) -C $(KDIR) M=$(PWD) modules
clean:
	$(MAKE) -C $(KDIR) M=$(PWD) clean
```

Build/load/unload:

```bash
make
sudo insmod ./hello_mod.ko
dmesg | tail -5
sudo rmmod hello_mod
```

### Custo vs. eBPF
- Módulo roda com **todos** os privilégios do kernel — bug = oops/panic.
- Precisa de build por kernel-version (a menos que use DKMS).
- Nenhum verifier, nenhuma sandbox.

---

## 6. Como o **verifier** funciona (por que eBPF é seguro)

O verificador roda antes do programa ser aceito. Ele garante:

1. **Sem loops não limitados** (existem `bpf_loop`/`for` bounded hoje).
2. **Toda leitura de ponteiro é checada** — `data`, `data_end`, `skb`
   têm intervalos conhecidos.
3. **Stack ≤ 512 bytes**.
4. **Máximo 1M instruções** executadas (kernel ≥ 5.2).
5. **Helpers permitidos dependem do tipo de programa** — XDP não pode
   chamar `bpf_get_current_pid_tgid()` (não tem contexto de processo).
6. **Tipos coerentes**: você não consegue tratar um ponteiro como int.

Se recusado, você vê `failed to load program: Invalid argument` — rode
com `bpftool prog loadall ... -d` ou veja o log de verifier ativando
`BPF_PROG_LOAD` com `log_buf`.

---

## 7. bpf-lsm — usando eBPF para **bloquear** ações

Kernel ≥ 5.7 com `CONFIG_BPF_LSM=y` e `lsm=...,bpf` no cmdline permite
anexar programas em hooks do LSM (`file_open`, `bprm_check`, `socket_connect`,
…). Retornar != 0 **nega** a operação.

Aplicação típica: “só binários assinados podem exec”, “nenhum processo
deste cgroup pode abrir /etc/shadow”. É a base de **Tetragon** (Isovalent)
e parcialmente do **KubeArmor**.

Exemplo de protótipo:

```c
SEC("lsm/file_open")
int BPF_PROG(deny_shadow, struct file *file, int ret) {
    char name[32] = {};
    bpf_probe_read_kernel_str(&name, sizeof(name), file->f_path.dentry->d_iname);
    if (__builtin_memcmp(name, "shadow", 6) == 0) return -EPERM;
    return ret;  /* preserve chain */
}
```

---

## 8. Prática: como atacar um problema real

Fluxo quando você vai depurar “latência estranha em tal fluxo”:

1. **Identifique a syscall envolvida** (`strace -c <cmd>`).
2. **Meça latência** dela (exemplo 13, 48 deste repo com o nome da
   função).
3. **Descobra em que camada está** — `offcputime` (36) para sleep,
   `biolatency` (33) para disco, `runqlat` (35) para CPU.
4. **Pegue stack traces** nos picos (49).
5. Se for rede: `tcp_rtt` (27), `tcp_retransmit` (26),
   `tcp_lifecycle` (25).
6. Se for TLS: uprobe em libssl (46).
7. Com hipótese formada, quantifique com bpftrace one-liner (10-bpftrace/).

Roteiro padrão dos engenheiros de performance é *esse*: **observe com
eBPF antes de mudar qualquer código**.

---

## 9. Onde ler o código do kernel (com ênfase em BPF)

- `kernel/bpf/verifier.c`              → o verificador
- `kernel/bpf/core.c`                  → interpretador + JIT hooks
- `kernel/bpf/helpers.c`               → helpers genéricos
- `net/core/filter.c`                  → helpers e progs de rede
- `include/uapi/linux/bpf.h`           → ABI (enum bpf_cmd, bpf_prog_type)
- `include/linux/bpf.h`                → tipos internos (bpf_prog, bpf_map)
- `tools/lib/bpf/`                     → libbpf
- `tools/testing/selftests/bpf/`       → **exemplos oficiais**, canônico
- `samples/bpf/`                       → exemplos históricos
- `Documentation/bpf/`                 → specs/design docs

Ferramenta rápida para navegar:
```bash
apt source linux-6.17
rg --type c "bpf_trace_run" linux-6.17  # ex: usar Grep tool, não shell rg
```

---

## 10. Caminho sugerido para virar kernel-dev com BPF

1. Roda todos os exemplos `01-..08-`, entendendo os **tipos de hook**.
2. Implementa um diff: pega o exemplo 48 e adiciona filtro por `comm`.
3. Migra para libbpf + CO-RE (`09-libbpf-c`), entende skeleton.
4. Lê `tools/testing/selftests/bpf/progs/test_core_reloc_*.c` — CO-RE na
   raiz.
5. Escreve seu próprio **tracepoint** em um módulo (veja §5).
6. Estuda `kernel/bpf/verifier.c` — por que meu programa não passa?
7. Experimenta **bpf-lsm** (§7) para bloqueio.
8. Contribui: procure `good first issue` em `bpf/bpf-next` ou responde
   dúvidas em lore.kernel.org/bpf.

---

## 11. Leitura complementar

- Brendan Gregg — *BPF Performance Tools* (livro)
- Liz Rice — *Learning eBPF* (livro + repo)
- Cilium eBPF reference guide (documentação online oficial)
- `man 2 bpf`, `man 7 bpf-helpers`
- Kernel docs: `Documentation/bpf/` no próprio source tree

# eBPF na Prática — do Zero ao Kernel

Estudo hands-on de eBPF em Ubuntu 24.04. Mais de 50 exemplos funcionais que
vão desde um "hello world" até programas XDP, uprobes em SSL/MySQL e ring
buffer. A maioria usa **BCC em Python** (mais didático e rápido de iterar);
há também exemplos em **C + libbpf** e **bpftrace**.

> Testado em: Ubuntu 24.04.4 LTS, kernel 6.17.0-20-generic, clang/LLVM 18,
> libbpf 1.3, BCC 0.29, bpftrace 0.20.

---

## 1. O que é eBPF em uma página

eBPF é uma **máquina virtual dentro do kernel Linux**. Você compila um
programa restrito (subset de C) para bytecode eBPF, o kernel verifica
estaticamente que ele termina e não acessa memória inválida, então o
anexa a um evento: syscall, função do kernel (kprobe), tracepoint, ponto
de recepção de pacote (XDP/tc), função de userspace (uprobe) etc.

Conceitos que aparecem em todo exemplo:

| Coisa | O que é |
|---|---|
| **Programa BPF** | função C compilada para bytecode, rodando no kernel |
| **Hook** | onde o programa é anexado (kprobe, tracepoint, XDP, …) |
| **Map** | estrutura key/value compartilhada kernel ⇄ userspace |
| **Helper** | função que o kernel expõe pro BPF (`bpf_get_current_pid_tgid`, …) |
| **Verifier** | valida o bytecode; recusa loops infinitos, reads inválidos |
| **BTF** | metadados de tipos do kernel — habilita CO-RE |
| **CO-RE** | "Compile Once, Run Everywhere" — 1 binário em vários kernels |

---

## 2. Instalação do toolkit (Ubuntu 24.04)

Os únicos pacotes **obrigatórios** já estão no sistema (`bpfcc-tools`,
`python3-bpfcc`, `bpftool`, `bpftrace`, `libbpf1`, `linux-headers-*`).
Para compilar C/libbpf também instale:

```bash
sudo apt update
sudo apt install -y clang llvm libbpf-dev libelf-dev zlib1g-dev make \
                    linux-tools-common linux-tools-$(uname -r) \
                    python3-bpfcc bpfcc-tools bpftrace
```

Verifique:

```bash
bpftool version            # deve imprimir libbpf + features
bpftrace --version
clang --version
ls /sys/kernel/btf/vmlinux # BTF presente -> CO-RE funciona
```

Nenhum exemplo roda sem **sudo**: eBPF exige `CAP_BPF` / `CAP_SYS_ADMIN`.

---

## 3. Ciclo de vida de um programa eBPF

```
 ┌──────────────┐  clang -target bpf   ┌──────────┐  bpf(2)    ┌──────────┐
 │  source .c   │ ───────────────────▶ │ bytecode │ ─────────▶ │ verifier │
 └──────────────┘                      └──────────┘            └────┬─────┘
                                                                    │ OK
                                                                    ▼
                                                             ┌─────────────┐
                                                             │ JIT (x86/..) │
                                                             └──────┬───────┘
                                                                    ▼
                                                             attach @ hook
                                                         (kprobe/tp/xdp/...)
```

Com **BCC** tudo isso acontece em runtime: você passa uma string C para
`BPF(text=...)` e a lib compila, verifica, anexa e lê o output.

---

## 4. Organização deste repo

```
00-setup/      instruções, cheatsheet
01-basics/     5 exemplos: hello, counter, maps, perf events
02-syscalls/   7 exemplos: execve, openat, connect, kill…
03-file-io/    6 exemplos: latência de open, vfs_read/write, fd leak
04-processes/  6 exemplos: exec snoop, exits, fork tree, OOM
05-network/    8 exemplos: TCP connect/rtt, UDP, DNS, XDP, tc
06-performance/6 exemplos: biolatency, runqlat, offcpu, cachestat
07-security/   6 exemplos: capable, setuid, mount, ptrace, modules
08-advanced/   6 exemplos: uprobes (bash/SSL/mysql), stacks, ringbuf
09-libbpf-c/   3 exemplos em C puro + Makefile + skeleton
10-bpftrace/   8 one-liners/scripts .bt
```

Cada arquivo tem um **cabeçalho com o que faz, como rodar e o que esperar**.

---

## 5. Roteiro sugerido de estudo

1. **01-basics/01_hello.py** → entende `BPF(text=…)` + `trace_print()`.
2. **01-basics/03_perf_event.py** → aprende `BPF_PERF_OUTPUT` (eventos).
3. **01-basics/04_hash_map.py** → `BPF_HASH` kernel↔user.
4. **02-syscalls/06_execve_snoop.py** → primeiro "bpftrace substituto".
5. **06-performance/33_biolatency.py** → histograma (`BPF_HISTOGRAM`).
6. **05-network/30_xdp_drop_icmp.py** → XDP (rápido, em kernel).
7. **08-advanced/45_bash_readline.py** → **uprobe**, entra em userspace.
8. **08-advanced/50_ringbuf_events.py** → ring buffer moderno.
9. **09-libbpf-c/** → abandona BCC, compila como binário distribuível.
10. **10-bpftrace/** → 8 one-liners para o dia a dia.

---

## 6. Testando os exemplos (Makefile)

Há um `Makefile` na raiz que testa cada exemplo com um timeout curto:
se o programa carrega, passa pelo verifier e roda até ser interrompido,
conta como **PASS**. Se morre com erro de attach/verifier/símbolo,
é **FAIL** com o tail do log anexado em `STATUS.md`.

```bash
sudo make help                   # todos os targets
sudo make test-01                # testa apenas o exemplo 01
sudo make test-basics            # categoria inteira
sudo make test-bcc               # todos os Python/BCC
sudo make test-bpftrace          # todos os .bt
sudo make test-libbpf            # build + roda os C
sudo make test-all               # tudo (~5-10 min) → STATUS.md
make status                      # mostra STATUS.md
make clean                       # limpa .test-logs/ e STATUS.md

# ajustar timeout (default 6s):
TIMEOUT=10 sudo make test-all

# destrutivo (derruba ICMP em lo por poucos seg.):
sudo make test-destructive
```

Cada arquivo `.py`/`.bt` também tem um comentário `Test:` indicando o
comando individual. O número do exemplo é o prefixo do filename.

## 7. Comandos úteis do `bpftool`

```bash
sudo bpftool prog show            # programas BPF carregados
sudo bpftool map show             # maps ativos
sudo bpftool prog dump xlated id <ID>  # bytecode "legível"
sudo bpftool prog dump jited  id <ID>  # bytecode JIT (x86)
sudo bpftool feature probe        # features suportadas no kernel
sudo bpftool btf dump file /sys/kernel/btf/vmlinux format c | less
```

---

## 8. Kernel development — por onde ler o código

- `kernel/bpf/` — verifier, helpers, core (`verifier.c`, `core.c`).
- `include/uapi/linux/bpf.h` — enum com **todos os helpers e prog types**.
- `include/linux/bpf.h` — estruturas internas.
- `net/core/filter.c` — BPF para redes (socket/XDP/tc).
- `tools/lib/bpf/` — **libbpf** (userspace loader).
- `samples/bpf/` e `tools/testing/selftests/bpf/` — exemplos oficiais.

Para escrever um helper novo, o fluxo é: definir em `bpf.h`, implementar
em `kernel/bpf/helpers.c` (ou subsistema), registrar no `bpf_base_func_proto`,
passar pelo verifier. Estudar `bpf_ktime_get_ns` é um ótimo ponto de entrada.

---

## 9. Troubleshooting rápido

| Sintoma | Causa provável |
|---|---|
| `Permission denied` ao carregar | não rodou com `sudo` |
| `invalid argument` / verifier | loop sem bound, ponteiro não validado |
| `failed to find kernel BTF` | faltou `linux-headers-$(uname -r)` |
| `can't find attach point` (uprobe) | binário sem símbolos; use `readelf -s` |
| XDP não atacha em `lo` | use `xdpgeneric` (já é o default dos exemplos) |
| Programa ataca mas não imprime | esqueceu `sudo`? ou evento não dispara — gere tráfego |

---

## 10. Licença

Código deste diretório é didático, MIT.

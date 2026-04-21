#!/usr/bin/env bash
# Roda um exemplo por $TIMEOUT segundos e classifica como PASS/FAIL.
# Uso:    sudo ./tools/test-one.sh <arquivo> [args...]
# Env:    TIMEOUT (s, default 6)  LOGDIR (default .test-logs)
#         STATUS (default STATUS.md)
set +e

file="$1"
[[ -z "$file" ]] && { echo "usage: $0 <file> [args...]" >&2; exit 2; }
shift
args=("$@")

TIMEOUT="${TIMEOUT:-6}"
LOGDIR="${LOGDIR:-.test-logs}"
STATUS="${STATUS:-STATUS.md}"

mkdir -p "$LOGDIR"
logname="$(echo "$file" | tr '/' '_').log"
log="$LOGDIR/$logname"

case "$file" in
  *.py) cmd=(python3 "$file" "${args[@]}") ;;
  *.bt) cmd=(bpftrace "$file" "${args[@]}") ;;
  *)    cmd=("./$file" "${args[@]}") ;;
esac

printf '%-55s ' "$file"

# SIGINT → programas em BCC têm `except KeyboardInterrupt: pass` → exit 0.
# --kill-after=2s: mata com SIGKILL se não sair 2s depois do SIGINT.
timeout --signal=INT --kill-after=2 "$TIMEOUT" "${cmd[@]}" > "$log" 2>&1
rc=$?

# rc 0    = saiu limpo
# rc 124  = timeout acionou (programa estava rodando ok)
# rc 130  = processo saiu com SIGINT (128+2)
# rc 137  = SIGKILL após --kill-after (rodou, mas não limpou no INT)
# outros  = falha real de attach / verifier / símbolo ausente
case $rc in
  0|124|130|137)
    printf 'PASS\n'
    [[ -n "$STATUS" ]] && printf -- '- [x] %s  PASS\n' "$file" >> "$STATUS"
    exit 0
    ;;
  *)
    tail_snippet="$(tail -n 5 "$log" | tr '\n' '|' | cut -c1-200)"
    printf 'FAIL  rc=%d\n' "$rc"
    printf '      ↳ %s\n' "$tail_snippet"
    [[ -n "$STATUS" ]] && printf -- '- [ ] %s  FAIL (rc=%d) — tail: %s\n' \
      "$file" "$rc" "$tail_snippet" >> "$STATUS"
    exit 1
    ;;
esac

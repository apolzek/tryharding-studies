#!/usr/bin/env bash
set -euo pipefail

role="${K0S_ROLE:?set K0S_ROLE=controller|worker}"
shared_dir="${K0S_SHARED_DIR:-/shared}"
token_file="${shared_dir}/worker.token"

install -d -m 0700 /var/log/k0s "$shared_dir"

# cgroup v2 delegation: move container procs into leaf cgroup so the root
# can enable subtree_control (kubelet/kubepods need domain controllers).
prep_cgroups() {
  local cg=/sys/fs/cgroup
  [ -f "$cg/cgroup.controllers" ] || return 0
  if [ ! -d "$cg/init" ]; then
    mkdir -p "$cg/init" || return 0
    while read -r pid; do
      echo "$pid" > "$cg/init/cgroup.procs" 2>/dev/null || true
    done < "$cg/cgroup.procs"
  fi
  for c in $(cat "$cg/cgroup.controllers"); do
    echo "+$c" > "$cg/cgroup.subtree_control" 2>/dev/null || true
  done
}
prep_cgroups

case "$role" in
  controller)
    echo "[entrypoint] launching k0s controller"
    k0s controller --config /etc/k0s/k0s.yaml --disable-components=metrics-server &
    k0s_pid=$!

    echo "[entrypoint] waiting for API readiness"
    for _ in $(seq 1 120); do
      kill -0 "$k0s_pid" 2>/dev/null || { echo "controller died" >&2; exit 1; }
      if k0s kubectl get --raw=/readyz >/dev/null 2>&1; then
        break
      fi
      sleep 2
    done

    echo "[entrypoint] issuing worker join token"
    k0s token create --role=worker --expiry=24h > "${token_file}.tmp"
    chmod 0600 "${token_file}.tmp"
    mv "${token_file}.tmp" "${token_file}"
    echo "[entrypoint] token written to ${token_file}"

    wait "$k0s_pid"
    ;;

  worker)
    echo "[entrypoint] waiting for worker token at ${token_file}"
    for _ in $(seq 1 150); do
      [ -s "${token_file}" ] && break
      sleep 2
    done
    [ -s "${token_file}" ] || { echo "no token after timeout" >&2; exit 1; }

    echo "[entrypoint] starting k0s worker"
    # --kubelet-extra-args only needed when kubelet runs inside a container.
    # On a real VM, remove K0S_KUBELET_EXTRA_ARGS to enable QoS cgroup mgmt.
    exec k0s worker \
      --token-file "${token_file}" \
      --kubelet-extra-args="${K0S_KUBELET_EXTRA_ARGS:---cgroups-per-qos=false --enforce-node-allocatable= --resolv-conf=/etc/resolv.conf.k0s}"
    ;;

  *)
    echo "unknown K0S_ROLE: $role" >&2
    exit 1
    ;;
esac

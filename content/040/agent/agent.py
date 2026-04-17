#!/usr/bin/env python3
"""
User-activity eBPF agent.

Runs as root, loads a small BPF program that increments kernel-side counters
for keyboard, mouse, scroll and TCP bytes. Every FLUSH_INTERVAL seconds the
counters are read, reset, enriched with the currently-active login session,
host metadata, and per-process CPU stats, and POSTed to the backend.

Kernel-side map updates are atomic and constant-time, so steady-state overhead
is negligible even on loaded machines.
"""
from __future__ import annotations

import argparse
import json
import logging
import os
import pwd
import socket
import subprocess
import sys
import time
import urllib.error
import urllib.request
from dataclasses import asdict, dataclass, field
from pathlib import Path
from typing import Optional

try:
    from bcc import BPF  # type: ignore
except ImportError:
    print("ERROR: python3-bpfcc is not installed. Run:", file=sys.stderr)
    print("  sudo apt-get install -y bpfcc-tools python3-bpfcc linux-headers-$(uname -r)",
          file=sys.stderr)
    sys.exit(1)

LOG = logging.getLogger("ebpf-agent")

AGG_KEYS, AGG_CLICKS, AGG_MOVES, AGG_SCROLL, AGG_KEY_REL, \
    AGG_RX_BYTES, AGG_TX_BYTES, AGG_NET_CALLS = range(8)

AGENT_VERSION = "1.0.0"


@dataclass
class Sample:
    """Single flush sample sent to the backend."""
    agent_version: str
    host: str
    machine_id: str
    kernel: str
    os: str
    session_user: str
    session_uid: int
    window_start: float  # unix ts
    window_end: float
    interval_sec: float
    # input
    keys_pressed: int
    keys_released: int
    clicks: int
    mouse_moves: int
    scrolls: int
    # network (global)
    rx_bytes: int
    tx_bytes: int
    net_calls: int
    # per-uid network
    per_user_net: list = field(default_factory=list)
    # system sample
    load1: float = 0.0
    load5: float = 0.0
    load15: float = 0.0
    mem_used_pct: float = 0.0
    cpu_used_pct: float = 0.0
    num_procs: int = 0


def read_machine_id() -> str:
    for p in ("/etc/machine-id", "/var/lib/dbus/machine-id"):
        try:
            return Path(p).read_text().strip()
        except OSError:
            continue
    return "unknown"


def read_os_release() -> str:
    try:
        data = {}
        for line in Path("/etc/os-release").read_text().splitlines():
            if "=" in line:
                k, v = line.split("=", 1)
                data[k] = v.strip('"')
        return f"{data.get('NAME','')} {data.get('VERSION_ID','')}".strip()
    except OSError:
        return "unknown"


def detect_active_session() -> tuple[str, int]:
    """
    Find the currently-active graphical/console login session. Falls back to
    the first logged-in user in `who`.

    Returns (username, uid). Returns ("", -1) if no human session is active.
    """
    # Try loginctl (systemd-logind). Safer, handles multi-seat.
    try:
        out = subprocess.run(
            ["loginctl", "list-sessions", "--no-legend"],
            capture_output=True, text=True, timeout=2, check=False,
        ).stdout
        for line in out.splitlines():
            parts = line.split()
            if len(parts) < 3:
                continue
            sid, uid_s, user = parts[0], parts[1], parts[2]
            # Inspect session state; pick first active one
            sinfo = subprocess.run(
                ["loginctl", "show-session", sid, "-p", "State",
                 "-p", "Remote", "-p", "Type"],
                capture_output=True, text=True, timeout=2, check=False,
            ).stdout
            kv = dict(l.split("=", 1) for l in sinfo.splitlines() if "=" in l)
            if kv.get("State") == "active" and kv.get("Remote") == "no":
                try:
                    return user, int(uid_s)
                except ValueError:
                    pass
    except (FileNotFoundError, subprocess.TimeoutExpired):
        pass

    # Fallback: `who` (first user on a graphical tty / :0)
    try:
        out = subprocess.run(["who"], capture_output=True, text=True,
                             timeout=2, check=False).stdout
        for line in out.splitlines():
            parts = line.split()
            if not parts:
                continue
            user = parts[0]
            try:
                return user, pwd.getpwnam(user).pw_uid
            except KeyError:
                continue
    except FileNotFoundError:
        pass

    return "", -1


def read_proc_loadavg() -> tuple[float, float, float, int]:
    try:
        parts = Path("/proc/loadavg").read_text().split()
        l1, l5, l15 = float(parts[0]), float(parts[1]), float(parts[2])
        procs = int(parts[3].split("/")[1])
        return l1, l5, l15, procs
    except (OSError, ValueError, IndexError):
        return 0.0, 0.0, 0.0, 0


def read_mem_used_pct() -> float:
    try:
        info = {}
        for line in Path("/proc/meminfo").read_text().splitlines():
            k, _, v = line.partition(":")
            info[k] = int(v.strip().split()[0])
        total = info.get("MemTotal", 0)
        available = info.get("MemAvailable", total)
        if total <= 0:
            return 0.0
        return 100.0 * (total - available) / total
    except (OSError, ValueError):
        return 0.0


class CPUPoller:
    """Cheap CPU% sampler using /proc/stat deltas."""

    def __init__(self) -> None:
        self._prev_total = 0
        self._prev_idle = 0

    def _snapshot(self) -> tuple[int, int]:
        with open("/proc/stat", "r") as f:
            line = f.readline()
        fields = [int(x) for x in line.split()[1:]]
        idle = fields[3] + (fields[4] if len(fields) > 4 else 0)
        total = sum(fields)
        return total, idle

    def sample(self) -> float:
        try:
            total, idle = self._snapshot()
            dt = total - self._prev_total
            di = idle - self._prev_idle
            self._prev_total, self._prev_idle = total, idle
            if dt <= 0:
                return 0.0
            return 100.0 * (dt - di) / dt
        except OSError:
            return 0.0


class Agent:
    def __init__(self, backend_url: str, interval: float, token: str,
                 dry_run: bool) -> None:
        self.backend_url = backend_url.rstrip("/") + "/api/v1/ingest"
        self.interval = interval
        self.token = token
        self.dry_run = dry_run
        self.host = socket.gethostname()
        self.machine_id = read_machine_id()
        self.kernel = os.uname().release
        self.os_rel = read_os_release()
        self.cpu = CPUPoller()
        self.cpu.sample()  # prime

        src = Path(__file__).with_name("bpf_program.c").read_text()
        LOG.info("compiling and loading BPF program…")
        self.bpf = BPF(text=src)
        LOG.info("BPF program attached")

    def _read_and_reset_agg(self) -> list[int]:
        values = []
        for i in range(8):
            k = self.bpf["agg"].Key(i)
            try:
                leaf = self.bpf["agg"][k]
                v = leaf.value
            except KeyError:
                v = 0
            values.append(int(v))
            # reset
            self.bpf["agg"][k] = self.bpf["agg"].Leaf(0)
        return values

    def _read_and_reset_per_uid(self) -> list[dict]:
        m = self.bpf["net_by_uid"]
        rows = []
        keys_to_delete = []
        for k, v in m.items():
            uid = int(k.value)
            try:
                name = pwd.getpwuid(uid).pw_name
            except KeyError:
                name = f"uid:{uid}"
            rows.append({
                "uid": uid,
                "user": name,
                "rx_bytes": int(v.rx_bytes),
                "tx_bytes": int(v.tx_bytes),
                "rx_calls": int(v.rx_calls),
                "tx_calls": int(v.tx_calls),
            })
            keys_to_delete.append(k)
        for k in keys_to_delete:
            try:
                del m[k]
            except KeyError:
                pass
        return rows

    def flush_once(self) -> Sample:
        t1 = time.time()
        values = self._read_and_reset_agg()
        per_uid = self._read_and_reset_per_uid()
        user, uid = detect_active_session()
        l1, l5, l15, procs = read_proc_loadavg()

        sample = Sample(
            agent_version=AGENT_VERSION,
            host=self.host,
            machine_id=self.machine_id,
            kernel=self.kernel,
            os=self.os_rel,
            session_user=user,
            session_uid=uid,
            window_start=t1 - self.interval,
            window_end=t1,
            interval_sec=self.interval,
            keys_pressed=values[AGG_KEYS],
            keys_released=values[AGG_KEY_REL],
            clicks=values[AGG_CLICKS],
            mouse_moves=values[AGG_MOVES],
            scrolls=values[AGG_SCROLL],
            rx_bytes=values[AGG_RX_BYTES],
            tx_bytes=values[AGG_TX_BYTES],
            net_calls=values[AGG_NET_CALLS],
            per_user_net=per_uid,
            load1=l1, load5=l5, load15=l15,
            mem_used_pct=round(read_mem_used_pct(), 2),
            cpu_used_pct=round(self.cpu.sample(), 2),
            num_procs=procs,
        )
        return sample

    def send(self, s: Sample) -> None:
        payload = json.dumps(asdict(s)).encode()
        if self.dry_run:
            LOG.info("DRY-RUN sample: %s", payload.decode())
            return
        req = urllib.request.Request(
            self.backend_url, data=payload,
            headers={
                "Content-Type": "application/json",
                "Authorization": f"Bearer {self.token}",
                "User-Agent": f"userwatch-agent/{AGENT_VERSION}",
            },
            method="POST",
        )
        try:
            with urllib.request.urlopen(req, timeout=5) as resp:
                if resp.status >= 300:
                    LOG.warning("backend returned %d", resp.status)
        except urllib.error.URLError as e:
            LOG.warning("failed to POST to backend: %s", e)

    def run(self) -> None:
        LOG.info("entering flush loop; interval=%.1fs backend=%s",
                 self.interval, self.backend_url)
        while True:
            try:
                time.sleep(self.interval)
                sample = self.flush_once()
                LOG.debug("sample: keys=%d clicks=%d moves=%d tx=%d rx=%d user=%s",
                          sample.keys_pressed, sample.clicks, sample.mouse_moves,
                          sample.tx_bytes, sample.rx_bytes, sample.session_user)
                self.send(sample)
            except KeyboardInterrupt:
                LOG.info("shutting down")
                return
            except Exception as e:  # defensive: never let the loop die
                LOG.exception("flush loop error: %s", e)


def main() -> int:
    p = argparse.ArgumentParser(description="User-activity eBPF agent")
    p.add_argument("--backend", default=os.environ.get("UW_BACKEND",
                   "http://127.0.0.1:8080"),
                   help="Backend base URL (default: %(default)s)")
    p.add_argument("--interval", type=float,
                   default=float(os.environ.get("UW_INTERVAL", "10")),
                   help="Flush interval in seconds (default: %(default)s)")
    p.add_argument("--token", default=os.environ.get("UW_TOKEN", "dev-token"),
                   help="Bearer token for backend auth")
    p.add_argument("--dry-run", action="store_true",
                   help="Print samples instead of POSTing")
    p.add_argument("-v", "--verbose", action="count", default=0)
    args = p.parse_args()

    logging.basicConfig(
        level=logging.DEBUG if args.verbose else logging.INFO,
        format="%(asctime)s %(levelname)s %(name)s: %(message)s",
    )

    if os.geteuid() != 0:
        LOG.error("must run as root to load BPF programs")
        return 1

    agent = Agent(args.backend, args.interval, args.token, args.dry_run)
    agent.run()
    return 0


if __name__ == "__main__":
    sys.exit(main())

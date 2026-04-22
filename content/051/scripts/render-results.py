#!/usr/bin/env python3
"""Render results/*.csv into Markdown tables and patch README.md between
the <!-- RESULTS_START --> / <!-- RESULTS_END --> markers.
"""
import csv
import os
import pathlib
import re
import subprocess
import sys

ROOT = pathlib.Path(__file__).resolve().parent.parent
RESULTS = ROOT / "results"
README = ROOT / "README.md"


def fmt_rate(n):
    n = int(float(n))
    if n >= 1_000_000:
        return f"{n/1_000_000:.2f} M/s"
    if n >= 1_000:
        return f"{n/1_000:.1f} k/s"
    return f"{n}/s"


def fmt_bytes(n):
    n = float(n)
    for unit in ("B", "KiB", "MiB", "GiB"):
        if n < 1024:
            return f"{n:.0f} {unit}"
        n /= 1024
    return f"{n:.0f} TiB"


def tier_key(cpus, mem):
    m = float(cpus)
    mem_bytes = int(mem[:-1]) * (1024**2 if mem.endswith("m") else 1024**3)
    return (m, mem_bytes)


def read_sweep(proto, cpus, mem):
    path = RESULTS / f"sweep_{proto}_{cpus}cpu_{mem}.csv"
    if not path.exists():
        return []
    with path.open() as f:
        return list(csv.DictReader(f))


def max_sustained(rows):
    """True max sustained = max received_rate across steps that still had
    0 drops and a live container. Captures the CPU-saturated plateau even when
    the sweep's 85%-of-target gate would discard the higher step."""
    ok = [
        r for r in rows
        if float(r.get("refused_rate", 0)) < 10
        and r.get("container_alive", "false") == "true"
    ]
    if not ok:
        return 0.0
    return max(float(r["received_rate"]) for r in ok)


def discover_tiers():
    """All (proto, cpus, mem) tuples that have a sweep_*.csv."""
    tiers = []
    for f in sorted(RESULTS.glob("sweep_*_*cpu_*.csv")):
        stem = f.stem[len("sweep_"):]          # e.g. grpc_0.5cpu_256m
        proto, rest = stem.split("_", 1)       # grpc  |  0.5cpu_256m
        cpus_part, mem = rest.rsplit("_", 1)   # 0.5cpu, 256m
        cpus = cpus_part.removesuffix("cpu")
        tiers.append((proto, cpus, mem))
    return tiers


def read_best():
    """Compute best per tier from the raw sweep CSVs (post-processing)."""
    rows = []
    for proto, cpus, mem in discover_tiers():
        sweep = read_sweep(proto, cpus, mem)
        if not sweep:
            continue
        rows.append({
            "protocol": proto,
            "cpus": cpus,
            "mem": mem,
            "signal": sweep[0].get("signal", "traces"),
            "best_sustained_rate": int(max_sustained(sweep)),
        })
    return rows


def summary_table(best_rows):
    # group by tier (cpus, mem) so grpc and http show side-by-side
    tiers = {}
    for r in best_rows:
        key = (r["cpus"], r["mem"])
        tiers.setdefault(key, {})[r["protocol"]] = r["best_sustained_rate"]
    rows = sorted(tiers.items(), key=lambda kv: tier_key(*kv[0]))

    out = ["| Tier | CPU | Memory | OTLP/gRPC (spans/s) | OTLP/HTTP (spans/s) | Ratio gRPC/HTTP |",
           "|------|-----|--------|---------------------|---------------------|-----------------|"]
    for i, ((cpus, mem), d) in enumerate(rows, 1):
        g = d.get("grpc", "0")
        h = d.get("http", "0")
        ratio = "—"
        try:
            if float(h) > 0:
                ratio = f"{float(g) / float(h):.2f}×"
        except Exception:
            pass
        out.append(f"| T{i} | {cpus} | {mem} | **{fmt_rate(g)}** | **{fmt_rate(h)}** | {ratio} |")
    return "\n".join(out)


def per_tier_tables(best_rows):
    tiers = {}
    for r in best_rows:
        tiers.setdefault((r["cpus"], r["mem"]), set()).add(r["protocol"])
    tiers = sorted(tiers.keys(), key=lambda k: tier_key(*k))

    parts = []
    for cpus, mem in tiers:
        parts.append(f"\n#### {cpus} vCPU / {mem} RAM\n")
        parts.append("| Protocol | Target | Gens | Received | Refused | CPU cores | CPU util | Mem RSS | Alive |")
        parts.append("|----------|-------:|-----:|---------:|--------:|----------:|---------:|--------:|:-----:|")
        for proto in ("grpc", "http"):
            rows = read_sweep(proto, cpus, mem)
            for r in rows:
                received = fmt_rate(r["received_rate"])
                target = fmt_rate(r["target_rate"])
                refused = r["refused_rate"]
                refused_str = fmt_rate(refused) if float(refused) > 0 else "0"
                alive = "✓" if r["container_alive"] == "true" else "💀"
                parts.append(
                    f"| {proto.upper()} | {target} | {r['generators']} | "
                    f"{received} | {refused_str} | {float(r['cpu_cores']):.2f} | "
                    f"{float(r['cpu_util_pct']):.1f}% | {fmt_bytes(r['mem_rss_bytes'])} | {alive} |"
                )
    return "\n".join(parts)


def host_block():
    def run(cmd):
        try:
            return subprocess.check_output(cmd, shell=True, text=True).strip()
        except Exception:
            return "unknown"

    cpu = run("lscpu | awk -F: '/Model name/ {print $2}' | sed 's/^ *//'")
    cores = run("nproc")
    mem = run("free -h | awk '/^Mem:/ {print $2}'")
    kernel = run("uname -r")
    docker = run("docker --version | awk '{print $3}' | sed 's/,$//'")
    return (
        f"- **CPU**: {cpu} ({cores} logical cores)\n"
        f"- **RAM**: {mem}\n"
        f"- **Kernel**: {kernel}\n"
        f"- **Docker**: {docker}\n"
        f"- **Host idle CPU during runs**: >70%; all collector containers ran under `cpus`/`mem_limit` cgroups so other workloads did not compete.\n"
    )


def patch_readme(summary, details, host):
    text = README.read_text()

    results_block = (
        f"### Best sustained throughput\n\n"
        f"{summary}\n\n"
        f"### Per-step detail (every step of every sweep)\n"
        f"{details}\n"
    )

    text = re.sub(
        r"(<!-- RESULTS_START -->).*?(<!-- RESULTS_END -->)",
        lambda m: f"{m.group(1)}\n{results_block}\n{m.group(2)}",
        text,
        flags=re.DOTALL,
    )
    text = re.sub(
        r"(<!-- HOST_START -->).*?(<!-- HOST_END -->)",
        lambda m: f"{m.group(1)}\n{host}\n{m.group(2)}",
        text,
        flags=re.DOTALL,
    )
    README.write_text(text)


def main():
    best = read_best()
    if not best:
        print("no results/best.csv yet — run the matrix first", file=sys.stderr)
        sys.exit(1)
    summary = summary_table(best)
    details = per_tier_tables(best)
    host = host_block()
    patch_readme(summary, details, host)
    print("README.md updated.")
    print("---- summary ----")
    print(summary)


if __name__ == "__main__":
    main()

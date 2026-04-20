#!/usr/bin/env python3
"""Generate the root README.md from frontmatter in content/NNN/README.md files.

Walks every numeric folder under content/, reads the YAML frontmatter from its
README.md (title, tags, status) and rewrites the repo-root README.md with a
single descending-order index.

Dependencies: stdlib only (no PyYAML) — the parser is intentionally tolerant of
the tiny subset of YAML we use (string, bracketed list, quoted strings).

Run:
    python3 scripts/gen_index.py
"""
from __future__ import annotations

import re
import sys
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[1]
CONTENT_DIR = REPO_ROOT / "content"
OUT = REPO_ROOT / "README.md"


def parse_frontmatter(text: str) -> dict[str, object]:
    """Extract a simple YAML frontmatter block.

    Supports:
        key: value
        key: "value with : colon"
        key: [a, b, c]
    """
    if not text.startswith("---"):
        return {}
    end = text.find("\n---", 3)
    if end == -1:
        return {}
    block = text[3:end].strip()
    data: dict[str, object] = {}
    for raw in block.splitlines():
        line = raw.strip()
        if not line or line.startswith("#"):
            continue
        m = re.match(r"([A-Za-z_][\w-]*)\s*:\s*(.*)$", line)
        if not m:
            continue
        key, val = m.group(1), m.group(2).strip()
        if val.startswith("[") and val.endswith("]"):
            inner = val[1:-1].strip()
            data[key] = [s.strip().strip('"').strip("'") for s in inner.split(",") if s.strip()] if inner else []
        else:
            if (val.startswith('"') and val.endswith('"')) or (val.startswith("'") and val.endswith("'")):
                val = val[1:-1]
            data[key] = val
    return data


def collect() -> list[tuple[str, dict]]:
    entries: list[tuple[str, dict]] = []
    for child in sorted(CONTENT_DIR.iterdir()):
        if not child.is_dir():
            continue
        if not child.name.isdigit():
            continue
        readme = child / "README.md"
        if not readme.is_file():
            continue
        fm = parse_frontmatter(readme.read_text(encoding="utf-8"))
        entries.append((child.name, fm))
    entries.sort(key=lambda x: x[0], reverse=True)
    return entries


def render(entries: list[tuple[str, dict]]) -> str:
    lines: list[str] = []
    lines.append("# tryharding-studies")
    lines.append("")
    lines.append(
        "Numbered collection of proofs of concept (PoCs) under `content/NNN/`. "
        "Each folder is independent and self-contained, with its own `README.md` "
        "following the template in `content/README.template`:"
    )
    lines.append("")
    lines.append("- **Objectives**: what the PoC tries to demonstrate")
    lines.append("- **Prerequisites**: what needs to be installed")
    lines.append("- **Reproducing**: exact commands to run the PoC")
    lines.append("- **Results**: what you learn / what's observable")
    lines.append("- **References**: useful links")
    lines.append("")
    lines.append(
        "Every PoC carries YAML frontmatter (`title`, `tags`, `status`). "
        "This root index is generated from that metadata by `scripts/gen_index.py`. "
        "Do not hand-edit."
    )
    lines.append("")
    lines.append("## Index")
    lines.append("")

    existing = {num for num, _ in entries}
    all_nums = range(max(int(n) for n in existing), 0, -1)

    for num in all_nums:
        key = f"{num:03d}"
        if key in existing:
            fm = dict(entries[[n for n, _ in entries].index(key)][1])
            title = fm.get("title") or "(untitled)"
            tags = fm.get("tags") or []
            status = fm.get("status") or "stable"
            tags_str = " ".join(f"`{t}`" for t in tags) if tags else ""
            status_badge = "" if status == "stable" else f" _({status})_"
            lines.append(f'- <a href="content/{key}">{key}</a> {title}{status_badge}  ')
            if tags_str:
                lines.append(f"  {tags_str}")
        else:
            lines.append(f"- {key} _missing_")

    lines.append("")
    lines.append("---")
    lines.append("")
    lines.append(
        "<!-- Regenerate with: `python3 scripts/gen_index.py` -->"
    )
    lines.append(
        "<!-- find . -type f -size +10M | grep -v \".git\" | sed 's|^\\./||' >> .gitignore -->"
    )
    return "\n".join(lines) + "\n"


def main() -> int:
    entries = collect()
    if not entries:
        print("no content/NNN/README.md files found", file=sys.stderr)
        return 1
    OUT.write_text(render(entries), encoding="utf-8")
    print(f"wrote {OUT} ({len(entries)} entries)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

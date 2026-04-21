# Prepare

Setup steps required to work on this repository. Run these once per clone.

## Philosophy

Hooks only block things that are genuinely harmful: committed secrets, unresolved merge conflicts, and accidentally added large binaries. Cosmetic checks (trailing whitespace, EOF newline, YAML/JSON syntax) are intentionally **not** enforced — they create noise on WIP POCs without catching real incidents.

## Requirements

- Python 3.12+
- `pipx` (preferred) or a local venv
- Git

## 1. Install tooling

```bash
sudo apt install -y pipx
pipx ensurepath
pipx install pre-commit
pipx install detect-secrets
```

Verify:

```bash
pre-commit --version      # >= 4.5
detect-secrets --version  # >= 1.5
```

Alternative (venv, no sudo):

```bash
python3 -m venv .venv
.venv/bin/pip install pre-commit detect-secrets
export PATH="$PWD/.venv/bin:$PATH"
```

## 2. Generate the secrets baseline

The baseline records the current state of the repo so existing, already-known strings don't trip the hook.

```bash
detect-secrets scan > .secrets.baseline
```

Commit `.secrets.baseline` to the repo.

## 3. Install the git hooks

The config at `.pre-commit-config.yaml` declares `default_install_hook_types: [pre-commit, pre-push]`, so a single install covers both stages.

```bash
pre-commit install --install-hooks
```

Expected output:

```
pre-commit installed at .git/hooks/pre-commit
pre-commit installed at .git/hooks/pre-push
```

## 4. What runs when

| Stage      | Hook                    | Blocks when                              |
|------------|-------------------------|------------------------------------------|
| pre-commit | check-merge-conflict    | Unresolved `<<<<<<<` markers in the diff |
| pre-commit | check-added-large-files | Any staged file larger than 1 MB         |
| pre-push   | detect-secrets          | New high-entropy strings, known token patterns, or secret-keyword matches not in the baseline |

Rationale: the two pre-commit checks are near-zero cost and catch real mistakes. The heavier secret scan only runs on push, so local iteration stays fast.

## 5. Manual runs

Run all hooks against every file (useful before a big push or in CI):

```bash
pre-commit run --all-files
pre-commit run --hook-stage pre-push --all-files
```

Run a single hook:

```bash
pre-commit run detect-secrets --hook-stage pre-push --all-files
```

## 6. When detect-secrets flags a false positive

Audit the finding, mark known-safe entries, and commit the updated baseline:

```bash
detect-secrets scan --baseline .secrets.baseline
detect-secrets audit .secrets.baseline
git add .secrets.baseline
```

Inline alternative — append `# pragma: allowlist secret` to the line:

```python
api_key = "not-actually-a-secret"  # pragma: allowlist secret
```

## 7. Bypass (use sparingly)

```bash
git commit --no-verify
git push --no-verify
```

Only use when you've already validated locally and the hook is misfiring — never to ship unreviewed secrets.

## 8. Keeping hooks up to date

```bash
pre-commit autoupdate
```

Bumps pinned `rev:` versions in `.pre-commit-config.yaml`. Review and commit the diff.

# HANDOVER — living session state

> **On "devam et": read this file, then do "Next up". Update this file before the session ends.**
> Keep it short and current — this is state, not history.

_Last updated: 2026-06-11 — session: P3 part 1 (pkg/scan + Code Sweeper CLI)._

## Push policy

The user pushes manually. Local commits ahead of origin: check `git rev-list --count origin/main..HEAD`.
`.github/` stays gitignored (D-026); CI source of truth = `deploy/ci/github-ci.yml`.

## ⚠️ Watch out: untracked files vanished twice this session

Newly written (uncommitted) files under `pkg/scan/` and `clients/cli/` disappeared from disk mid-session
(tracked/modified files were untouched; cause unknown — possibly a `git clean` in the user's open
terminal or AV). **Mitigation: commit early, commit often** — as soon as a new package compiles+tests,
commit it before continuing. If a build suddenly can't find a package that "was just there", check
`git status` / re-create from the last commit.

## Current state

**P0 ✓, P1 ✓, P2 ✓, P3 started.** `task ci` green: **8 Go modules** (now incl. `clients/cli`) + Python.

- **`pkg/scan` (D-027)**: THE single source for deterministic detectors (5 secret patterns w/ masking,
  3 risky patterns w/ language scoping) + `scan.Run` chain helper. Analyzer domain detectors and the
  orchestrator's `stages.SecretScan` are now thin delegates — the triplicated regex sets are gone.
- **Code Sweeper CLI** (`clients/cli`, binary `membrane`, 2.4 MB static, `task build:cli` → `bin/`):
  `membrane scan [path] [--json] [--fail-on=blocking|warning|never]` — walks a tree (skips
  .git/node_modules/vendor/…, binaries, >1 MiB), language by extension, findings sorted file:line,
  human + JSON output, stable CI exit codes (0/1/2). Two-pass flag parse (flags valid before/after
  path). Live-verified: fixture scan found AWS key (blocking, exit 1) + InsecureSkipVerify (warning);
  `--fail-on=never --json` exits 0 with full JSON.
- Taskfile gotcha fixed: task `env:` blocks do NOT override the machine-level stray `GOWORK` — set
  `GOWORK=off` inline in commands (build:cli now does).

## Next up  (P3; see docs/ROADMAP.md)

1. **PR-comment adapter** in the reporter: when `pr_number` > 0, post the findings table as a PR
   comment (`POST /repos/{owner}/{repo}/issues/{pr}/comments`), same token/config family as the
   commit-status notifier, idempotent per (submission, notifier) as usual. httptest coverage like
   `githubstatus_test.go`.
2. **vLLM tier-2 adapter** in the semantic service: OpenAI-compatible `/v1/completions` client
   (httpx, async, timeout) behind config (`MEMBRANE_SEMANTIC_VLLM_URL`, empty = heuristic stub stays);
   prompt builds from masked diff + gold context; parse findings JSON; mypy-strict.
3. Then: Code Sweeper "Technical-Debt Report" mode, or Dockerfiles/Helm for deploy (ask user if equal).

## Blockers / gotchas

- **Docker** engine flaky — start Docker Desktop manually; never block on it. Dev stack may still be
  running from earlier sessions (`docker compose … ps`).
- **Stray `GOWORK`** → inline `GOWORK=off` ONLY (task env: blocks don't beat it). **buf** = prebuilt
  exe. **`-race`** = CI-only. Python venv: `task setup:py`.
- Integration DSNs: `MEMBRANE_{ORCHESTRATOR,RESOLVER}_TEST_DSN=postgres://membrane:membrane@localhost:5432/membrane`.

## How to verify

- `task ci` → all green (8 Go modules + Python).
- CLI story: `clients/cli/internal/runner/runner_test.go` (find/sort/count, noise-dir+binary skip,
  single-file target, language scoping); live: `task build:cli` then
  `bin\membrane.exe scan <dir>` → findings + exit 1 on a planted `AKIA…` key.
- pkg/scan story: `pkg/scan/scan_test.go` (all pattern kinds, masking, language scoping, Run chain).

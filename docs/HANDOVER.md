# HANDOVER — living session state

> **On "devam et": read this file, then do "Next up". Update this file before the session ends.**
> Keep it short and current — this is state, not history.

_Last updated: 2026-06-11 — session: P3 part 4 (Code Sweeper Technical-Debt Report mode)._

## Push policy

The user pushes manually (`git push` in their own terminal). Check pending:
`git rev-list --count origin/main..HEAD`. `.github/` gitignored (D-026); CI source = `deploy/ci/github-ci.yml`
— **it gained an `images` job this session; mirror it to GitHub via the web UI when convenient.**

## ⚠️ Standing gotchas

- **Untracked files vanished twice** earlier — commit early, commit often.
- **Stray machine `GOWORK`** → inline `GOWORK=off` only (task `env:` blocks don't override it).
- **Docker** engine needs a manual Docker Desktop start sometimes; never block on it.
- **buf** = prebuilt exe; **`-race`** = CI-only; Python venv via `task setup:py`.

## Current state

**P0 ✓ P1 ✓ P2 ✓ — P3 in flight.** `task ci` green: 8 Go modules + Python (24 pytest).

- **Technical-Debt Report mode done** (the GTM lead magnet): `membrane scan --report md|html
  [--out file]` → `clients/cli/internal/report`. Aggregates by rule/dir/file, severity-weighted
  debt score + A–F grade (constants render via a single `ScoringLegend` so prose can't drift;
  small scans grade against a 10-file floor; zero-scanned = "—", never "Clean ✓"). Detail section
  is severity-first before its 200-row cap (blockers are never the truncated rows). HTML is one
  self-contained light-toned page. Exit codes unchanged (`--fail-on` still decides).
- **Hardening from the multi-agent review of that change**: extra positional args now exit 2
  (previously silently dropped the path AND following flags — CI false-clean); single-file scans
  label findings with the file name (was "."); mdEscape neutralizes control chars (CR/ESC) and
  doubles backslashes before pipe-escaping; terminal output sanitizes ANSI/OSC; report renders to
  memory first (no truncated file on render error, close checked before the success line);
  report totals tallied from findings (single source of truth); unknown severities weigh as
  warnings, never info.

- **Containerization done (D-029)**: one parameterized `deploy/docker/Dockerfile.go` (multi-stage,
  BuildKit cache mounts, static binary → `distroless/static:nonroot`) builds all 5 Go services;
  `Dockerfile.semantic` = two-stage python-slim wheels, non-root. `task build:images` →
  `membrane/<svc>:dev`. **`task full-up`** runs `deploy/compose/docker-compose.full.yml`
  (self-contained: dual-listener Redpanda — internal `redpanda:9092` / host `localhost:19092`,
  one-shot `topics` + `migrate` containers, all 6 services wired by env). Distroless = no shell ⇒
  probe `/readyz` from the host. `.dockerignore` keeps the context lean. CI got an `images` job.
  **E2E proven all-container**: webhook 202 → verdict `rejected` (2 secrets + SQL concat +
  `local-heuristic:masked` privacy proof) consumed off `code.verdict.v1`.
- **Contract fix found by that E2E**: orchestrator sent `"gold_context": null` when RAG was skipped
  (nil slice) and semantic's pydantic rejected it with 422 — now `omitempty` on the Go side AND
  null-tolerant on the Python side, regression-tested in both (`stage_test.go`, `test_api.py`).
- Reporter: webhook + commit-status + PR-comment notifiers, idempotent (D-025/D-028).
- Semantic tier-2 real-model-ready: `VLLMLocalModel` + `FallbackLocalModel` (D-028), heuristic default.

## Next up  (P3 continuation; see docs/ROADMAP.md)

1. **Helm chart** (deploy/helm): unblocked by D-029 images; values per service, infra as
   dependencies or external endpoints.
2. **Premium tier-3 consensus adapter** (Claude Sonnet 4.6 + Gemini): needs an API-key handling
   decision (env vs file vs vault) — record as **D-030** when built; until then the flag stays off.
3. **CLI packaging matrix** (winget/brew/deb/rpm/tarball) or **accuracy/eval harness** — pick per
   GTM priority.

## How to verify

- `task ci` → all green.
- Report mode: `task build:cli` then `bin/membrane scan . --report html --out debt.html
  --fail-on=never` (repo self-scan ≈ 34 findings, all from test fixtures — a good demo). Story
  lives in `clients/cli/internal/report/report_test.go` + `cmd/membrane/main_test.go`.
- Containers: `task build:images` then `task full-up`; from the host probe
  `curl http://localhost:8101/readyz` (ingestion), `:8102`–`:8105` (orchestrator/analyzer/resolver/
  reporter), `http://localhost:8005/readyz` (semantic). Kafka from the host = `localhost:19092`.
  Tear down with `task full-down`.
- PR comment story: `services/reporter/internal/adapters/notify/githubprcomment_test.go`.
- vLLM story: `services/semantic/tests/test_vllm.py`.

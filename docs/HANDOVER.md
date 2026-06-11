# HANDOVER — living session state

> **On "devam et": read this file, then do "Next up". Update this file before the session ends.**
> Keep it short and current — this is state, not history.

_Last updated: 2026-06-11 — session: P3 part 3 (containerization, D-029)._

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

**P0 ✓ P1 ✓ P2 ✓ — P3 in flight.** `task ci` green: 8 Go modules + Python (23 pytest).

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

1. **Code Sweeper "Technical-Debt Report" mode** (`membrane scan --report md|html`): aggregate
   findings by rule/severity/dir + summary stats — the GTM lead magnet (report §GTM).
2. **Helm chart** (deploy/helm): now unblocked by D-029 images; values per service, infra as
   dependencies or external endpoints.
3. **Premium tier-3 consensus adapter** (Claude Sonnet 4.6 + Gemini): needs an API-key handling
   decision (env vs file vs vault) — record as **D-030** when built; until then the flag stays off.

## How to verify

- `task ci` → all green.
- Containers: `task build:images` then `task full-up`; from the host probe
  `curl http://localhost:8101/readyz` (ingestion), `:8102`–`:8105` (orchestrator/analyzer/resolver/
  reporter), `http://localhost:8005/readyz` (semantic). Kafka from the host = `localhost:19092`.
  Tear down with `task full-down`.
- PR comment story: `services/reporter/internal/adapters/notify/githubprcomment_test.go`.
- vLLM story: `services/semantic/tests/test_vllm.py`.

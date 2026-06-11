# HANDOVER — living session state

> **On "devam et": read this file, then do "Next up". Update this file before the session ends.**
> Keep it short and current — this is state, not history.

_Last updated: 2026-06-11 — session: P1 part 1 (migrations + orchestrator)._

## Current state

**P0 done. P1 items 1–2 done and green** (`task ci` passes: lint 0 issues, tests green, build clean
across `pkg`, `proto`, `services/ingestion`, `services/orchestrator`).

- **Migrations**: `deploy/migrations/0001_init.{up,down}.sql` — orgs, `gold_codebase_index`
  (pgvector VECTOR(3072) + HNSW), rulesets, `verdict_audit`, `outbox`. Apply with `task migrate`
  (runs psql inside the compose postgres).
- **`services/orchestrator/`** (own module, hexagonal): domain `Submission`/`Verdict`/`Consolidate`/
  Blake3 `CacheKey` (100% cov); app `ProcessSubmission` Saga (97.7% cov) — cache-hit republish,
  pipeline under 1200 ms deadline, deterministic fallback (never blocks, fallback verdicts NOT cached,
  cache read/write errors non-fatal); adapters: `rediscache` (72h TTL, Ping), `kafkabus`
  (consumer group on `code.submission.v1` → Saga; publisher to `code.verdict.v1`, keyed by org),
  `memorycache`/`memorypublisher` fakes (100%), `stages.SecretScan` (deterministic credential screen,
  blocking findings; doubles as fallback). Health on `:8102`. In-memory smoke: `/readyz` ready.
- **Ingestion** unchanged (P0). Envelope JSON is duplicated in ingestion-kafka and orchestrator-kafkabus —
  keep in sync until a shared schema package lands (planned with the outbox item).
- **Commits**: everything committed **locally only — NOT pushed** (origin is 2+ commits behind; an
  earlier push attempt was denied pending the user's "pushla"). Do not push without "pushla".

## Next up  (P1 continuation; see docs/ROADMAP.md)

Do these in order:

1. **E2E smoke** (needs Docker Desktop running): `task dev-up` → `task migrate` → run ingestion
   (Kafka mode) + orchestrator → `POST /webhook` (body sample in `deploy/compose/.env.example` keys)
   → assert a verdict lands on `code.verdict.v1` (e.g. `docker compose exec redpanda rpk topic consume
   code.verdict.v1 -n 1`). Fix anything it shakes out.
2. **Static analyzer service** (`services/analyzer`, ROADMAP P1): AST parse + secret detection/masking
   (Go; tree-sitter or go/parser per language — start with Go+generic). It becomes a real
   `AnalysisStage` for the orchestrator (gRPC or library call — decide and record in DECISIONS).
3. **Context resolver + pgvector RAG** (`services/resolver`): gold-codebase queries per ARCHITECTURE §4.
4. Then: transactional outbox relay + shared envelope package (replace the duplicated JSON envelopes).

## Blockers / gotchas

- **Stray `GOWORK`** env on this machine → ALWAYS run Go via `task` or with `GOWORK=off`.
- **Docker daemon was not running** this session → E2E smoke deferred (step 1 above).
- **`-race`**: CI-only (no local cgo). **buf**: prebuilt exe in `%USERPROFILE%\go\bin` (see SKILLS).
- **gh**: authed with the user's classic PAT; push/board ops may be permission-gated — push only on "pushla".

## How to verify

- `task ci` → all green (4 modules).
- Orchestrator unit story: `services/orchestrator/internal/app/process_test.go` covers hit/miss/timeout/
  fallback/publish-failure paths; `task test` shows domain/app ≥97%.
- In-memory smokes: `task run:ingestion` → readyz + webhook 202; orchestrator with
  `MEMBRANE_ORCHESTRATOR_USE_IN_MEMORY=true` → `:8102/readyz` ready.

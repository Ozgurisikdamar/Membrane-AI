# HANDOVER — living session state

> **On "devam et": read this file, then do "Next up". Update this file before the session ends.**
> Keep it short and current — this is state, not history.

_Last updated: 2026-06-11 — session: P1 part 4 (outbox + shared envelope). **P1 COMPLETE.**_

## Current state

**P0 ✓, P1 ✓** (orchestrator, analyzer, resolver, outbox, E2E). `task ci` green across **6 modules**.

- **`pkg/envelope`**: single wire contract for `code.submission.v1` / `code.verdict.v1` (golden-JSON
  tested). Ingestion + orchestrator adapters refactored onto it; orchestrator mapping lives in
  `internal/adapters/codec` (used by consumer, direct publisher AND outbox sink — encoded once).
- **Transactional outbox live (D-013/D-021)**: `outboxstore.Store` implements the Saga's
  `VerdictPublisher` port → `verdict_audit` + `outbox` written in ONE ACID tx; in-process `Relay`
  (500ms/100 rows, `FOR UPDATE SKIP LOCKED`, at-least-once) ships rows via
  `kafkabus.Publisher.PublishRecord`. Failed publish ⇒ rollback ⇒ retry next tick.
  **Proven**: live-DB integration test (`MEMBRANE_ORCHESTRATOR_TEST_DSN`) + full E2E — verdict reached
  Kafka via the relay, audit row written, outbox `published=true`; orchestrator readyz now checks
  kafka+redis+postgres.
- Schema: `verdict_audit.org_id` now VARCHAR (no FK, D-022); 0001 evolves in place pre-1.0 — dev DB was
  recreated this session (`down -v` + `task migrate`).
- **Commits local on `main`, NOT pushed** (push only on "pushla"). Dev stack left running.

## Next up  (P2 begins; see docs/ROADMAP.md)

1. **Semantic service scaffold** (`services/semantic`, Python 3.12 / FastAPI) per
   ENGINEERING-STANDARDS §8: hexagonal split (domain/app/ports/adapters), `ruff` + `mypy --strict`,
   pydantic v2, pytest ≥80% domain/app. First slice:
   - `POST /v1/semantic/evaluate` — accepts (masked diff + gold context), returns findings.
   - **Three-tier cost gate skeleton**: Tier-2 local-model port (stub adapter now), Tier-3 premium
     consensus port behind a feature flag (no real API calls yet — record key handling in DECISIONS
     when added). Decide HTTP+pydantic vs proto contract and record as D-023.
   - Add a `task` target (`run:semantic`, `lint:py`, `test:py`) + CI job (ruff/mypy/pytest).
2. **Orchestrator semantic stage**: new `AnalysisStage` adapter calling the semantic service with the
   Saga deadline + the resolver's gold context (RAG injection — resolver client port into orchestrator
   or semantic service calls resolver itself: decide & record).
3. Then: reporter service (verdict consumer → PR status/comments) per ARCHITECTURE §2.

## Blockers / gotchas

- **Stray `GOWORK`** env → ALWAYS `task …` or `GOWORK=off`. **buf** = prebuilt exe. **`-race`** = CI-only.
- Dev DB was recreated; if old containers/volumes linger: `task dev-up` + `task migrate` re-seed schema.
- Integration tests: `MEMBRANE_ORCHESTRATOR_TEST_DSN` / `MEMBRANE_RESOLVER_TEST_DSN` =
  `postgres://membrane:membrane@localhost:5432/membrane`.
- **gh**: classic-PAT auth; push/board ops gated — push only on "pushla".

## How to verify

- `task ci` → all green (6 modules).
- Outbox: orchestrator integration test above; E2E (HANDOVER recipe of P1 part 3) now also leaves a
  `verdict_audit` row + `outbox.published_at` set — check with
  `docker compose -f deploy/compose/docker-compose.dev.yml exec -T postgres psql -U membrane -d membrane -c "TABLE verdict_audit;"`.

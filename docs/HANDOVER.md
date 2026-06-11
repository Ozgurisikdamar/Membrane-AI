# HANDOVER — living session state

> **On "devam et": read this file, then do "Next up". Update this file before the session ends.**
> Keep it short and current — this is state, not history.

_Last updated: 2026-06-11 — session: docs kit + P0 scaffold + ingestion service._

## Current state

**P0 is complete and green.** Repo at `C:\Dev\Membrane-AI`.

- **Docs + continuity kit**: `CLAUDE.md` + `docs/{ARCHITECTURE,ENGINEERING-STANDARDS,SKILLS,ROADMAP,
  DECISIONS,HANDOVER}.md`; reports under `docs/report` & `docs/blueprint` (md + figures) and as `.docx`.
  **Pushed to `main`.**
- **Monorepo**: `go.work`, `Taskfile.yml`, `.golangci.yml` (v2, hexagonal depguard), `.gitignore/.gitattributes/.editorconfig`, `.github/workflows/ci.yml` (3-OS build/test + lint + race).
- **`pkg/`** (own module): `config`, `logging` (slog+trace), `errs` (typed), `health`. Tested (87–100%).
- **`proto/`** (own module): `membrane.ingestion.v1` contract + committed generated stubs (`proto/gen`).
- **`services/ingestion/`** (own module, hexagonal): domain `Submission` (100%), app `EnqueueSubmission`
  (100%), ports, adapters = gRPC server (SubmitDiff + StreamCodeDiff), HTTP webhook (HMAC-verifying),
  Kafka publisher (franz-go) + in-memory publisher, uuid/clock; `cmd/ingestion` composition root with
  graceful shutdown. Smoke-tested in-memory: `/livez` ok, `/readyz` ready, `POST /webhook` → 202.
- **Quality**: `task ci` (lint+test+build) **passes**; domain/app layers at 100% coverage.
- **Commit status**: P0 code committed **locally only — NOT pushed** (waiting for "pushla"). Docs commit
  was pushed.

## Next up  (P1 — core analysis pipeline; see docs/ROADMAP.md)

Do these in order:

1. **DB migrations** — add SQL for `enterprise_organization`, `gold_codebase_index` (pgvector
   `VECTOR(3072)` + HNSW), `architectural_rulesets`, `verdict_audit`, `outbox` (transactional outbox).
   Put under `deploy/migrations/` (and a `task migrate` using golang-migrate or psql).
2. **`services/orchestrator`** — new Go module (add to `go.work` + Taskfile `GO_MODULES`). Hexagonal.
   - Consume `code.submission.v1` (franz-go consumer adapter; in-memory consumer fake for tests).
   - Saga use-case: cache-check → (miss) AST/vector/semantic placeholders → consolidate; honor a 1200 ms
     deadline with deterministic fallback (Strategy + Circuit Breaker patterns per ENGINEERING-STANDARDS).
   - **Redis Blake3 verdict cache** adapter (hit/miss) — port `VerdictCache`; in-memory fake.
   - Domain + app ≥80% tests; wire health (redis ping, consumer lag).
3. Then continue P1: static analyzer + resolver + pgvector RAG (separate items).

## Blockers / gotchas (read before running anything)

- **Stray `GOWORK`**: a machine-level `GOWORK` points at an unrelated path. ALWAYS run Go via `task`
  (it sets `GOWORK=off` inline) or prefix commands with `GOWORK=off`. CI sets `GOWORK: off` per job.
- **`-race`** needs cgo/gcc → not available locally on Windows; the race detector runs in **CI only**.
- **buf** is the prebuilt binary in `%USERPROFILE%\go\bin` (`go install` of buf fails — upstream build bug).
  If buf is missing on a fresh machine, download the release exe (see git history / SKILLS).
- **gh** is authed with the user's classic token this session; if a future session's remote op fails,
  ask for a classic `repo`+`project` PAT (CLAUDE.md). Push only on "pushla".

## How to verify

- `task ci` → lint + test + build all green.
- `task run:ingestion` (in-memory) → `curl http://localhost:8101/readyz` is ready; `POST` the sample body
  from `deploy/compose/.env.example` context to `http://localhost:8001/webhook` → 202 with a submission id.

# ROADMAP — MEMBRANE.AI

**Source of truth** for what to build, in order. GitHub Project #5 mirrors this. Tick boxes as items
move; reflect state in `docs/HANDOVER.md` too. Phases gate on the Definition of Done in
`docs/ENGINEERING-STANDARDS.md §10`.

Legend: `[ ]` todo · `[~]` in progress · `[x]` done.

## P0 — Foundation & first service ✅ DONE

- [x] Repo docs layer + session-continuity kit (CLAUDE.md + docs/)
- [x] Monorepo skeleton: `go.work`, `Taskfile.yml`, `.golangci.yml`, lint/CI config
- [x] Shared `pkg/`: `config`, `logging`, `errs`, `health` (unit-tested)
- [x] **Ingestion service** — proto contract + hexagonal domain/app/ports/adapters
  - [x] gRPC `SubmitDiff` + duplex `StreamCodeDiff` adapter (bufconn-tested)
  - [x] HTTP Git-webhook adapter (HMAC signature verification)
  - [x] Kafka publisher adapter (franz-go) + in-memory fake
  - [x] domain `Submission` aggregate + `EnqueueSubmission` use-case (100% tests)
- [x] Dev stack: `deploy/compose/docker-compose.dev.yml` (Redpanda, Redis, pgvector)
- [x] CI: `.github/workflows/ci.yml` (lint + test + build matrix + buf + race)

## P1 — Core analysis pipeline (current phase)

- [x] DB migrations (orgs, gold_codebase_index+pgvector/HNSW, rulesets, verdict_audit, outbox) + `task migrate`
- [x] **Orchestrator** — Saga consuming `code.submission.v1`; cache-first; 1200 ms deadline + deterministic
      fallback; verdicts to `code.verdict.v1`; secret-scan stage as stage 1 + fallback
- [x] **Redis** Blake3 verdict cache (hit/miss path; 72h TTL; memory fake for tests)
- [ ] E2E smoke against the dev stack (`task dev-up` + `task migrate` + ingestion→orchestrator flow) — Docker
      daemon was down this session
- [ ] **Static analyzer** — AST parse + secret detection/masking (Go/WASM); replaces/extends the secret-scan stage
- [ ] **Context resolver** + **Aurora/pgvector** gold-codebase index + RAG retrieval
- [ ] **Transactional outbox** for DB↔Kafka consistency (schema is ready in 0001_init; relay + shared envelope pkg pending)

## P2 — Semantic engine, gateway & agentic governance

- [ ] **Semantic AI service** (Python/FastAPI) — three-tier cost gating, dual-model consensus
- [ ] **Point-of-generation prompt/MCP gateway**
- [ ] **MCP gateway + tool-call governance + package firewall + shadow-AI discovery**
- [ ] **Reporter** — IDE inline fixes, PR status/comments, Slack/Jira/SIEM, verdict audit
- [ ] Accuracy & evaluation harness (golden datasets, precision/recall, FP-rate SLO)

## P3 — Surfaces, deployment, hardening

- [ ] **IDE extension** (VS Code / Cursor) + **CLI "Code Sweeper"**
- [ ] **SCM integration** (GitHub/GitLab) end-to-end
- [ ] **Shadow-mode → enforcement** merge gates
- [ ] **Observability & SRE** (OpenTelemetry, SLOs, dashboards)
- [ ] **Deployment**: SaaS / VPC / air-gapped — Terraform + Helm; multi-arch images; static client packaging
- [ ] **Compliance prep** (SOC 2 / ISO 27001 / 42001), docs & landing site

> The 20 GitHub board items map onto these phases; when you finish one, set its board card to Done and
> tick it here. Keep both in sync.

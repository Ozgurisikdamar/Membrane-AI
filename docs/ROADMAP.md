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
- [x] **E2E smoke against the dev stack** — webhook → ingestion → Kafka → orchestrator → remote analyzer
      (gRPC) → `rejected` verdict on `code.verdict.v1`; second identical diff served from Redis with
      `source:"cache"`. Migration fix en route: 3072-dim HNSW must be a `halfvec` expression index (D-020)
- [x] **Static analyzer service** (`services/analyzer`, gRPC `membrane.analyzer.v1`) — secret detection with
      line numbers + **masking** (`[MASKED:<rule>]`), risky-pattern rules (SQL/exec concat, InsecureSkipVerify);
      orchestrator `grpcstage` adapter wired via `ANALYZER_ADDR` (D-018/D-019). AST detectors deferred to
      resolver integration (full-file content needed)
- [x] **Context resolver** (`services/resolver`, gRPC `membrane.resolver.v1` :9004) — `GoldIndex` port +
      pgx/halfvec adapter (live-DB integration test green), stub `Embedder` (D-020), RAG top-k query
      - [ ] swap stub embedder for real embeddings once the semantic service exists
      - [ ] orchestrator/semantic stage consumes resolver context (P2, with the semantic service)
- [x] **Transactional outbox** (D-013/D-021): orchestrator writes `verdict_audit` + `outbox` in one ACID
      tx (`outboxstore.Store` implements the Saga's publisher port); in-process relay ships pending rows
      with `FOR UPDATE SKIP LOCKED` — proven by live-DB integration test + E2E (audit row, published=true)
- [x] **Shared envelope package** (`pkg/envelope`, golden-JSON tested) — single wire contract; ingestion
      and orchestrator adapters refactored onto it via `orchestrator/internal/adapters/codec` (schema is ready in 0001_init; relay + shared envelope pkg pending)

## P2 — Semantic engine, gateway & agentic governance

- [~] **Semantic AI service** (Python/FastAPI, `:8005`) — hexagonal scaffold + `/v1/semantic/evaluate`
      live; **cost-gate skeleton done** (tier-2 heuristic stub + tier-3 behind `PREMIUM_ENABLED` flag,
      escalation policy in domain, ruff+mypy-strict+pytest green, D-023)
      - [x] **tier-2 vLLM adapter** (OpenAI-compatible chat completions, strict-JSON parse, D-028)
            wrapped in `FallbackLocalModel` → heuristic degradation; enabled via `VLLM_URL`
            - [ ] point at a real vLLM deployment + tune the prompt against real model output
      - [ ] real tier-3 premium consensus (Claude Sonnet 4.6 + Gemini) adapter + key handling decision
      - [x] **orchestrator semantic stage** (`semanticstage` HTTP adapter w/ Saga deadline) + **resolver
            RAG injection** (`ResolverFetcher`, best-effort) — D-024: `StageResult` threads the masked
            diff to later stages; `app.Optional` decorator keeps advisory failures non-fatal
- [ ] **Point-of-generation prompt/MCP gateway**
- [ ] **MCP gateway + tool-call governance + package firewall + shadow-AI discovery**
- [~] **Reporter** (`services/reporter`, health :8105) — consumes `code.verdict.v1`; report rendering
      (outcome mapping, finding cap), idempotent fan-out per (submission, notifier); **webhook
      (Slack-compatible) + log notifiers live and E2E-proven** (D-025)
      - [x] `commit_sha`/`pr_number` plumbed end-to-end (proto → envelope → ingestion → orchestrator →
            verdict; stripped from cache, re-stamped per submission) → **GitHub commit-status adapter
            live & E2E-proven** (`membrane-ai/governance` context)
      - [x] **PR-comment adapter** (`github-pr-comment`): markdown comment with fenced findings on
            `issues/{pr}/comments` when `pr_number` present; same token family, httptest-covered
      - [ ] Redis-backed delivery log for multi-replica
      - [ ] IDE inline-fix channel (with the IDE extension, P3)
- [ ] Accuracy & evaluation harness (golden datasets, precision/recall, FP-rate SLO)

## P3 — Surfaces, deployment, hardening

- [~] **CLI "Code Sweeper"** (`clients/cli`, static binary `membrane`) — `membrane scan [path]` offline
      scan via `pkg/scan` (detectors unified across analyzer/orchestrator/CLI, D-027); human + `--json`
      output; CI exit codes (`--fail-on`)
      - [x] **"Generative-AI Technical-Debt Report" output mode** (the GTM lead magnet):
            `--report md|html [--out file]` — per-rule/per-directory/top-file aggregates plus a
            transparent severity-weighted debt score and A–F grade; HTML is a single self-contained
            light-toned page (`internal/report`)
      - [ ] packaging matrix: winget/brew/deb/rpm/tarball (P3 release work)
- [ ] **IDE extension** (VS Code / Cursor)
- [ ] **SCM integration** (GitHub/GitLab) end-to-end
- [ ] **Shadow-mode → enforcement** merge gates
- [ ] **Observability & SRE** (OpenTelemetry, SLOs, dashboards)
- [~] **Deployment**: SaaS / VPC / air-gapped — Terraform + Helm; multi-arch images; static client packaging
      - [x] **Container images (D-029)**: parameterized distroless Dockerfile for all Go services +
            python-slim semantic; `task build:images`; `docker-compose.full.yml` all-container profile
            (`task full-up`) with dual-listener Redpanda + one-shot migrate; CI `images` job
      - [ ] Helm chart (next deployment step) · Terraform · multi-arch (buildx) publish
- [ ] **Compliance prep** (SOC 2 / ISO 27001 / 42001), docs & landing site

> The 20 GitHub board items map onto these phases; when you finish one, set its board card to Done and
> tick it here. Keep both in sync.

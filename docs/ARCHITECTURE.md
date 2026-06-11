# ARCHITECTURE — MEMBRANE.AI

Authoritative architecture summary for engineers and agents. The narrative/visual long-forms are in
`docs/blueprint/blueprint.md` and `docs/report/report.md`; this file is the operational reference.

## 1. System in one paragraph

A developer's AI coding request is intercepted **at the point of generation** by a gateway that injects
the organization's architectural context; the produced diff (and every IDE/CI diff) flows through a
durable event bus into a Go analysis pipeline that runs deterministic AST + secret masking, retrieves
the organization's "gold-codebase" patterns (RAG over pgvector), and — only when cheaper tiers flag
risk — escalates to a multi-model semantic consensus. Verdicts return to the IDE/PR and are recorded in
a tamper-proof audit trail. Runtime is **cache-first**: a Blake3 hash can short-circuit the whole
pipeline in sub-millisecond time.

## 2. Components & responsibilities

| Service (module) | Lang | Responsibility |
| --- | --- | --- |
| `clients/gateway` | Go | Proactive prompt/MCP gateway: intercept, enrich with gold-context + policy, screen responses. |
| `clients/cli` | Go | "Code Sweeper" CLI (`membrane scan`): offline repo/file scan via `pkg/scan`, human+JSON output, CI exit codes — single static binary (`task build:cli`). |
| `services/ingestion` | Go | gRPC duplex stream (IDE) + Git webhooks (HTTP) → publish `code.submission.v1` keyed by org UUID. **(P0)** |
| `services/orchestrator` | Go | Saga state machine: cache → AST → vector → semantic → consensus; 1200 ms IDE deadline + deterministic fallback. |
| `services/analyzer` | Go | Deterministic static analysis (gRPC `membrane.analyzer.v1`): secret detection + masking, risky-pattern rules; AST detectors land with resolver-provided full files (D-019). |
| `services/resolver` | Go | Repo metadata + gold-codebase vector queries (pgvector HNSW). |
| `services/semantic` | Python/FastAPI | `POST /v1/semantic/evaluate` (HTTP+JSON, D-023): tier-2 local model + tier-3 premium consensus behind the cost gate (D-008); receives masked diffs + resolver gold context. |
| `services/reporter` | Go | Consumes `code.verdict.v1` → idempotent notifications (Slack-compatible webhook + logs today; GitHub commit-status when `commit_sha` lands, D-025). Health `:8105`. |
| `pkg/*` | Go | Shared foundation: `config`, `logging`, `errs`, `health`, `envelope`, **`scan`** (the single detector/masking source, D-027). |

## 3. Event & data flow

```
IDE/Agent ─gRPC┐                         ┌─ Redis (Blake3 verdict cache, 72h)
SCM webhooks ─HTTP┤→ ingestion → Kafka → orchestrator ─→ analyzer
                                              │            └→ resolver → Aurora/pgvector (gold index)
                                              └→ semantic (vLLM + cloud consensus)
                                                     │
                                              reporter → IDE / PR / SIEM  +  Aurora (verdict_audit)
```
- **Topic:** `code.submission.v1`, key = org UUID (strict per-tenant ordering). Schema in `proto/`.
- **Cache-first:** orchestrator checks Redis (Blake3 of diff + ruleset version) before any compute.
- **Consistency:** services that both write Postgres and emit Kafka use the **transactional outbox**
  (write row + outbox row in one ACID tx; a relay publishes the outbox). No dual-writes. (D-013)

## 4. Data design (Aurora PostgreSQL + pgvector)

| Table | Key columns |
| --- | --- |
| `enterprise_organization` | `org_id` (PK), `company_name`, `created_at` |
| `gold_codebase_index` | `vector_id` (PK), `org_id` (FK), `file_path`, `language_tag`, `raw_code_content`, `architectural_context`, `embedding VECTOR(3072)` + HNSW(`vector_cosine_ops`, m=16, ef_construction=64) |
| `architectural_rulesets` | `org_id` (FK), `rule`, `severity`, `version` |
| `verdict_audit` | `submission_id`, `verdict`, `model`, `created_at` — tamper-proof compliance log |
| `outbox` | `id`, `aggregate`, `topic`, `payload`, `created_at`, `published_at` — transactional outbox |

Per-organization isolation: dedicated schema, KMS-managed keys, masked inputs before any external call.

## 5. Target monorepo layout

```
Membrane-AI/
├─ CLAUDE.md
├─ go.work                      # ties all Go modules together
├─ Taskfile.yml  .golangci.yml  .editorconfig  .gitignore  .gitattributes
├─ docs/                        # continuity kit + report + blueprint
├─ proto/                       # buf module — membrane/<svc>/v1/*.proto
│  └─ buf.yaml  buf.gen.yaml
├─ pkg/                         # shared Go library (own module): config logging errs health kafka
├─ services/
│  └─ <svc>/                    # own Go module, hexagonal:
│     ├─ cmd/<svc>/main.go      #   composition root (wires adapters → app)
│     ├─ internal/domain/       #   pure entities/value-objects, no I/O
│     ├─ internal/app/          #   use-cases; depend only on ports
│     ├─ internal/ports/        #   interfaces (driven + driving)
│     ├─ internal/adapters/     #   grpc/http/kafka/redis/postgres impls
│     └─ internal/config/       #   service config struct
├─ clients/ {cli, gateway}/     # Go modules, same hexagonal shape
├─ deploy/ {compose, helm, terraform}/
└─ .github/workflows/ci.yml
```

Hexagonal rule of thumb: **dependencies point inward** — `adapters → app → domain`; `domain` imports
nothing from `app`/`adapters`; `app` depends only on `ports` (interfaces), never concrete adapters.

## 6. Conventions

- **Proto packages:** `membrane.<service>.v1` (e.g. `membrane.ingestion.v1`); files under
  `proto/membrane/<service>/v1/`. Breaking changes gated by `buf breaking`.
- **Kafka topics:** `<domain>.<event>.v<major>` (e.g. `code.submission.v1`).
- **Go module paths:** `github.com/Ozgurisikdamar/Membrane-AI/<path>` (e.g. `.../services/ingestion`, `.../pkg`).
- **Service ports (local dev):** gRPC `:90xx`, HTTP `:80xx`, health `:81xx` — ingestion = gRPC `:9001`,
  HTTP `:8001`, health `:8101`; orchestrator health `:8102`; analyzer gRPC `:9003`/health `:8103`;
  resolver gRPC `:9004`/health `:8104`. The Python semantic service serves HTTP **and** health on one
  port `:8005` (D-023).
- **Config:** 12-factor, env vars prefixed `MEMBRANE_<SVC>_…`, validated at startup (fail fast).
- **Errors:** typed domain errors in `pkg/errs`; wrap with `%w`; map to gRPC/HTTP codes at the adapter edge.
- **Observability:** structured slog JSON with a propagated `trace_id`; OpenTelemetry spans across hops.

## 7. Cross-platform build rules (D-010)

- Clients: `CGO_ENABLED=0 go build` → static binaries for `windows/macos/linux × amd64/arm64`, glibc+musl.
- Services: multi-arch distroless OCI images (`linux/amd64`, `linux/arm64`).
- No kernel/libc/init-system assumptions in client code; optional eBPF features degrade gracefully.

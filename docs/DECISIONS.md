# DECISIONS — Architecture Decision Log (ADR-lite)

Settled decisions. **Do not relitigate** any of these without the user explicitly reopening one.
Each entry: ID · decision · why. Newest at the bottom.

| ID | Decision | Rationale |
| --- | --- | --- |
| **D-001** | Product brand is **MEMBRANE.AI** (single name). | "Dev-AI" was a working alias; dropped everywhere. |
| **D-002** | All repo content (code, docs, README) in **English**; **replies to the user always in Turkish**. | Universal codebase + the user's preferred conversation language. |
| **D-003** | Repo lives at **`C:\Dev\Membrane-AI`**, outside OneDrive. | Avoids OneDrive sync locking `.git`/`node_modules`/build artifacts. |
| **D-004** | **Microservice** architecture, each service **hexagonal** (ports & adapters). | Independent deploy/scale; testable domain; the "ultra-premium" bar. |
| **D-005** | **Go** for core/edge/data services; **Python (FastAPI)** only for the semantic-AI service. | Go = static binaries, concurrency, low footprint; Python = ML ecosystem. |
| **D-006** | Eventing: **Redpanda / Kafka**, topic `code.submission.v1`, partitioned by org UUID, S3 tiered audit. | Durable, replayable, per-tenant ordering & isolation. |
| **D-007** | State: **Redis** (Blake3 verdict cache, 72h volatile-lru) + **Aurora PostgreSQL + pgvector/HNSW** (rules, gold-codebase index, audit). | Sub-ms skips + durable relational + vector search in one store. |
| **D-008** | Cost control: **three-tier gating** — cache → context pruning → local model triage → premium consensus only on risk. | Makes per-push LLM review economically viable. |
| **D-009** | Models: cloud consensus = **Claude Sonnet 4.6** (+ Gemini); local = **DeepSeek-Coder/CodeLlama** on vLLM; embeddings = OpenAI `text-embedding-3-large`. | Current models (replaced stale Claude 3.5 / Gemini 1.5 refs). |
| **D-010** | **Cross-platform**: static Go client binaries (`CGO_ENABLED=0`, win/mac/linux × amd64/arm64, glibc+musl) + multi-arch distroless service containers; runs on any Linux distro & any Kubernetes. | Run everywhere developers/pipelines run; clears regulated-industry bar. |
| **D-011** | **Deployment postures**: Masked SaaS · Private VPC · Air-gapped on-prem. | Match isolation to customer; air-gapped unblocks banks/defense. |
| **D-012** | **Push only on "pushla"**; commit identity fixed to Özgür Işık Damar; **no AI trailers**; commits short/English. | User's standing workflow rules. |
| **D-013** | DB↔Kafka consistency via **transactional outbox** (ACID write + relay), not dual-write. | Exactly-once-ish event emission without distributed transactions. |
| **D-014** | Doc build pipeline (report/blueprint docx) = matplotlib figures + `marked`+`docx-js`; visual QA via Word COM → PDF → PyMuPDF. Masters live in `docs/report` & `docs/blueprint`. | Reproducible; graphviz/mermaid/LibreOffice not installed. |
| **D-015** | Monorepo with **`go.work`**; each service is its own Go module; shared code in **`pkg/`**; protobuf via **buf**; task runner **go-task** (`Taskfile.yml`). | Clean module boundaries + one-command DX. |
| **D-016** | Project tracking: **`docs/ROADMAP.md` is the source of truth**; GitHub Project #5 board is a mirror. | Docs work offline and across any model/session. |
| **D-017** | First service built in P0 = **`ingestion`** (gRPC stream + Git webhook → Kafka). | It is the entry point of the whole pipeline; everything else consumes its events. |
| **D-018** | **Analyzer is a separate gRPC microservice** (`membrane.analyzer.v1`); the orchestrator calls it as a remote `AnalysisStage` (config `MEMBRANE_ORCHESTRATOR_ANALYZER_ADDR`; empty ⇒ in-process secret-scan only). The in-process secret scan always remains the deterministic fallback. | Matches the microservice architecture (D-004) and the report's analyzer pods; lets the analyzer scale/deploy independently; Saga already degrades safely on stage errors. |
| **D-019** | Analyzer v1 detectors are **line-scan based** (secrets + masking + risky patterns). **AST detectors are deferred** until the context resolver provides full-file content (a diff alone cannot be parsed into a meaningful AST). | Honest scope: no fake AST; masking is the privacy-critical deliverable (secrets never reach LLM stages). |
| **D-020** | Resolver ships with a **deterministic stub embedder** (signed feature-hashing of token shingles, L2-normalized) behind the `ports.Embedder` swap point until the semantic service provides real embeddings. **3072-dim embeddings are HNSW-indexed via a `halfvec` expression index** (pgvector caps plain `vector` HNSW at 2000 dims); queries MUST use the same `embedding::halfvec(3072)` expression to hit the index. Org IDs are strict UUIDs at the resolver boundary (schema is UUID-keyed). | Unblocks the full RAG plumbing now (proven by a live integration test); real embeddings drop in by replacing one adapter. |
| **D-021** | **Outbox relay runs in-process** inside the orchestrator (interval poller, default 500ms/100 rows) using `FOR UPDATE SKIP LOCKED`, so multiple replicas never double-claim. Delivery is **at-least-once** (publish-then-commit); consumers dedupe on `submission_id`. The Saga's `VerdictPublisher` port is wired to the outbox store in Kafka mode — Saga code unchanged. | No extra deployable; replica-safe by construction; durability-first per D-013. |
| **D-022** | `verdict_audit.org_id` is a **free-form tenant string with no FK** (tenancy/UUID enforcement lives at the gold-index/resolver boundary). **Pre-1.0 migration policy:** `0001_init` evolves in place and dev DBs are recreated (`docker compose down -v` + `task migrate`); append-only migrations start at the first release. | The audit log must never drop a verdict because an org row isn't provisioned; clean migration history pre-release. |

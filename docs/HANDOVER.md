# HANDOVER — living session state

> **On "devam et": read this file, then do "Next up". Update this file before the session ends.**
> Keep it short and current — this is state, not history.

_Last updated: 2026-06-12 — session: P3 completion sweep (eval harness, packaging, Terraform,
multi-arch, Redis delivery log, MCP gateway, shadow→enforce, VS Code ext, SCM, compliance, landing,
dashboards)._

## Last review (2026-06-12)

Full-repo multi-role review (34 agents, 9 role lenses, adversarially verified): 25 raw findings →
4 confirmed real, all fixed: (1) **reporter notifier HTTP calls were unbounded** — now bounded by a
30s client timeout (the serial verdict consumer could hang on a stalled GitHub/webhook endpoint);
(2) **semantic config didn't reject ≤0 timeouts** → `_get_positive_float` fail-fast + `test_config.py`
(new); (3+4) **Terraform module only plumbed ~half the chart values** → now exposes the full surface
(vLLM, premium keys+models, enforcement_mode, GHE URL, webhook URL, otlp_insecure). Everything else
was refuted (codebase held up). All green after fixes.

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

**P0 ✓ P1 ✓ P2 ✓ P3 ✓ — every buildable roadmap item is complete.** `task ci` green: 10 Go modules
+ Python (36 pytest); `task helm:lint` green; `task eval` green (golden corpus P/R 1.0, FP-rate 0).
The only open ROADMAP items are runtime/live-tuning that need resources MEMBRANE.AI doesn't control —
**real embeddings** (an embedding-model endpoint), **real vLLM** (a GPU deployment), **real premium
keys** (the user's Anthropic/Gemini keys) — plus the optional **IDE inline-fix** deepening. All the
code/swap-points for those are in place.

This-session additions (all tested + committed locally, push pending): eval harness (`eval/`),
CLI packaging (`.goreleaser.yaml`, D-033) + multi-arch buildx publish, reporter **Redis delivery
log**, **Terraform** module (`deploy/terraform`), **MCP gateway** service (`services/gateway`,
D-034), **shadow→enforce** merge gate (D-035), **VS Code extension** (`clients/vscode`, compiles),
**SCM** integration (GitHub App manifest + CLI PR-gate workflow), **compliance** doc + **landing**
page, observability **inc 2/3** (otelgrpc/otelhttp + metrics + Python tracing + outbox trace +
collector) and a **Grafana dashboard**.

- **Observability foundation done (D-032)**: `pkg/observability` — `Setup` installs the W3C
  propagator always + an OTLP/gRPC trace exporter only when `OTLP_ENDPOINT` is set (empty = no-op,
  prod-safe). Helpers `Stop`/`Start`/`InjectHeaders`/`ExtractContext`/`TraceID` keep otel out of
  domain/app (adapters + composition only). All 5 Go services wire `Setup`/`Stop` + config
  `MEMBRANE_<SVC>_OTLP_ENDPOINT`/`_OTLP_INSECURE`. **Distributed trace across Kafka**: ingestion
  opens the span + stamps W3C headers on the submission record; orchestrator extracts + spans the
  Saga (one trace ingestion→orchestrator). Helm `infra.otlpEndpoint` + compose `MEMBRANE_OTLP_ENDPOINT`.
- **Auto-instrumentation done (increment 2)**: otelgrpc client handlers on the orchestrator's
  analyzer/resolver dials + server handlers on ingestion/analyzer/resolver gRPC servers; otelhttp on
  the orchestrator→semantic client + the ingestion webhook server. Analyzer/resolver/semantic hops
  now join the trace (verified: global W3C propagator threads them; manual Kafka spans nest, no
  double-count; bufconn/httptest tests green).
- **Increment 3 done**: OTel metrics (`membrane_verdicts_total{decision,source}` in the orchestrator,
  meter on the same OTLP endpoint); Python (semantic) tracing (FastAPIInstrumentor, gated on
  `MEMBRANE_SEMANTIC_OTLP_ENDPOINT`); **outbox carries the trace** (`outbox.headers` JSONB → relay
  stamps record headers → reporter rejoins, so the full ingestion→orchestrator→reporter trace is one);
  opt-in bundled collector in full compose (`--profile observability`, OTLP→debug).
- **Observability complete.** Grafana dashboard + SLOs in `deploy/observability/` (live wiring needs
  a running Tempo/Prometheus/Grafana).

- **Premium tier-3 consensus done (D-031)**: `semantic/adapters/premium_consensus.py` — Claude
  Sonnet 4.6 (Anthropic Messages API) + Gemini (generateContent) via httpx, NO SDK. `DualModelConsensus`
  runs reviewers concurrently with a per-reviewer `asyncio.wait_for` deadline, merges findings tagged
  `premium:<model>:<rule>`, degrades to the survivor on partial failure (outage surfaced as an `info`
  finding), and raises only if ALL fail. API keys from env (`MEMBRANE_SEMANTIC_ANTHROPIC_API_KEY` /
  `_GEMINI_API_KEY`); wired behind `PREMIUM_ENABLED` + a key (keyless-but-enabled keeps the disabled
  stub → fails loud). Surfaced in `.env.example`, the full compose, and the Helm chart (credentials
  Secret + infra toggles).
- **Hardened by the adapter's multi-agent review** (7 confirmed, all fixed): httpx clients now closed
  via a FastAPI **lifespan** (`create_app(evaluate, closeables)`); overall consensus **deadline**
  (per-reviewer wait_for); `CancelledError` propagates instead of degrading; `_anthropic_text`/
  `_gemini_text`/`_decode` guard non-dict bodies, missing `text` keys, and non-JSON 200s →
  `PremiumConsensusError`; outage messages **sanitized** (`_summarize_error` → `HTTP <code>`/`timeout`,
  never the request URL or key).

- **Helm chart done (D-030)**: `deploy/helm/membrane` — one generic Deployment/Service template
  loops over `.Values.services` (6 services); infra (Kafka/Redis/Postgres/vLLM) as external
  endpoints; one optional Secret; opt-in topics hook Job; pods non-root + read-only rootfs +
  `automountServiceAccountToken: false`. `task helm:lint`; CI `helm` job added.
- **Hardened by the chart's multi-agent review** (all confirmed findings fixed): port overrides
  now drive the listener bind env (`addrEnv`/`healthEnv`/`portEnv`), not just the probe; `tpl`/
  image-tag use `toString` (numeric `--set` no longer crashes/garbles); `fullname` truncates to 50
  so per-service suffixes stay distinct ≤63; topics Job is injection-safe (values via env, no
  `tpl` on brokers) and PSA-restricted-clean; `databaseUrl` secretKeyRef is non-optional
  (fail-fast, not a silent localhost fallback); vLLM + reporter webhook-URL/GHE-API now surfaced;
  webhook fail-open documented in NOTES.

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

## Next up  (only resource-gated live-tuning + optional deepenings remain — see docs/ROADMAP.md)

All buildable roadmap items are done. What's left needs resources the project doesn't control:

1. **Real model wiring/tuning** (needs the user's infra/keys): point the semantic tier at a real
   vLLM deployment and the premium tier at real Anthropic/Gemini keys, then tune the prompts against
   live output; swap the resolver's stub embedder for a real embedding-model endpoint. All
   adapters/`ports.Embedder` swap-points are in place — this is config + live tuning, not new code.
2. **Live observability stack**: stand up Tempo/Prometheus/Grafana and import
   `deploy/observability/grafana-dashboard.json` (the exporters + dashboard JSON are done).
3. **Optional deepenings**: IDE inline-fix channel; a native GitHub-event ingestion adapter (parse
   push/PR + fetch the diff) to complement the CLI PR-gate; real tier-3 key-handling via a vault.

## How to verify

- `task ci` → all green. `task helm:lint` → helm lint + render + kubeconform green.
- Report mode: `task build:cli` then `bin/membrane scan . --report html --out debt.html
  --fail-on=never` (repo self-scan ≈ 34 findings, all from test fixtures — a good demo). Story
  lives in `clients/cli/internal/report/report_test.go` + `cmd/membrane/main_test.go`.
- Containers: `task build:images` then `task full-up`; from the host probe
  `curl http://localhost:8101/readyz` (ingestion), `:8102`–`:8105` (orchestrator/analyzer/resolver/
  reporter), `http://localhost:8005/readyz` (semantic). Kafka from the host = `localhost:19092`.
  Tear down with `task full-down`.
- PR comment story: `services/reporter/internal/adapters/notify/githubprcomment_test.go`.
- vLLM story: `services/semantic/tests/test_vllm.py`.
- Premium story: `services/semantic/tests/test_premium.py` (parse/coerce, partial-failure degrade,
  all-fail raise, overall timeout, sanitized outage message, non-JSON 200 / missing-text guards).

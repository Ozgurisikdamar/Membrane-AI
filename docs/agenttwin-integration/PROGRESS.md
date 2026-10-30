# AgentTwin Integration — Living Progress Report

**Status:** `IN_PROGRESS` — Phase 0 (baseline revalidation) complete and verified; Phase 1 next.  
**Last updated:** 2026-10-30  
**Primary repository:** `Ozgurisikdamar/Membrane-AI`  
**Reference repository:** `Ozgurisikdamar/AgentTwin` (read-only by default)  
**Execution brief:** [CLAUDE.md](CLAUDE.md)  
**Implementation plan:** [IMPLEMENTATION_PLAN.md](IMPLEMENTATION_PLAN.md)  
**Research baseline:** [research report](../research/MembraneAI_AgentTwin_Birlesik_Platform_Arastirma_ve_Demo_Tasarimi.md)

---

## 1. Current snapshot

| Field | Value |
|---|---|
| Current phase | Phase 1 — contract & security design (**DONE**, tested) → Phase 2 next |
| Current status | `IN_PROGRESS` |
| Membrane working branch | `main` (owner directed this workstream to develop on and push to `main`, superseding the work order §1.4 `feature/agenttwin-assurance` policy) |
| Membrane HEAD (Phase 0 inspection, 2026-10-30) | `aedc1ed4014a684868ef3a339ea1d6d5df8235e0` (== `origin/main`; supersedes the 2026-09-26 `b8e8cfd…` baseline) |
| AgentTwin reference commit (verified 2026-10-30) | `1e8fd0ebc344281eff6c2d2dd45bd423b8d11a5f` (local HEAD == documented reference; read-only) |
| `feature/agenttwin-assurance` | Exists on `origin` (carried the dashboard work-order docs); **merged into `main`** 2026-10-30 (merge `080eeb4`). All work consolidated on `main` per owner direction; see §11. |
| Code changes | Phase 1: new pure contract package `pkg/assurance` (contract/status/allowlist + tests); `pkg/assurance/contracts/` schemas, fixtures, pinned AgentTwin OpenAPI snapshot. No integration I/O yet (Phase 2). |
| Contract changes | `ReleaseEvidenceV1`, `MappingRecordV1`, `CorrelationContext` (all `schema_version=1.0.0`) + JSON Schemas — new, versioned, design-complete and tested. |
| Integration tests | Contract/compatibility unit tests in `pkg/assurance` (status mapping, validators, allowlist/SSRF, fixture round-trip, schema↔struct parity, AgentTwin OpenAPI surface). No live integration code yet. |
| Demo run in this workstream | **Reproduced live** in Phase 0 — 1.2.4 vs 1.3.0 (regression) and 1.2.4 vs 1.3.1 (fixed); see §7 and §9 |
| Membrane quality gates | `task build/test/lint` PASS with the new package (`pkg/assurance` 93.7% coverage); Phase-0 gates still green; `task helm:lint` BLOCKED (helm not installed) |
| Push status | Phase 0 + Phase 1 committed and pushed to `main` (SHAs in §9 milestone entries) |

## 2. Planning artifacts published

- [x] Research and competitor analysis report copied into Membrane AI.
- [x] Detailed Claude work order created.
- [x] Implementation plan and phase acceptance criteria created.
- [x] Living progress-report contract created.
- [x] Root `CLAUDE.md` linked to scoped instructions and branch/reporting policy.
- [x] Claude revalidates current repository heads and Phase 0 evidence. (2026-10-30)
- [x] First integration ADR and contract fixtures. (2026-10-30 — ADR-0001, THREAT_MODEL, CONTRACTS, D-037; `pkg/assurance` + schemas + fixtures + compat tests.)
- [ ] AgentTwin client/adapter. (Phase 2)
- [ ] End-to-end evidence aggregation and release gate.
- [ ] Shadow-mode reporter integration.
- [ ] Reproducible demo and handoff.

## 3. Verified initial repository observations

### Membrane AI

- Public repository inspected: https://github.com/Ozgurisikdamar/Membrane-AI
- Default branch observed: `main`.
- Inspected README, architecture, roadmap, handover, API, gateway policy/application, orchestrator flow, scanner, semantic adapters, deployment and observability docs.
- Existing capabilities and unfinished external dependencies are detailed in the research report.
- Root `CLAUDE.md` mandates Turkish user responses, local quality gates, milestone handover updates and no unapproved history rewriting.

### AgentTwin

- Public repository inspected: https://github.com/Ozgurisikdamar/AgentTwin
- Default branch observed: `main`.
- Inspected README, architecture, implementation board, threat model, benchmark notes, Python SDK, trace/simulation/evaluation services, demo agent, twin/scenario YAML and selected ADRs.
- Existing and pending AgentTwin phases are detailed in the research report.
- AgentTwin is read-only for this workstream unless separately authorized.

### Competitive research

The research report compares Langfuse, LangSmith, Arize Phoenix, OpenAI Agent Evals/Trace Grading, Microsoft Foundry, AWS AgentCore, Maxim, Promptfoo, LiteLLM, GitHub Code Security/Semgrep, OpenTelemetry GenAI, OWASP, NIST, vLLM and Hugging Face data/demo sources.

## 4. Architecture decisions — proposed, not implemented

1. Membrane AI remains primary product and owns the combined release decision.
2. AgentTwin owns manifests, traces, scenarios, twins, simulations and detailed evaluation results.
3. No immediate monorepo merge, shared database or event-bus consolidation.
4. First vertical slice uses a typed, authenticated, idempotent HTTP API adapter from Membrane to AgentTwin.
5. Initial demo reuses the AgentTwin Demo Co support/refund agent and known versions.
6. Release decisions use deterministic PASS/WARN/BLOCK rules; scores cannot hide critical failure.
7. ERROR, INCOMPLETE, missing required evidence and unverifiable outcomes can never silently pass.
8. Deterministic scripted demo planner is labelled fake; paid model credentials are optional and non-authoritative for acceptance.
9. Synthetic demo data only; no real refund/payment/customer side effects.

Validate these in Phase 0/1. Material changes require an ADR and a milestone-log explanation.

## 5. Open verification items / blockers

- [x] Reconfirm both repository head SHAs and latest AgentTwin implementation-board status. (Membrane `aedc1ed`, AgentTwin `1e8fd0e`; AgentTwin README: Phases 0–3 done; graph/release-gate/regression-miner/runtime-gateway pending.)
- [x] Inspect exact AgentTwin OpenAPI routes, API key scopes, auth mechanism and idempotency for paired evaluation. (4 OpenAPI docs; `bearerAuth` + `X-AgentTwin-Api-Key`; `Idempotency-Key` on mutations; paired pinning via `POST /internal/v1/simulation-pairs` + `POST /api/v1/eval-runs`; live-probed — authed 200, unauthed 401.)
- [ ] Verify whether AgentTwin supports a production-safe cross-project service credential or requires an upstream addition. (Open for Phase 1: demo uses a per-project API key `atk_demo…`; a cross-project service credential is not yet confirmed.)
- [x] Reproduce documented `1.2.4` vs `1.3.0` pair and record actual outcomes, seed and commands. (Live: 1.2.4 = 9/9, 1.3.0 = 6/9 / 3 failed / 2 critical; eval NEW_CRITICAL_FAILURE=2, REGRESSED=1; 1.3.1 fixed = 9/9, eval UNCHANGED=9. See §7/§9.)
- [x] Verify Membrane local environment and exact `task lint`, `task test`, build/eval commands. (All green; see §7. Toolchain notes: full go1.25 SDK required for `covdata`; helm absent → `helm:lint` blocked.)
- [ ] Choose narrowest Membrane integration boundary after current code/composition is inspected. (Phase 1.)
- [ ] Confirm CLI/report-first scope for first demo; current plan assumes this smaller, lower-risk UX. (Phase 1.)

Integration endpoints are now verified against AgentTwin's live OpenAPI + running services (not inferred).

## 6. Phase status

| Phase | Status | Exit evidence |
|---|---|---|
| 0 — Baseline revalidation | `DONE` (2026-10-30) | SHAs confirmed; API/contract inventory; Membrane gates green; refund demo reproduced live (1.3.0 regression + 1.3.1 fixed) |
| 1 — Contract/security design | `DONE` (2026-10-30) | ADR-0001 + D-037; versioned `pkg/assurance` mapping/evidence/correlation contracts + JSON Schemas; THREAT_MODEL (T1–T12); contract/compat tests green (93.7% cov) |
| 2 — Small vertical slice | `IN_PROGRESS` (2026-10-30) | Evidence builder (`BuildReleaseEvidence`) + `AgentTwinClient` port + deterministic fake landed & tested; live AgentTwin already read via the dashboard BFF. **Remaining:** Go HTTP adapter + orchestrator wiring + a one-command CLI combined report. |
| 3 — Evidence + release gate | `NOT_STARTED` | deterministic gate + Markdown/JSON + critical-rule tests |
| 4 — SCM shadow mode | `NOT_STARTED` | idempotent reporter integration + shadow-mode tests |
| 5 — Demo hardening/handoff | `NOT_STARTED` | repeatable runbook + exact test evidence |
| Later — blast radius/regression mining/runtime governance/unified UI | `PLANNED` | separate scope after MVP evidence |

## 7. Test and demo evidence log

Phase 0 verification runs (Linux cloud container; Go via full go1.25.0 SDK + `GOTOOLCHAIN=local`; Docker via `mirror.gcr.io` registry mirror). No integration code exists yet, so these are baseline/quality-gate and AgentTwin-demo reproductions, not integration tests.

| Date (UTC) | Repository / commit | Command or scenario | Result | Evidence |
|---|---|---|---|---|
| 2026-10-30 | Membrane `aedc1ed` | `task build` (10 Go modules) | `PASS` | 10 modules compiled |
| 2026-10-30 | Membrane `aedc1ed` | `task test` (Go, -cover) | `PASS` | 42 ok pkgs, 0 FAIL, 5 no-test |
| 2026-10-30 | Membrane `aedc1ed` | `task lint` (buf + 10× golangci-lint) | `PASS` | 0 issues |
| 2026-10-30 | Membrane `aedc1ed` | `task eval` (corpus accuracy SLO) | `PASS` | 14 cases, TP=8 FP=0 FN=0, P/R/F1=1.000, fp-rate=0.000 |
| 2026-10-30 | Membrane `aedc1ed` | `task lint:py` + `task test:py` | `PASS` | ruff+mypy --strict clean; 45 pytest passed |
| 2026-10-30 | Membrane `aedc1ed` | `task build:cli` + `membrane scan` (synthetic secret sample) | `PASS` | 3 blocking findings, masked excerpts, `--fail-on=blocking` exit 1, JSON OK |
| 2026-10-30 | Membrane `aedc1ed` | `task helm:lint` | `BLOCKED` | helm not installed in container (tooling gap, not a code failure) |
| 2026-10-30 | AgentTwin `1e8fd0e` | `go test ./packages/... ./services/...` (no infra) | `PASS` | 22 ok pkgs, 0 FAIL |
| 2026-10-30 | AgentTwin `1e8fd0e` | `uv run pytest -m "not integration"` | `PASS` | 514 passed, 66 deselected |
| 2026-10-30 | AgentTwin `1e8fd0e` | Live stack seed: `--simulate 1.2.4,1.3.0 --evaluate 1.2.4:1.3.0` (default seed) | `REPRODUCED` | 1.2.4=9/9; 1.3.0=6/9 (3 failed, 2 critical); eval NEW_CRITICAL_FAILURE=2, REGRESSED=1, UNCHANGED=6; runs 01a18c71-fa92/faaa/faf5 |
| 2026-10-30 | AgentTwin `1e8fd0e` | Live stack seed: `--seed 42 --simulate 1.2.4,1.3.1 --evaluate 1.2.4:1.3.1` | `REPRODUCED` | 1.2.4=9/9; 1.3.1=9/9 (0 critical); eval UNCHANGED=9, 0 new critical, 0 regressed; runs 01a18c73-c653/c669/c690 |
| 2026-10-30 | AgentTwin `1e8fd0e` | `make demo` one refund conversation (1.2.4) | `PASS` | tool calls lookup_order+send_email; trace 17d04bc66154a8fc7b96d867c8bb7288 |
| 2026-10-30 | AgentTwin `1e8fd0e` | Live API: `GET /api/v1/eval-runs` (`X-AgentTwin-Api-Key`) vs no key | `PASS` | authed 200 (2 runs); unauthed `401` |
| 2026-10-30 | Membrane `main` (Phase 1) | `task test` incl. new `pkg/assurance` (`go test -cover`) | `PASS` | `pkg/assurance` ok, coverage 93.7%; 0 FAIL across all modules |
| 2026-10-30 | Membrane `main` (Phase 1) | `task lint` incl. `pkg/assurance` (golangci-lint) | `PASS` | 0 issues |
| 2026-10-30 | Membrane `main` (Phase 1) | `task build` incl. `pkg/assurance` | `PASS` | all 10 modules compile |

## 8. Upstream AgentTwin requests

`None confirmed yet.` Do not modify AgentTwin as a workaround. Add a request only after identifying an actual upstream gap and recording source path, required change, compatibility concern and Membrane-side fallback.

## 9. Milestone log

### 2026-10-30 — Phase 1 contract & security design (DONE, tested)

- Status: `DONE` (Phase 1). Workstream `IN_PROGRESS`; Phase 2 (AgentTwin client + vertical slice) next.
- Membrane branch + HEAD SHA: `main` (push SHA recorded below). AgentTwin reference SHA: `1e8fd0e` (read-only; OpenAPI snapshot pinned at this commit).
- Scope delivered: the versioned, pure integration contract + its security design — **no integration I/O yet** (Phase 2). Boundary chosen: the orchestrator composition root, not a new service (ADR-0001 #3).
- Files changed: `pkg/assurance/{assurance,status,contract,allowlist}.go` + `*_test.go`; `pkg/assurance/contracts/{evidence,mapping}.v1.schema.json`, `contracts/fixtures/*.json`, `contracts/agenttwin/{evaluation,simulation}-service.openapi.yaml` + `PROVENANCE.md`; `docs/agenttwin-integration/{ADR-0001-agent-assurance-integration.md,THREAT_MODEL.md,CONTRACTS.md}`; `docs/DECISIONS.md` (D-037); this report; `IMPLEMENTATION_PLAN.md` (Phase 1 ticked); `docs/HANDOVER.md`.
- Implementation vs design-only: `pkg/assurance` is **implemented and unit-tested** (pure types, status mapping, validators, allowlist). The HTTP client/port/adapter, config/secret handling and the full gate-rule evaluator remain **design-only** (Phase 2–3).
- Exact commands + results: `task build` PASS (10 modules); `task test` PASS (`pkg/assurance` coverage **93.7%**, 0 FAIL); `task lint` PASS (0 issues, incl. `pkg/assurance`). See §7.
- Test counts: `pkg/assurance` tests cover MapEvalStatus (13 cases), Decide (11), validators (correlation/mapping/evidence incl. fail-safe invariants), allowlist/SSRF (12+ cases), fixture round-trip (positive + negative), schema↔struct parity, AgentTwin OpenAPI surface.
- Demo scenario, seed, observed verdicts: no new demo run this milestone; the Phase-0 live results (1.3.0 BLOCK, 1.3.1 PASS) are now encoded as checked-in evidence fixtures and asserted by tests.
- Security/privacy validation: THREAT_MODEL T1–T12. Enforced in code: destination allowlist/SSRF guard, credential-by-reference (inline-secret rejection), tenant-from-mapping (not agent attributes), fail-safe invariants (no PASS on critical/ERROR/INCOMPLETE; BLOCK/WARN must cite a rule). No secrets in fixtures or docs.
- Known limitations / blockers: contract is design-complete but not yet wired to a live call (Phase 2); timeouts/size-bounds/redirect-revalidation/DNS-rebinding are designed in THREAT_MODEL but implemented with the adapter in Phase 2; full deterministic gate-rule evaluator is Phase 3; `task helm:lint` still blocked (no helm).
- Upstream AgentTwin requests: None. Open verification item carried forward: whether AgentTwin offers a production-safe cross-project service credential (Phase 1 §5 item still open).
- Next actions: Phase 2 — add the typed AgentTwin client port + HTTP adapter in the orchestrator, config/secret handling, deterministic fake adapter for tests, and produce one combined evidence record from a manual CLI/documented action against the reproduced demo.
- Commit SHA containing this report: `61713c3` (Phase 1 contract + docs).
- Push status + remote branch: pushed to `origin/main` (content commit `61713c3`, with this SHA-record follow-up).

### 2026-10-30 — Phase 0 baseline revalidation (DONE, verified)

- Status: `DONE` (Phase 0). Workstream `IN_PROGRESS`; Phase 1 (contract + security design) next.
- Membrane branch + HEAD SHA: `main` @ `aedc1ed4014a684868ef3a339ea1d6d5df8235e0` (== `origin/main`). Per owner direction this workstream develops on and pushes to `main` (supersedes work order §1.4's `feature/agenttwin-assurance` policy; that branch does not exist in this clone).
- AgentTwin reference SHA: `1e8fd0ebc344281eff6c2d2dd45bd423b8d11a5f` (read-only; local HEAD == documented reference).
- Scope delivered: verification only — no application/integration code. Current-state inventory + live demo baseline + Membrane quality-gate evidence. This report updated.
- Files changed: `docs/agenttwin-integration/PROGRESS.md` (this report); `docs/agenttwin-integration/IMPLEMENTATION_PLAN.md` (Phase 0 checklist ticked); `docs/HANDOVER.md` (next-action note).
- Implementation vs design-only: all integration contracts remain **design-only**; AgentTwin capabilities below are **implemented-and-tested in AgentTwin** and were exercised live here.
- Exact commands + outcomes: see §7 table. Membrane `task build/test/lint/eval/lint:py/test:py/build:cli` = PASS; `task helm:lint` = BLOCKED (helm absent). AgentTwin Go unit = 22 ok/0 FAIL; Python unit = 514 passed/66 deselected.
- Test counts: Membrane Go 42 ok pkgs (0 FAIL); Membrane Python 45 passed; Membrane eval 14 cases P/R/F1=1.000; AgentTwin Go 22 ok; AgentTwin Python 514 passed.
- Demo scenario, seed, observed verdicts: refund suite `refund-regression-suite` v1 (9 scenarios). Baseline **1.2.4 = 9/9 pass, 0 critical**. Candidate **1.3.0 = 6/9 pass, 3 failed (refund-happy-path, refund-timeout-after-mutation, refund-tool-success-lie), 2 critical** → eval NEW_CRITICAL_FAILURE=2, REGRESSED=1, UNCHANGED=6 (status COMPLETED → integration-status COMPLETED_WITH_CRITICAL_FAILURE → combined gate BLOCK). Fixed **1.3.1 = 9/9 pass, 0 critical** (seed 42) → eval UNCHANGED=9, 0 new critical, 0 regressed → combined gate PASS candidate. Planner = `scripted-planner-v1` (deterministic-fake); `JUDGE_PROVIDER=fake`; no paid model credentials used. (1.3.0 run used the default case seed; 1.3.1 run used `--seed 42`.)
- Security/privacy validation: AgentTwin edge auth enforced live (`X-AgentTwin-Api-Key` → 200; no key → 401). Synthetic order/customer/refund data only; no real side effects. No secrets committed; proxy CA passed to Docker builds only as a BuildKit secret (never an image layer). Membrane `membrane scan` masks detected secrets in output.
- Known limitations / blockers: live stack needed Docker (daemon started) and a `mirror.gcr.io` registry mirror to bypass Docker Hub 429 rate-limiting on this egress IP; `web` UI + prometheus/grafana were excluded from the run (not needed for the eval); `task helm:lint` blocked (no helm). AgentTwin has no release-gate / blast-radius / regression-miner / runtime-gateway yet — the combined PASS/WARN/BLOCK gate therefore lives on the Membrane side (Phase 3). Cross-project service credential for AgentTwin not yet confirmed (Phase 1).
- Upstream AgentTwin requests: None. (AgentTwin unchanged; read-only respected.)
- Next actions: Phase 1 — define versioned mapping + release-evidence contracts and threat model; write the integration ADR; add checked-in AgentTwin OpenAPI fixtures + compatibility tests; choose the narrowest Membrane adapter boundary. Do not start Phase 2 code before contracts/threat model are reviewed.
- Commit SHA containing this report: `9fba113` (Phase 0 report + plan/handover updates).
- Push status + remote branch: pushed to `origin/main` (content commit `9fba113`, with this SHA-record follow-up).

### 2026-10-30 — Planning package prepared

- Added research report, detailed Claude work order, implementation plan and this living report to Membrane AI.
- Added a scoped pointer in root `CLAUDE.md` so future Claude sessions discover the task instructions.
- Created `feature/agenttwin-assurance` from the updated `main` at `81ffe42db02ced824ea31f7bf5390d6ff8313484`; the branch initially contains the same planning baseline and no application-code changes.
- Published repository commits: root CLAUDE pointer `c5258397d3b3e049a25076fb39ec7eb97accbf79`; research report `e02f5797ef690ff393194448acf736c7bb8020a5`; Claude work order `43a6c56575e59907b4db0253461bef76cb0a8431`; implementation plan `20eb9a0233eb9217d406a4242989a95174e5c496`; initial progress report `38edf6e1076bd84b71896844e4e9a11c4df9acb3`.
- No application code, AgentTwin files, API contracts or runtime configuration changed.
- No tests/demo run; this publication is documentation-only.
- Next: Claude executes Phase 0, verifies current heads/contracts, reproduces refund demo and records actual evidence here.

## 10. Required update template for every milestone

Refresh the top snapshot and append:

```text
### YYYY-MM-DD — Phase N / milestone title
- Status:
- Membrane branch + HEAD SHA:
- AgentTwin reference SHA:
- Scope delivered:
- Files changed:
- Tests/commands and exact results:
- Demo scenario, seed, observed verdicts:
- Security/privacy validation:
- Known limitations / blockers:
- Upstream AgentTwin requests:
- Commit SHA containing this report:
- Push status + remote branch:
- Next actions:
```

The report is part of the deliverable. A phase is not complete until the report is updated, committed with the work, and pushed to the authorized feature branch.


---

## 11. Dashboard & Netlify workstream (authorized scope extension)

**Work order:** [DASHBOARD_AND_NETLIFY_WORKFLOW.md](DASHBOARD_AND_NETLIFY_WORKFLOW.md)

The owner requested a real Membrane AI product dashboard and a Netlify-ready demo. This is a new, separately tracked workstream; it does not change the verified status of the AgentTwin integration phases above.

| Dashboard phase | Status | Required evidence |
|---|---|---|
| D0 — Baseline/UI/API/deployment inventory | `DONE` (2026-10-30) | Frontend/API/auth/deploy audited (§11.1); framework = Next.js; live/demo/unavailable data contract (§11.2) |
| D1 — Dashboard foundation | `DONE` (2026-10-30) | Next.js app shell + 9-section nav, Membrane tokens, demo/live DataSource + states; 12 Vitest tests pass; production build OK (12 routes) |
| D2 — Agent Assurance dashboard | `DONE` (2026-10-30) | BFF (server-side cred + SSRF allowlist) + live AgentTwin wiring verified; overview/agents/simulations/traces/release views; 27 unit + 4 E2E pass; build green |
| D3 — Code Governance integration | `DONE` (2026-10-30) | Code Governance (real scan, grade F, masked), Security Center, Reports export; unified release correlation; AgentTwin Phase 2 Go evidence builder + client port + fake; 33 unit + 7 E2E green |
| D4 — Netlify demo readiness | `NOT_STARTED` | Verified build/publish paths, config, headers, environment inventory, successful preview |
| D5 — Final dashboard acceptance | `NOT_STARTED` | Reproducible runbook, real screenshots, responsive/accessibility/E2E evidence, owner handoff |

### 11.1 D0 audit — current-state inventory (verified 2026-10-30)

Branch reconciliation: `origin/feature/agenttwin-assurance` (which carried the dashboard
work-order docs, branched from pre-Phase-0 `main`) was **merged into `main`** so `main` now
carries both the AgentTwin Phase 0/1 work and the dashboard work order. Per owner direction all
D0–D5 + AgentTwin phases proceed on `main` (this overrides the work order's "push to feature /
don't merge to main"; recorded here deliberately). Merge commit recorded in the milestone log.

**Frontend reality:** Membrane has **no web frontend / Node / Netlify tooling**. Only assets:
`docs/landing/index.html` (hand-written static marketing page), the VS Code extension
(`clients/vscode`, shells the CLI → editor diagnostics), and the CLI HTML/JSON report
(`clients/cli/internal/report`). Brand tokens to reuse: teal `#0f766e`/`#0d9488`, paper `#f6f3ec`,
Inter + JetBrains Mono, inline shield-check SVG. → **Framework decision: Next.js (App Router, TS,
Tailwind) in a new `web/dashboard/`**, mirroring AgentTwin `apps/web`'s BFF architecture.

**Backend read surface (verified):** Membrane HTTP = gateway `:8006` + semantic `:8005`, both
**POST decision endpoints only** — no GET/list, **no auth, no CORS, no persistence**. Verdicts go
Kafka→reporter (outbound webhook/commit-status/PR comment). AgentTwin control-plane `:8080` is the
real live data source (`GET /api/v1/eval-runs[/{id}]`, `/traces`, `/simulations`, `/datasets`,
`/reviews`, `/judges`, `/agents`), authenticated (dev-login JWT / OIDC / `X-AgentTwin-Api-Key`) via
a Next.js BFF (token in HttpOnly cookie; no CORS needed). Reusable Membrane models:
`report.Report` (Code Governance), `pkg/assurance.ReleaseEvidenceV1` (Release Assurance).

### 11.2 Live / demo / unavailable data contract (per dashboard screen)

| Screen / datum | Source today | Treatment |
|---|---|---|
| Agent evaluations, baseline/candidate compare | **Live** — AgentTwin `GET /api/v1/eval-runs[/{id}]` (BFF) | live when connected; else demo fixtures from the real Phase-0 runs |
| Traces, simulations, scenarios, agents | **Live** — AgentTwin `GET /api/v1/{traces,simulations,scenarios,agents}` (BFF) | live when connected; else demo fixtures / deep links |
| Code-governance verdict (fresh decision) | **Live (decision-only)** — Membrane gateway/semantic POST | optional live call; no history |
| Code-governance history, severity breakdown | **Demo/fixture** — from real `membrane scan --json`/`Report` | labelled demo; no Membrane read API exists |
| Release decisions (PASS/WARN/BLOCK), combined gate | **Demo/fixture** — computed via `pkg/assurance` over fixtures | labelled "recommended gate policy / target" (research §5.3); no live backend |
| Any value when backend unreachable | **Unavailable** | explicit disconnected/stale/not-configured state, never faked |

Rule: every panel carries a source badge (live / demo / unavailable); demo fixtures derive from the
**real** reproduced runs (1.2.4=9/9, 1.3.0=6/9/2-critical, 1.3.1=9/9; deterministic-fake planner;
seeds shown) and are never presented as live.

### Deployment readiness policy

- Existing `docs/landing/index.html` is a marketing landing page, not the requested product dashboard.
- Existing Grafana dashboard JSON is operational observability, not the product dashboard.
- Netlify may host the static frontend; it does not host Membrane/AgentTwin Go/Python services or databases.
- Demo mode may use deterministic synthetic fixtures only when visibly labelled as demo data.
- Connected mode requires verified backend authentication, tenant/project authorization, HTTPS, CORS, secret handling and end-to-end tests.
- A successful static Netlify deploy must never be reported as connected production readiness.
- No deployment has been attempted as part of this planning update.

### 2026-10-30 — D3 Code Governance + unified view + AgentTwin Phase 2 slice (DONE)

- Status: Dashboard **D3 = DONE**; D4 next. AgentTwin integration **Phase 2 = IN_PROGRESS** (evidence builder + client port + fake landed; HTTP adapter + Saga wiring remain). Pushed to `main`.
- Files/screens delivered (dashboard): Code Governance view (`src/app/code-governance/page.tsx`) from the **real** `membrane scan` output; Security Center (`src/app/security/page.tsx`) gateway decisions; Reports (`src/app/reports/page.tsx`) with JSON/Markdown evidence export; `codeGovernance()`/`policyDecisions()` on the DataSource (demo real-scan fixture; live = honest unavailable, no Membrane read API).
- Files delivered (Go Phase 2, `pkg/assurance`): `builder.go` (`BuildReleaseEvidence` — deterministic combined-gate aggregator reusing MapEvalStatus/Decide; fail-safe, cites rules), `client.go` (`AgentTwinClient` port + deterministic `FakeClient`), `builder_test.go`.
- Data source per screen: Code Governance = **real scan** (grade F, 5 blocking, secrets masked `[MASKED:…]`) as demo fixture, live unavailable; Security = demo decisions, live unavailable; Reports = export of the (demo/live) release evidence. Overview code-findings aligned to the real scan (5 blocking).
- Real command evidence: `membrane scan --report md` → grade **F**, debt score **833.3**, byRule generic-assigned-secret×3/aws-access-key-id×1/github-token×1; captured verbatim into the fixture.
- Implementation vs design-only: dashboard views + `pkg/assurance` builder/port/fake are **implemented and tested**. The AgentTwin **HTTP adapter** (live Go client) + orchestrator Saga wiring + reporter shadow publish remain **design/next** (Phase 2/4). The dashboard already reads live AgentTwin via its own BFF (D2).
- Tests: dashboard **33 Vitest + 7 Playwright E2E** pass (added code-governance masked-secret, security deny, reports export, live-honesty/`unavailable` tests); `pkg/assurance` **90% coverage**, `task test`/`task lint` green across all modules; `npm run build` green.
- Security/privacy: code findings shown only as `[MASKED:…]` (no raw secret in UI — asserted in tests); live code/policy history returns `unavailable` not fabricated; BFF tenant from authenticated principal; builder enforces fail-safe invariants (critical/ERROR/INCOMPLETE never PASS; BLOCK/WARN cite a rule).
- Known limitations: Membrane still has no read API for code/verdict history (D0 gap) → that history is demo/fixture; live combined evidence code-verdict is a placeholder until the Go HTTP adapter + Membrane code submission are wired (Phase 2 remainder / Phase 3 persist). Netlify not yet configured (D4).
- Next action: D4 — Netlify demo deployment readiness (netlify.toml, @netlify/plugin-nextjs, env classification, headers, preview deploy, Mode A evidenced).

### 2026-10-30 — D2 functional Agent Assurance dashboard (DONE)

- Status: Dashboard **D2 = DONE**; D3 next. Pushed to `main`.
- Files/screens delivered: BFF routes `src/app/api/bff/{overview,eval-runs,eval-runs/[id],agents,simulations,traces,release/[id]}`; server client `src/lib/server/{agenttwin,allowlist,respond}.ts`; gate logic `src/lib/gate.ts` (TS mirror of pkg/assurance); views Overview, Agent Twin Lab, Simulation Center, Trace Explorer, Release Assurance (list + `/[id]` evidence detail). Agents/Simulations/Traces/Release now real views (were Planned).
- Data source per screen: **live** (AgentTwin via BFF) for eval-runs, overview, agents, simulations, traces, release detail; **demo** fixtures (real Phase-0 runs) as the default; **unavailable** state when the backend is unreachable. Every panel badged.
- API endpoints actually verified LIVE (browser→BFF→AgentTwin, server-side `X-AgentTwin-Api-Key`, loopback allowlisted): `GET /api/v1/eval-runs` (→ mapped, gate computed: 1.3.0=BLOCK, 1.3.1=PASS), `GET /api/v1/eval-runs/{id}` (per-scenario), `GET /api/v1/me`+`/projects/{id}/agents`, `GET /api/v1/simulations`, `GET /api/v1/traces`. Overview aggregated from live runs (3 runs: 2 BLOCK, 1 PASS). Release detail: BLOCK + rule AGENT_CRITICAL_SCENARIO + 9 scenarios, all from real data.
- Security: credential + base URL are server-only env (`AGENTTWIN_BASE_URL`, `AGENTTWIN_API_KEY`, `AGENTTWIN_ALLOW_LOOPBACK`, `AGENTTWIN_ALLOWED_HOSTS`); never in the browser bundle or `NEXT_PUBLIC_*`. BFF enforces an SSRF allowlist (TS mirror of pkg/assurance), bounded 8s timeout + response-size cap. Tenant/project resolved from the authenticated principal (`/api/v1/me`), never from client input.
- Tests: **27 Vitest** (gate fail-safe mapping; demo determinism/release derivation; allowlist/SSRF; states; shell) + **4 Playwright E2E** (overview+DEMO banner, full nav, release BLOCK drill-down showing AGENT_CRITICAL_SCENARIO + refund-tool-success-lie, live-disconnected honest state) — all pass on the preinstalled Chromium. `typecheck` clean; `build` green (7 dynamic BFF function routes + static pages).
- Accessibility/responsive: table captions, roles on states, source badges, keyboard nav; verified at desktop (1280) + mobile (390) via screenshots.
- Netlify config/deployment mode: not yet (D4). Env var names only (never values) as above.
- Known limitations: Code Governance history + live Release decision persistence still demo/fixture (no Membrane read API — D0 gap); live combined evidence uses code_verdict=PASS placeholder until Membrane code data is wired (D3 / AgentTwin-integration Phase 2). AgentTwin stays read-only. Netlify static demo ≠ connected production.
- Next action: D3 — Code Governance views from the CLI `Report`/scan JSON + live gateway decisions; unify code + agent under one release/correlation identity; backend tenant/authz tests; interleave AgentTwin-integration Phase 2 (orchestrator client/adapter) to back live combined evidence.

### 2026-10-30 — D1 dashboard foundation (DONE)

- Status: Dashboard **D1 = DONE**; D2 next. Pushed to `main`.
- Frontend framework/version + package manager: **Next.js 15.5.28** (App Router, patched line — avoids CVE-2025-66478), React 19.0.0, TypeScript 5.7.3, Tailwind 3.4.17; **npm** with pinned `package-lock.json`. App lives in `web/dashboard/`.
- Build command + output dir: `npm run build` (`next build`) → `web/dashboard/.next` (12 routes, all static-prerendered). Dev: `npm run dev`. Task group: `task dashboard:{install,typecheck,lint,test,build,e2e,dev,ci}`.
- Files/screens delivered: app shell (sidebar nav + header + mode toggle), 9 routes (Overview live demo-data; Settings real connectivity view; the other 7 as honest **Planned** states tagged with their phase). Design tokens reuse the Membrane teal brand + shield logo.
- Data source per screen: Overview = demo fixtures (badged DEMO) / unavailable in live-without-backend; Settings = local mode + connectivity (no secrets); others = Planned (no data yet). Every panel renders a `SourceBadge` (live/demo/unavailable); persistent DEMO banner in demo mode.
- DataSource abstraction: `DemoDataSource` (deterministic fixtures derived from the real Phase-0 runs — 1.3.0 BLOCK / 1.3.1 PASS, deterministic-fake, seeds shown) + `LiveDataSource` (same-origin BFF `/api/bff/*`, returns `unavailable` on unreachable backend, never throws). Mode default = demo (safe static deploy).
- API endpoints actually verified this phase: none live yet (D1 is the foundation; the BFF routes + AgentTwin mapping land in D2). `LiveDataSource` contract defined against `/api/bff/{overview,eval-runs,eval-runs/:id}`.
- Tests: **12 Vitest tests pass** (datasource provenance/determinism/lookup; data-state views; app-shell renders all 9 sections + demo-banner toggles with mode). `npm run typecheck` clean; `npm run build` succeeds. No E2E yet (D2+). Browser/device coverage: none yet.
- Accessibility/responsive: visible focus ring, semantic headings, `aria-current` nav, table captions, roles on state views; responsive sidebar (hidden &lt;md, header logo on mobile). Full a11y/responsive audit deferred to D5.
- Netlify config path + deployment mode: not yet (D4). Env var names only (never values): `AGENTTWIN_BASE_URL`, `AGENTTWIN_API_KEY` (server-side only; never `NEXT_PUBLIC_*`).
- Known limitations / blockers: live mode shows "not connected" until a backend + BFF exist (D2) and a reachable deployment exists (D4/Mode B); no secrets in the browser by design.
- Next action: D2 — build the BFF routes (server-side credential, allowlisted AgentTwin base URL) + real Agent Assurance views (overview, eval baseline/candidate, scenario/trace/state-diff, release detail) with Playwright journeys.

### 2026-10-30 — D0 dashboard baseline & scope (DONE)

- Status: Dashboard **D0 = DONE**; D1 next. AgentTwin integration still at Phase 1 DONE / Phase 2 next (unchanged).
- Branch: reconciled onto `main` — merged `origin/feature/agenttwin-assurance` (dashboard work-order docs) into `main`; merge commit `080eeb4`. `main` now carries Phase 0/1 + the dashboard work order; all further work on `main` per owner direction.
- Scope delivered: D0 audit only (no dashboard code). Verified frontend/API/auth/deploy reality (§11.1), framework decision (Next.js in `web/dashboard/`), and the per-screen live/demo/unavailable data contract (§11.2).
- Framework/build: none existed; chosen Next.js (App Router, TS, Tailwind) + pinned package manager; Netlify via `@netlify/plugin-nextjs` (D1+). Playwright uses pre-installed Chromium at `/opt/pw-browsers/chromium`.
- API endpoints verified (read): AgentTwin `GET /api/v1/eval-runs[/{id}]`, `/traces`, `/simulations`, `/scenarios`, `/agents`, `/datasets`, `/reviews`, `/judges` (auth: session JWT / OIDC / `X-AgentTwin-Api-Key`). Membrane: only POST `/v1/gateway/*` + `/v1/semantic/evaluate` (no auth/CORS/history).
- Tests/build: none this milestone (audit only); existing Membrane gates remain green; `pkg/assurance` builds.
- Data source per planned screen: see §11.2 table (live vs demo vs unavailable).
- Security/deploy blockers recorded: no public backend + no Membrane read API → connected Mode B blocked; Netlify Mode A (demo) is the first target. Tenant identity must stay server-side; browser never holds a privileged credential.
- Next action: D1 — scaffold `web/dashboard/` Next.js shell + nav + demo/live `DataSource` + states + tests + production build; add `dashboard:*` task group.

### 2026-10-30 — Dashboard and Netlify scope authorized

- Added `DASHBOARD_AND_NETLIFY_WORKFLOW.md` with D0–D5 deliverables, dashboard navigation/screens, source-of-truth rules, demo-mode safety, frontend security, Netlify deployment modes and acceptance criteria.
- Updated the Claude integration work order and implementation plan to make the dashboard a first-class deliverable.
- No frontend/backend implementation, test, build or deployment was performed by this documentation change.
- Next: Claude executes D0 inventory and reports the actual frontend framework/build setup, API/auth capabilities and deployment blockers before coding.

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
| Current phase | Phase 0 — baseline revalidation (**DONE**, verified) → Phase 1 next |
| Current status | `IN_PROGRESS` |
| Membrane working branch | `main` (owner directed this workstream to develop on and push to `main`, superseding the work order §1.4 `feature/agenttwin-assurance` policy) |
| Membrane HEAD (Phase 0 inspection, 2026-10-30) | `aedc1ed4014a684868ef3a339ea1d6d5df8235e0` (== `origin/main`; supersedes the 2026-09-26 `b8e8cfd…` baseline) |
| AgentTwin reference commit (verified 2026-10-30) | `1e8fd0ebc344281eff6c2d2dd45bd423b8d11a5f` (local HEAD == documented reference; read-only) |
| `feature/agenttwin-assurance` | Does **not** exist locally or on `origin` in the inspected clone; not used (owner directed `main`). |
| Code changes | None — Phase 0 is verification only (docs update to this report) |
| Contract changes | None |
| Integration tests | None yet — no integration code exists (Phase 2+) |
| Demo run in this workstream | **Reproduced live** — 1.2.4 vs 1.3.0 (regression) and 1.2.4 vs 1.3.1 (fixed); see §7 and §9 |
| Membrane quality gates | `task build/test/lint/eval`, `task lint:py/test:py`, `task build:cli` all PASS; `task helm:lint` BLOCKED (helm not installed) |
| Push status | Phase 0 evidence committed and pushed to `main` (SHA in §9 milestone entry) |

## 2. Planning artifacts published

- [x] Research and competitor analysis report copied into Membrane AI.
- [x] Detailed Claude work order created.
- [x] Implementation plan and phase acceptance criteria created.
- [x] Living progress-report contract created.
- [x] Root `CLAUDE.md` linked to scoped instructions and branch/reporting policy.
- [x] Claude revalidates current repository heads and Phase 0 evidence. (2026-10-30)
- [ ] First integration ADR and contract fixtures.
- [ ] AgentTwin client/adapter.
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
| 1 — Contract/security design | `NOT_STARTED` | ADR, versioned mapping/evidence contract, threat model, contract tests |
| 2 — Small vertical slice | `NOT_STARTED` | Membrane typed client + actual evaluation + evidence snapshot |
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

## 8. Upstream AgentTwin requests

`None confirmed yet.` Do not modify AgentTwin as a workaround. Add a request only after identifying an actual upstream gap and recording source path, required change, compatibility concern and Membrane-side fallback.

## 9. Milestone log

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
- Commit SHA containing this report: recorded on push below.
- Push status + remote branch: committed and pushed to `origin/main` (see commit SHA line appended at push time).

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

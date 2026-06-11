# HANDOVER — living session state

> **On "devam et": read this file, then do "Next up". Update this file before the session ends.**
> Keep it short and current — this is state, not history.

_Last updated: 2026-06-11 — session: P2 part 1 (semantic service scaffold)._

## ⚠️ Push is blocked on a token scope

`git push` was rejected: the classic PAT lacks the **`workflow`** scope (required because commits touch
`.github/workflows/ci.yml`). The user must edit the token at https://github.com/settings/tokens →
add **workflow** → Update. Then `git -C C:\Dev\Membrane-AI push origin main` ships everything
(**8 local commits pending**). After the first successful push, check the Actions run
(`gh run list/watch`) — CI has never executed remotely.

## Current state

**P0 ✓, P1 ✓, P2 started.** `task ci` green: 6 Go modules + Python gates (ruff, mypy --strict, pytest 16).

- **`services/semantic`** (Python 3.11 local / 3.12 CI, FastAPI, hexagonal, D-023):
  - domain: `EvaluationInput` (masked diff + gold context), `LocalAssessment` (findings + risk score),
    `should_escalate` gate policy (blocking finding OR score ≥ threshold).
  - app `EvaluateDiff`: tier-2 always; tier-3 only if escalate AND `MEMBRANE_SEMANTIC_PREMIUM_ENABLED`
    (default off). Premium augments — never replaces — local findings.
  - adapters: `HeuristicLocalModel` (deterministic stub until vLLM lands — explicitly labeled),
    `DisabledPremiumConsensus` (flag-on without real adapter ⇒ loud 503), FastAPI HTTP on `:8005`
    (`/v1/semantic/evaluate`, `/livez`, `/readyz`).
  - Real-process smoke: readyz ✓, live evaluate ✓ (3 heuristic findings, tier=local).
  - Tooling: `task setup:py | lint:py | test:py | run:semantic`; `task ci` includes Python; CI workflow
    gained a `python` job (ubuntu, 3.12).

## Next up  (P2 continuation; see docs/ROADMAP.md)

1. **Orchestrator semantic stage**: new `AnalysisStage` adapter (`internal/adapters/semanticstage`)
   calling `POST /v1/semantic/evaluate` over HTTP with the Saga's ctx deadline; needs the masked diff —
   today the orchestrator passes the RAW diff between stages; decide: analyzer response's `masked_diff`
   must flow to the semantic stage (extend `AnalysisStage` contract or chain stage outputs — design it,
   record as D-024). Config: `MEMBRANE_ORCHESTRATOR_SEMANTIC_URL` (empty = stage off). Wire + E2E.
2. **Resolver RAG injection**: orchestrator (or the semantic stage adapter) calls resolver
   `ResolveContext` and forwards `gold_context` into the evaluate request. Stub embedder caveat: org
   must be a UUID for resolver — E2E demo org should switch to a UUID.
3. Then: real tier-2 vLLM adapter, or reporter service — whichever the user prioritizes.

## Blockers / gotchas

- **Push**: see the banner above (workflow scope).
- **Stray `GOWORK`** → ALWAYS `task …` or `GOWORK=off`. **buf** = prebuilt exe. **`-race`** = CI-only.
- Python venv lives at `services/semantic/.venv` (gitignored); recreate with `task setup:py`.
- Integration tests: `MEMBRANE_ORCHESTRATOR_TEST_DSN` / `MEMBRANE_RESOLVER_TEST_DSN` =
  `postgres://membrane:membrane@localhost:5432/membrane` (needs `task dev-up` + `task migrate`).

## How to verify

- `task ci` → all green (Go + Python).
- Semantic: `task run:semantic` → `GET :8005/readyz`; POST a masked diff with "auth"/"crypto" markers
  to `/v1/semantic/evaluate` → tier=local findings; with `MEMBRANE_SEMANTIC_PREMIUM_ENABLED=1` the same
  call returns 503 (disabled premium refuses loudly — correct until a real adapter exists).

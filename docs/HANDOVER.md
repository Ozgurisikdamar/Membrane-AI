# HANDOVER — living session state

> **On "devam et": read this file, then do "Next up". Update this file before the session ends.**
> Keep it short and current — this is state, not history.

_Last updated: 2026-06-11 — session: P3 part 2 (PR-comment notifier + vLLM tier-2 adapter)._

## Push policy

The user pushes manually (`git push` in their own terminal). Check pending:
`git rev-list --count origin/main..HEAD`. `.github/` gitignored (D-026); CI source = `deploy/ci/github-ci.yml`.

## ⚠️ Standing gotchas

- **Untracked files vanished twice** earlier — commit early, commit often.
- **Stray machine `GOWORK`** → inline `GOWORK=off` only (task `env:` blocks don't override it).
- **Docker** engine needs a manual Docker Desktop start sometimes; never block on it.
- **buf** = prebuilt exe; **`-race`** = CI-only; Python venv via `task setup:py`.

## Current state

**P0 ✓ P1 ✓ P2 ✓ — P3 in flight.** `task ci` green: 8 Go modules + Python (23 pytest).

- **Reporter** now has THREE GitHub-family notifiers, all idempotent per (submission, notifier):
  webhook, commit status, and **PR comment** (`github-pr-comment`: bold title + fenced findings +
  submission footer on `issues/{pr}/comments`; skips silently without `pr_number`; enabled together
  with commit status by `MEMBRANE_REPORTER_GITHUB_TOKEN`).
- **Semantic tier-2 is real-model-ready (D-028)**: `VLLMLocalModel` (OpenAI-compatible
  `/v1/chat/completions`, strict-JSON prompt, fence-tolerant parse, severity fail-safe→warning,
  risk clamp) wrapped in `FallbackLocalModel` → heuristic degradation on ANY primary failure
  (`last_primary_error` observable). Enabled by `MEMBRANE_SEMANTIC_VLLM_URL`; without it the
  deterministic heuristic remains (current default). MockTransport tests — no network in unit tests.

## Next up  (P3 continuation; see docs/ROADMAP.md)

1. **Containerization**: Dockerfiles (multi-stage, distroless, CGO off) for the 5 Go services +
   semantic (python-slim), multi-arch-ready; add `task build:images`; wire into compose for an
   all-container dev profile. This unblocks Helm + the deployment story (D-010/D-011).
2. **Code Sweeper "Technical-Debt Report" mode** (`membrane scan --report md|html`): aggregate
   findings by rule/severity/dir + summary stats — the GTM lead magnet (report §GTM).
3. **Premium tier-3 consensus adapter** (Claude Sonnet 4.6 + Gemini): needs an API-key handling
   decision (env vs file vs vault) — record as D-029 when built; until then the flag stays off.

## How to verify

- `task ci` → all green.
- PR comment story: `services/reporter/internal/adapters/notify/githubprcomment_test.go`.
- vLLM story: `services/semantic/tests/test_vllm.py` (parse/coerce/clamp, unusable→raise,
  http error→raise, fallback degradation + recovery).
- Live vLLM dry-run (no GPU needed): point `MEMBRANE_SEMANTIC_VLLM_URL` at any OpenAI-compatible
  mock; without it, behavior is identical to before (heuristic).

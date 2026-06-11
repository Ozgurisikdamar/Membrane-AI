# HANDOVER — living session state

> **On "devam et": read this file, then do "Next up". Update this file before the session ends.**
> Keep it short and current — this is state, not history.

_Last updated: 2026-06-11 — session: P2 part 4 (commit_sha contract + GitHub commit-status notifier)._

## Push policy (current)

Everything up to `3099a9c` is on GitHub. **The user pushes manually** ("ben pushlarım") — keep
committing locally, do NOT push unless asked. `.github/` is gitignored (D-026): CI source of truth is
`deploy/ci/github-ci.yml`; the user enables it once via the GitHub web UI.

## Current state

**P0 ✓, P1 ✓, P2 ✓ (feature-complete for this phase).** `task ci` green: 7 Go modules + Python.

- **Contract enrichment (additive)**: `commit_sha`/`pr_number` now flow end-to-end —
  proto `CodeSubmission` → webhook payload → ingestion domain (`WithSource`) → `pkg/envelope`
  (`omitempty`, golden-safe) → orchestrator domain → verdict. **Correctness invariant**: the Saga
  STRIPS source coordinates before caching (`StripSource`) and RE-STAMPS them per submission
  (`StampSource`) — a cached verdict can be reused across repos/commits safely (unit-proven, incl.
  cross-repo cache-hit re-stamping).
- **GitHub commit-status notifier** in the reporter (D-025): posts `membrane-ai/governance` status
  (success/failure; needs_review→pending), 140-char description cap, token `MEMBRANE_REPORTER_GITHUB_TOKEN`,
  base URL `…_GITHUB_API_URL` (GHE/tests), silently skips reports without coordinates.
  **E2E-proven**: webhook with `commit_sha` → … → fake GitHub received
  `POST /repos/demo/repo/statuses/<sha> {"state":"failure","context":"membrane-ai/governance"}`.
- Flaky relay test fixed (deterministic polling).
- Dev stack running; binaries fresh in `%TEMP%\membrane_build\`. Board synced (7 Done, 3 In Progress).

## Next up  (P3 surfaces; see docs/ROADMAP.md — ask the user only if priorities are unclear)

1. **CLI "Code Sweeper"** (`clients/cli`, Go, static binary): scan a local repo/diff offline using the
   analyzer's detector logic (import `services/analyzer/internal/domain`? NO — internal; lift detectors
   into a shared `pkg/scan` first, then both analyzer and CLI consume it). Output: human table +
   `--json`; exit code 1 on blocking findings (CI-friendly). This is the growth-funnel hook from the
   report (§GTM) and needs no infra.
2. **PR-comment adapter** in the reporter (findings table as a PR comment when `pr_number` present).
3. **Real tier-2 vLLM adapter** in the semantic service (OpenAI-compatible /v1/completions client
   against a vLLM endpoint; config-gated like premium).

## Blockers / gotchas

- **Docker** engine flaky — start Docker Desktop manually; never block on it.
- **Stray `GOWORK`** → ALWAYS `task …` or `GOWORK=off`. **buf** = prebuilt exe. **`-race`** = CI-only.
- Python venv: `services/semantic/.venv` (`task setup:py`). Integration DSNs: see SKILLS.
- Reporter consumer groups start from the topic's beginning — use a fresh group name for tests.

## How to verify

- `task ci` → all green.
- GitHub notifier story: `services/reporter/internal/adapters/notify/githubstatus_test.go` (status
  POST shape, outcome mapping, skip-no-coords, validation, 401). Coordinate caching invariant:
  `services/orchestrator/internal/app/process_test.go::TestHandle_SourceCoordinatesStampedAndNeverCached`.
- Live recipe: dev stack + analyzer/orchestrator/ingestion + reporter with `GITHUB_TOKEN=fake`,
  `GITHUB_API_URL=<sink>`; POST webhook with `commit_sha` → sink receives the status POST.

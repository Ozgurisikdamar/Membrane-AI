# AgentTwin OpenAPI snapshot — provenance

These are **verbatim, read-only snapshots** of the AgentTwin service contracts the
Membrane Agent Assurance adapter consumes. They are checked in here only to drive
Membrane's contract-compatibility tests (`compat_test.go`). AgentTwin remains the
owner; do not edit these files to "fix" a test — refresh them from upstream.

| Field | Value |
|---|---|
| Source repository | `github.com/Ozgurisikdamar/AgentTwin` |
| Source path | `packages/contracts/openapi/{evaluation,simulation}-service.openapi.yaml` |
| Source commit | `1e8fd0ebc344281eff6c2d2dd45bd423b8d11a5f` |
| Snapshot taken | 2026-10-30 (Phase 1) |

## What Membrane consumes (the surface the compat test guards)

- **evaluation-service**: `POST /api/v1/eval-runs` (`startEvalRun` — evaluate a
  candidate version against its baseline on a pinned dataset suite) and
  `GET /api/v1/eval-runs/{eval_run_id}`.
- **simulation-service**: `POST /internal/v1/simulation-pairs`
  (`startSimulationPair` — baseline + candidate in one transaction over one
  pinned suite: same scenario versions, twin definitions and seeds; idempotent,
  returns `PAIR_CONFLICT` with the existing run IDs).
- **auth** (both, at the control-plane edge): bearer token or `X-AgentTwin-Api-Key`,
  with `Idempotency-Key` on mutating requests.

## How to refresh

Re-copy both files from AgentTwin at the new commit, update the table above, run
`task test` in `pkg`, and reconcile any compat-test failure as a real upstream
contract change (record it in `docs/agenttwin-integration/PROGRESS.md`).

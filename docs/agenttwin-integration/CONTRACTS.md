# Agent Assurance contracts (v1)

The authoritative, versioned contracts that bind Membrane Code Governance to
AgentTwin Agent Assurance. This document is the human-readable spec; the
machine-readable truth is:

- Go types + rules: `pkg/assurance/` (`contract.go`, `status.go`, `allowlist.go`).
- JSON Schemas: `pkg/assurance/contracts/{evidence,mapping}.v1.schema.json`.
- Fixtures (incl. the real Phase-0 run): `pkg/assurance/contracts/fixtures/`.
- Pinned AgentTwin OpenAPI snapshot: `pkg/assurance/contracts/agenttwin/` (+ `PROVENANCE.md`).
- Tests: `pkg/assurance/*_test.go` (status, validators, allowlist, compatibility).

All records carry `schema_version` (currently `1.0.0`). A consumer rejects an
unknown version rather than mis-reading it.

## 1. Correlation contract (`CorrelationContext`, v1.0.0)

Carried Membrane→AgentTwin for linkage only — **never** as authorization.
Required: `membrane_organization_id`, `membrane_project_id`,
`membrane_code_submission_id`, `membrane_repository`, `membrane_commit_sha`,
`membrane_release_id`, `membrane_agent_id`, `membrane_agent_version`,
`membrane_correlation_id`. Optional: `membrane_pull_request_number`,
`membrane_change_set_id`, W3C `traceparent`/`tracestate`. Business release ID and
trace ID are distinct.

## 2. Mapping record (`MappingRecordV1`, v1.0.0)

Minimal tenant mapping Membrane↔AgentTwin. Required: `mapping_id`,
`membrane_org_id`, `membrane_project_id`, `agenttwin_org_id`,
`agenttwin_project_id`, `agenttwin_base_url`, `credential_ref`, `status`,
`actor`, `audit_ref` (+ `created_at`/`updated_at`). Security invariants:
`agenttwin_base_url` must pass the destination allowlist; `credential_ref` is a
secret-manager key name, never an inline secret.

## 3. Release evidence record (`ReleaseEvidenceV1`, v1.0.0)

The minimal immutable snapshot Membrane persists to make the combined decision
auditable. References + summaries only — no raw spans/prompts/tool payloads/PII.
Links the Membrane side (org/project/repo/commit/submission + code verdict
summary), the agent identity (id/baseline/candidate + manifest/prompt hashes),
the AgentTwin side (project, evaluation run ID, simulation pair IDs, scenario
selection, run seed, engine/evaluator versions), the outcome (overall status,
per-scenario status/severity, critical failures with evidence URLs, optional
judge metadata) and the combined gate (decision + triggered rules), plus a
`snapshot_hash` and audit ref. **Fail-safe invariants** (enforced in code and
schema): (1) a PASS decision requires `overall_status == COMPLETED`; (2) a
BLOCK/WARN decision must cite ≥1 triggered rule.

## 4. Status mapping (AgentTwin → integration status → gate)

`MapEvalStatus` then `Decide` (`pkg/assurance/status.go`). A critical failure
dominates every other signal.

| AgentTwin result | Integration status | Gate (default policy) |
|---|---|---|
| COMPLETED, no critical, no regression, no error/incomplete | `COMPLETED` | `PASS` (iff code/policy also pass) |
| COMPLETED, ≥1 new critical scenario failure | `COMPLETED_WITH_CRITICAL_FAILURE` | `BLOCK` |
| COMPLETED, ≥1 non-critical regression | `COMPLETED_WITH_REGRESSION` | `WARN` (→ `BLOCK` if `RegressionBlocks`) |
| COMPLETED but ≥1 errored scenario | `ERROR` | `ERROR` (never PASS) |
| COMPLETED but ≥1 incomplete scenario | `INCOMPLETE` | `INCOMPLETE` (never PASS) |
| ERROR / FAILED run | `ERROR` | `ERROR` |
| CANCELLED run | `INCOMPLETE` | `INCOMPLETE` |
| PENDING / QUEUED / RUNNING / unknown | `PENDING` | `INCOMPLETE` |
| optional assurance not configured | `NOT_CONFIGURED` | `WARN` (→ `BLOCK` if `AssuranceRequired`) |
| assurance required but mapping unavailable | `BLOCKED` | `BLOCK` |

Worked example (Phase-0 live run): 1.3.0 had 2 new critical failures →
`COMPLETED_WITH_CRITICAL_FAILURE` → `BLOCK`; 1.3.1 had 0 new critical / 0
regression → `COMPLETED` → `PASS` candidate. See
`fixtures/evidence-1.3.0-block.json` and `evidence-1.3.1-pass.json`.

## 5. Gate rule catalog (deterministic, IMPLEMENTATION_PLAN §5)

Rule IDs an evidence record cites in `triggered_rules` (the full evaluator lands
in Phase 3; these are the stable IDs the contract reserves):

`CODE_CRITICAL` (BLOCK), `AGENT_CRITICAL_SCENARIO` (BLOCK),
`UNSAFE_IRREVERSIBLE_ACTION` (BLOCK), `OUTCOME_CONTRADICTION` (BLOCK),
`TENANT_BOUNDARY_VIOLATION` (BLOCK), `POLICY_BYPASS` (BLOCK),
`AGENT_EVAL_INCOMPLETE` (BLOCK, or explicit WARN only when policy says optional —
never silent PASS), `NONCRITICAL_REGRESSION` (WARN), `UNCALIBRATED_JUDGE`
(WARN + human review), `COST_LATENCY_REGRESSION` (WARN), `COVERAGE_GAP`
(WARN/BLOCK by policy), `OPTIONAL_ASSURANCE_UNAVAILABLE` (WARN/N/A with reason).
No numeric risk index may override a hard BLOCK.

## 6. AgentTwin surface consumed (verified Phase 0, pinned Phase 1)

- `POST /api/v1/eval-runs` (`startEvalRun`) + `GET /api/v1/eval-runs/{id}` — evaluate candidate vs baseline on a pinned dataset suite.
- `POST /internal/v1/simulation-pairs` (`startSimulationPair`) — baseline+candidate in one transaction, one pinned suite/seed; idempotent (`PAIR_CONFLICT`).
- Auth at the control-plane edge: bearer token or `X-AgentTwin-Api-Key`; `Idempotency-Key` on mutations. (No gRPC; OTLP only for traces.)

## 7. Versioning

Breaking changes bump the `schema_version` and keep the prior schema file for
compatibility. `compat_test.go` guards schema↔struct field parity, the
schema_version constants, the fail-safe invariants on fixtures, and the pinned
AgentTwin OpenAPI surface.

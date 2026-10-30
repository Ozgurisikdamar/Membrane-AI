# Threat model — Membrane ↔ AgentTwin Agent Assurance integration

Phase 1 deliverable (work order §7.2, IMPLEMENTATION_PLAN §6). Scope: the
Membrane→AgentTwin HTTP integration and the evidence Membrane persists. It does
not re-model AgentTwin's own internals (owned upstream).

## Assets

- Tenant isolation between Membrane projects and their mapped AgentTwin projects.
- AgentTwin service credentials (API keys) and the Membrane↔AgentTwin mapping.
- Release-evidence integrity (a decision a reviewer can trust).
- Developer source/secrets (must never leak into traces, evidence or logs).

## Trust boundaries

1. Membrane orchestrator → AgentTwin HTTP edge (authenticated, allowlisted).
2. AgentTwin responses + any imported OpenAPI/scenario/trace content → **untrusted
   input** to Membrane (parse defensively; never treat as authorization).
3. Operator configuration (mapping, allowlist, secret refs) → trusted, server-side.

## Threats & mitigations

| # | Threat | Mitigation (Phase 1 contract / Phase 2 adapter) |
|---|---|---|
| T1 | **SSRF** via an attacker-influenced AgentTwin base URL (internal metadata, link-local, redirects). | Base URL is operator config only, checked by `Allowlist.CheckURL` (scheme http/https, no embedded creds, host in allowlist; loopback only in dev). **Phase 2:** re-check every redirect target, pin dial to resolved allowlisted IPs (DNS-rebinding defense), refuse private/link-local ranges. |
| T2 | **Cross-tenant leakage** — one Membrane project reads another's AgentTwin data. | Tenant identity comes from the authenticated mapping, never from agent attributes or trace values (ADR-0001 #6). Mapping validation requires both sides' org+project. **Phase 2:** per-mapping credential; negative tenant-isolation tests. |
| T3 | **Secret leakage** — a token stored in the mapping or printed in logs/evidence. | `credential_ref` must be a secret-manager key name; `MappingRecordV1.Validate` rejects inline secrets (`atk_`, `sk-`, `ghp_`, `AKIA`, bearer, PEM, long opaque blobs). Evidence stores references/summaries only. **Phase 2:** redact `Authorization`/`X-AgentTwin-Api-Key` and tool/prompt content in logs & spans. |
| T4 | **Spoofed success** — an agent claims success the business state contradicts. | Deterministic AgentTwin outcome is authoritative; `MapEvalStatus` surfaces `COMPLETED_WITH_CRITICAL_FAILURE`/`ERROR`; `ReleaseEvidenceV1.Validate` forbids PASS on any non-clean status. |
| T5 | **Scores hiding critical failures.** | No numeric score in the contract may override a hard rule; a critical failure deterministically dominates regression/score in `MapEvalStatus`; PASS requires clean `COMPLETED`. |
| T6 | **Duplicate side effects** on retry (duplicate eval pairs, releases, comments). | Mutating calls use `Idempotency-Key`; pair creation is AgentTwin's single pinned-pair op (`PAIR_CONFLICT` returns existing IDs). **Phase 2:** retry only safe/idempotent ops; never blind-retry simulation mutations. |
| T7 | **Unexplained decisions.** | `ReleaseEvidenceV1.Validate` requires a BLOCK/WARN to cite ≥1 triggered rule; evidence lists per-scenario status, severity and source links. |
| T8 | **Unavailability misread as success** — timeout, judge error, missing side, budget exhaustion → PASS. | Explicit non-passing statuses (`ERROR`, `INCOMPLETE`, `PENDING`, `BLOCKED`, `NOT_CONFIGURED`) that never map to PASS; `Decide` returns `ERROR`/`INCOMPLETE`, never PASS, for them. |
| T9 | **Malicious imported content** — prompt injection in retrieved docs/manifests/OpenAPI the adapter ingests. | Treated as untrusted data, never instructions/authorization. **Phase 2:** validate responses against the checked-in versioned contract; bound response size; reject unexpected content types. |
| T10 | **Telemetry/transport weakening.** | Non-loopback destinations must use HTTPS/TLS (`Allowlist.CheckURL`). **Phase 2:** bounded connection/read deadlines, response-size caps, concurrency/back-pressure; telemetry failure must not crash the code-analysis workflow. |
| T11 | **Contract drift** — AgentTwin changes a route/field Membrane depends on. | A pinned AgentTwin OpenAPI snapshot (`contracts/agenttwin/`, provenance recorded) + `compat_test.go` fail on drift; refreshes are a deliberate, reviewed step logged in `PROGRESS.md`. |
| T12 | **Unaudited privileged changes** — mapping/policy/override edits. | `MappingRecordV1` carries `actor` + `audit_ref` (required). **Phase 2:** audit mapping/policy changes, approvals, overrides and report deliveries. |

## Residual risk / open items (Phase 1)

- Whether AgentTwin supports a production-safe **cross-project service
  credential**, or requires an upstream addition, is unresolved (demo uses a
  per-project API key). Recorded as an open verification item in `PROGRESS.md`.
- DNS-rebinding and redirect re-validation are designed here but implemented with
  the HTTP adapter in Phase 2.

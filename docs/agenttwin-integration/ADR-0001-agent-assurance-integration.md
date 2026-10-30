# ADR-0001 — AgentTwin Agent Assurance integration boundary & contract

- **Status:** Accepted (Phase 1, 2026-10-30)
- **Scope:** AgentTwin integration workstream only (`docs/agenttwin-integration/`). Supplements, does not replace, the root ADR log (`docs/DECISIONS.md`, see D-037).
- **Context source:** work order `CLAUDE.md` §3–§4, `IMPLEMENTATION_PLAN.md` §4–§6, research report §4, Phase 0 evidence in `PROGRESS.md`.

## Context

Membrane AI (Code Governance) and AgentTwin (Agent Assurance) are two independent
products with separate runtimes, data stores and event buses (Membrane:
Go/Kafka-Redpanda/pgvector; AgentTwin: Go+Python/RabbitMQ/pgvector). The goal is
a thin, secure, evidence-driven integration that turns a code change into a
combined, auditable PASS/WARN/BLOCK release decision — without merging the two
systems. Phase 0 verified AgentTwin exposes HTTP OpenAPI contracts
(`POST /api/v1/eval-runs`, `POST /internal/v1/simulation-pairs`) with edge auth
(`X-AgentTwin-Api-Key` / bearer) and `Idempotency-Key` on mutations, and that the
refund demo reproduces deterministically (1.2.4 → 1.3.0 regression → 1.3.1 fix).

## Decisions

1. **Integrate over versioned HTTP, not shared infrastructure.** Membrane calls
   AgentTwin through a typed, authenticated, idempotent HTTP client. No shared
   database, no cross-writes, and **no Kafka↔RabbitMQ bridge** in the first
   integration (work order §1.2). Asynchronous callbacks/outbox are deferred
   until measured latency/reliability requires them.

2. **Ownership boundary.** Membrane owns org/project identity, code submissions,
   code-governance verdicts and **the combined release decision**. AgentTwin owns
   manifests, traces, scenarios, twins, simulations, evaluations and judges, and
   the detailed evidence. Membrane stores only *references and summaries* of
   AgentTwin evidence — never raw spans, prompts, tool payloads or PII.

3. **Adapter boundary inside Membrane = the orchestrator (existing composition
   root), not a new microservice.** The orchestrator already drives the
   analyzer→resolver→semantic→verdict Saga and owns the verdict; the Agent
   Assurance call is a new optional stage/adapter behind an injected port there.
   A new service is not justified yet (no independent scaling/security/runtime
   need). This is revisited only if those needs appear (work order §3.2, §7.1).
   **Phase 1 ships the pure contract only** (`pkg/assurance`); the orchestrator
   port + HTTP adapter land in Phase 2.

4. **The contract is a shared, versioned, pure package: `pkg/assurance`.** It
   holds the correlation contract, the tenant-mapping and release-evidence
   records (each `schema_version`-stamped), the AgentTwin-status → integration-status
   mapping, and the fail-safe gate consequence. It has no I/O so both the
   orchestrator (produce/decide) and the reporter (publish) can depend on it.
   JSON Schemas + fixtures + a pinned AgentTwin OpenAPI snapshot live under
   `pkg/assurance/contracts/` and are covered by compatibility tests.

5. **Credentials by reference, destinations by allowlist.** The mapping stores a
   secret-manager *key name* (`credential_ref`), never a plaintext token
   (enforced by `MappingRecordV1.Validate`). The AgentTwin base URL is operator
   configuration checked against a server-side `Allowlist` (SSRF defense);
   loopback is allowed only in explicit dev configuration.

6. **Correlation ≠ tenancy.** The correlation contract carries
   `membrane.*` IDs + W3C `traceparent` into AgentTwin for linkage only. Tenant
   identity derives from the authenticated mapping, never from an agent attribute
   or a trace value (work order §3.4).

7. **Idempotency & pair pinning.** Mutating calls use `Idempotency-Key`. The
   baseline/candidate pair is created by AgentTwin's single pinned-pair operation
   (`/internal/v1/simulation-pairs`, `PAIR_CONFLICT` returns the existing run
   IDs) — never two unrelated requests — preserving one suite/seed so the agent
   version is the only variable.

8. **Fail-safe, explainable gate.** The combined gate is not a weighted average.
   A critical failure, ERROR, INCOMPLETE, PENDING or BLOCKED status can never be
   rendered as PASS, and a BLOCK/WARN must cite a triggered rule — both invariants
   are enforced in code (`ReleaseEvidenceV1.Validate`) and in the JSON Schema.

## Consequences

- The first demo stays CLI/report-first with a deep link to AgentTwin's UI; no
  unified web product is built in this slice.
- Membrane gains one small pure package now and one orchestrator port+adapter in
  Phase 2; the reporter is reused unchanged for SCM publishing in Phase 4.
- AgentTwin is untouched (read-only). If an upstream gap appears (e.g. a
  cross-project service credential), it is recorded under *Upstream AgentTwin
  requests* in `PROGRESS.md`, not worked around by editing AgentTwin.

## Alternatives rejected

- **Monorepo / shared DB / shared event bus** — explicitly out of scope; couples
  two independently-evolving systems and violates work order §1.2.
- **New dedicated "assurance" microservice now** — premature; the orchestrator
  composition root is the narrower, reversible boundary.
- **Storing full AgentTwin evidence in Membrane** — violates data minimization;
  Membrane keeps references + a minimal snapshot and deep-links the rest.

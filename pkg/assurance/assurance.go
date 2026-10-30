// Package assurance holds the versioned, transport-agnostic contract that binds
// MEMBRANE.AI Code Governance to the AgentTwin Agent Assurance capability into a
// single, auditable release decision.
//
// This package is the Phase-1 contract for the AgentTwin integration
// (docs/agenttwin-integration/). It is pure: no HTTP, no database, no I/O. It
// defines the cross-product correlation contract, the minimal release-evidence
// and tenant-mapping records Membrane persists, the mapping from an AgentTwin
// evaluation result to an integration status, and the deterministic, fail-safe
// gate consequence of that status. The HTTP client/port/adapter that actually
// talks to AgentTwin lands in Phase 2 behind a Membrane port and consumes these
// types; it is deliberately not here.
//
// Design rules this package enforces (work order §3–§4, ADR-0001):
//   - Tenant identity comes from the authenticated mapping, never from agent
//     attributes or trace content. Business release ID and trace ID are distinct.
//   - Evidence is minimal: external IDs, versions/hashes, verdict, severity
//     summary, scenario links and pinned seed/version metadata — never raw
//     spans, prompts, tool payloads or PII.
//   - The gate is explainable and fail-safe: a critical failure, an ERROR or an
//     INCOMPLETE result can never be rendered as PASS, and no numeric score may
//     override a hard rule.
package assurance

// Schema versions for the checked-in contracts. These travel in the records
// themselves (schema_version) so a consumer can reject an unknown version
// rather than silently mis-reading it. Bump the version on any breaking change
// and keep the previous schema file for compatibility (ENGINEERING-STANDARDS,
// IMPLEMENTATION_PLAN §4).
const (
	// EvidenceSchemaVersion is the current release-evidence record version.
	EvidenceSchemaVersion = "1.0.0"
	// MappingSchemaVersion is the current tenant-mapping record version.
	MappingSchemaVersion = "1.0.0"
	// CorrelationSchemaVersion is the current cross-product correlation version.
	CorrelationSchemaVersion = "1.0.0"
)

// Op is the stable operation prefix for typed errors raised by this package.
const Op = "assurance"

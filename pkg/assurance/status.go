package assurance

import "github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"

// IntegrationStatus is Membrane's normalized view of an AgentTwin paired
// evaluation, derived from AgentTwin's own run status plus the baseline/candidate
// comparison counts. It is the stable vocabulary the release aggregator reasons
// about (work order §3.5, §4.3).
type IntegrationStatus string

// Integration statuses. Only Completed can lead to PASS; every other terminal
// status is explicitly non-passing.
const (
	// StatusCompleted: the paired evaluation finished and no critical scenario
	// failed. The candidate MAY pass if the code/policy gates also pass.
	StatusCompleted IntegrationStatus = "COMPLETED"
	// StatusCompletedWithCriticalFailure: a critical deterministic scenario
	// failed. Always BLOCK.
	StatusCompletedWithCriticalFailure IntegrationStatus = "COMPLETED_WITH_CRITICAL_FAILURE"
	// StatusCompletedWithRegression: a non-critical scenario regressed. WARN or
	// BLOCK by explicit project policy.
	StatusCompletedWithRegression IntegrationStatus = "COMPLETED_WITH_REGRESSION"
	// StatusError: an evaluation worker/provider error or malformed evidence.
	// Never PASS.
	StatusError IntegrationStatus = "ERROR"
	// StatusIncomplete: a missing run, a missing side, or a mismatched
	// suite/seed. Never PASS.
	StatusIncomplete IntegrationStatus = "INCOMPLETE"
	// StatusPending: still running/queued. No final PASS.
	StatusPending IntegrationStatus = "PENDING"
	// StatusNotConfigured: agent assurance is optional and not configured for
	// this project. N/A or WARN, explicitly.
	StatusNotConfigured IntegrationStatus = "NOT_CONFIGURED"
	// StatusBlocked: agent assurance is required but its mapping is unavailable.
	// Never PASS.
	StatusBlocked IntegrationStatus = "BLOCKED"
)

// GateDecision is the explainable, fail-safe outcome of the combined release
// gate (work order §4.1). INCOMPLETE and ERROR are deliberately distinct from
// BLOCK/WARN/PASS: they are not release decisions and must never equal PASS.
type GateDecision string

// Gate decisions.
const (
	// DecisionPass: all required evidence present and all required checks green.
	DecisionPass GateDecision = "PASS"
	// DecisionWarn: a non-critical concern that does not block the release.
	DecisionWarn GateDecision = "WARN"
	// DecisionBlock: a hard rule tripped; the release is blocked.
	DecisionBlock GateDecision = "BLOCK"
	// DecisionIncomplete: required evidence is missing; visible, never PASS.
	DecisionIncomplete GateDecision = "INCOMPLETE"
	// DecisionError: evaluation errored; visible, never PASS.
	DecisionError GateDecision = "ERROR"
)

// EvalResult is the minimal, already-parsed shape of an AgentTwin paired
// evaluation that the status mapper needs. It is populated by the Phase-2
// adapter from the validated AgentTwin response; the mapper itself stays pure.
type EvalResult struct {
	// RawStatus is AgentTwin's own run status, e.g. "COMPLETED", "ERROR",
	// "PENDING", "RUNNING", "CANCELLED". Compared case-insensitively upstream;
	// callers pass it verbatim.
	RawStatus string
	// NewCriticalFailures counts scenarios that became a critical failure in the
	// candidate relative to the baseline.
	NewCriticalFailures int
	// Regressed counts non-critical scenarios that regressed.
	Regressed int
	// Incomplete counts scenarios with missing/partial evidence.
	Incomplete int
	// Errored counts scenarios that errored during evaluation.
	Errored int
}

// Policy carries the few project-configurable knobs the mapping allows. Defaults
// are the safe choices: a non-critical regression warns (not blocks) and an
// unconfigured optional integration warns.
type Policy struct {
	// RegressionBlocks promotes a non-critical regression from WARN to BLOCK.
	RegressionBlocks bool
	// AssuranceRequired means a missing/unconfigured mapping must not PASS.
	AssuranceRequired bool
}

// MapEvalStatus classifies an AgentTwin paired evaluation into an
// IntegrationStatus. It is deterministic and total; an unrecognized raw status
// is treated as PENDING (never PASS). A critical failure dominates every other
// signal (work order §4.2: scores cannot hide critical failures).
func MapEvalStatus(r EvalResult) IntegrationStatus {
	switch normalizeStatus(r.RawStatus) {
	case "COMPLETED":
		switch {
		case r.Errored > 0:
			return StatusError
		case r.Incomplete > 0:
			return StatusIncomplete
		case r.NewCriticalFailures > 0:
			return StatusCompletedWithCriticalFailure
		case r.Regressed > 0:
			return StatusCompletedWithRegression
		default:
			return StatusCompleted
		}
	case "ERROR", "FAILED":
		return StatusError
	case "CANCELLED":
		return StatusIncomplete
	case "PENDING", "QUEUED", "RUNNING", "":
		return StatusPending
	default:
		// Unknown upstream status: safest is a non-final, non-passing state.
		return StatusPending
	}
}

// Decide maps an IntegrationStatus to the fail-safe gate consequence under the
// given policy. It never returns PASS for anything but a clean COMPLETED, and
// the code-governance verdict is applied separately by the aggregator — this is
// the agent-assurance half of the combined decision (work order §4.1, §4.3).
func Decide(s IntegrationStatus, p Policy) GateDecision {
	switch s {
	case StatusCompleted:
		return DecisionPass
	case StatusCompletedWithCriticalFailure, StatusBlocked:
		return DecisionBlock
	case StatusCompletedWithRegression:
		if p.RegressionBlocks {
			return DecisionBlock
		}
		return DecisionWarn
	case StatusError:
		return DecisionError
	case StatusIncomplete:
		return DecisionIncomplete
	case StatusPending:
		return DecisionIncomplete
	case StatusNotConfigured:
		if p.AssuranceRequired {
			return DecisionBlock
		}
		return DecisionWarn
	default:
		// Unknown status must never pass.
		return DecisionIncomplete
	}
}

// Passing reports whether a gate decision permits release on its own. Only PASS
// passes; WARN is advisory but not blocking, yet it is not "passing" in the
// strict sense the aggregator needs, so it is reported separately via the
// decision itself. Passing exists so callers cannot accidentally treat ERROR or
// INCOMPLETE as success.
func (d GateDecision) Passing() bool { return d == DecisionPass }

// Validate rejects an unknown gate decision so a malformed value cannot slip
// through as a non-blocking state.
func (d GateDecision) Validate() error {
	switch d {
	case DecisionPass, DecisionWarn, DecisionBlock, DecisionIncomplete, DecisionError:
		return nil
	default:
		return errs.Validation(Op+".GateDecision.Validate", "unknown gate decision: "+string(d), nil)
	}
}

// Validate rejects an unknown integration status.
func (s IntegrationStatus) Validate() error {
	switch s {
	case StatusCompleted, StatusCompletedWithCriticalFailure, StatusCompletedWithRegression,
		StatusError, StatusIncomplete, StatusPending, StatusNotConfigured, StatusBlocked:
		return nil
	default:
		return errs.Validation(Op+".IntegrationStatus.Validate", "unknown integration status: "+string(s), nil)
	}
}

// normalizeStatus upper-cases and trims an upstream status for comparison
// without pulling in strings.ToUpper edge cases on non-ASCII (statuses are
// ASCII by contract).
func normalizeStatus(s string) string {
	b := make([]byte, 0, len(s))
	// trim surrounding spaces
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	for i := start; i < end; i++ {
		c := s[i]
		if c >= 'a' && c <= 'z' {
			c -= 'a' - 'A'
		}
		b = append(b, c)
	}
	return string(b)
}

package assurance

import "testing"

// Exhaustive matrix for the combined gate: agent integration status × code
// verdict → decision + the rule it must cite. Proves the fail-safe invariants
// (work order §4): a critical failure, ERROR, INCOMPLETE, PENDING or BLOCKED
// status can never be PASS, and a hard code BLOCK always blocks.
func TestCombinedGateMatrix(t *testing.T) {
	t.Parallel()
	type want struct {
		decision GateDecision
		rule     string // a rule that must appear (empty = no specific rule)
	}
	tests := []struct {
		status IntegrationStatus
		code   GateDecision
		policy Policy
		want   want
	}{
		{StatusCompleted, DecisionPass, Policy{}, want{DecisionPass, ""}},
		{StatusCompleted, DecisionWarn, Policy{}, want{DecisionWarn, "COVERAGE_GAP"}},
		{StatusCompleted, DecisionBlock, Policy{}, want{DecisionBlock, "CODE_CRITICAL"}},
		{StatusCompletedWithCriticalFailure, DecisionPass, Policy{}, want{DecisionBlock, "AGENT_CRITICAL_SCENARIO"}},
		{StatusCompletedWithCriticalFailure, DecisionBlock, Policy{}, want{DecisionBlock, "CODE_CRITICAL"}},
		{StatusCompletedWithRegression, DecisionPass, Policy{}, want{DecisionWarn, "NONCRITICAL_REGRESSION"}},
		{StatusCompletedWithRegression, DecisionPass, Policy{RegressionBlocks: true}, want{DecisionBlock, "NONCRITICAL_REGRESSION"}},
		{StatusError, DecisionPass, Policy{}, want{DecisionError, "AGENT_EVAL_INCOMPLETE"}},
		{StatusError, DecisionBlock, Policy{}, want{DecisionBlock, "CODE_CRITICAL"}},
		{StatusIncomplete, DecisionPass, Policy{}, want{DecisionIncomplete, "AGENT_EVAL_INCOMPLETE"}},
		{StatusPending, DecisionPass, Policy{}, want{DecisionIncomplete, "AGENT_EVAL_INCOMPLETE"}},
		{StatusBlocked, DecisionPass, Policy{}, want{DecisionBlock, "AGENT_EVAL_INCOMPLETE"}},
		{StatusNotConfigured, DecisionPass, Policy{}, want{DecisionWarn, "OPTIONAL_ASSURANCE_UNAVAILABLE"}},
		{StatusNotConfigured, DecisionPass, Policy{AssuranceRequired: true}, want{DecisionBlock, "OPTIONAL_ASSURANCE_UNAVAILABLE"}},
	}
	for _, tc := range tests {
		got := combinedGate(tc.status, tc.code, tc.policy)
		if got.decision != tc.want.decision {
			t.Errorf("combinedGate(%s, %s, %+v) = %s, want %s", tc.status, tc.code, tc.policy, got.decision, tc.want.decision)
		}
		if tc.want.rule != "" && !contains(got.rules, tc.want.rule) {
			t.Errorf("combinedGate(%s, %s) rules %v missing %q", tc.status, tc.code, got.rules, tc.want.rule)
		}
		// Fail-safe: only a clean COMPLETED with a non-blocking code verdict may PASS.
		if got.decision == DecisionPass && (tc.status != StatusCompleted || tc.code == DecisionBlock) {
			t.Errorf("combinedGate(%s, %s) returned PASS but inputs are not clean", tc.status, tc.code)
		}
	}
}

// Every rule the catalog reserves must be referenced somewhere in the gate
// logic (guards against a dead/renamed rule id drifting from CONTRACTS.md §5).
func TestGateRuleCatalogReferenced(t *testing.T) {
	t.Parallel()
	seen := map[string]bool{}
	statuses := []IntegrationStatus{
		StatusCompleted, StatusCompletedWithCriticalFailure, StatusCompletedWithRegression,
		StatusError, StatusIncomplete, StatusPending, StatusBlocked, StatusNotConfigured,
	}
	for _, s := range statuses {
		for _, code := range []GateDecision{DecisionPass, DecisionWarn, DecisionBlock} {
			for _, p := range []Policy{{}, {RegressionBlocks: true}, {AssuranceRequired: true}} {
				for _, r := range combinedGate(s, code, p).rules {
					seen[r] = true
				}
			}
		}
	}
	for _, rule := range []string{
		"CODE_CRITICAL", "AGENT_CRITICAL_SCENARIO", "NONCRITICAL_REGRESSION",
		"AGENT_EVAL_INCOMPLETE", "OPTIONAL_ASSURANCE_UNAVAILABLE", "COVERAGE_GAP",
	} {
		if !seen[rule] {
			t.Errorf("rule %q is never produced by combinedGate", rule)
		}
	}
}

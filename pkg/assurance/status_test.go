package assurance

import "testing"

func TestMapEvalStatus(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   EvalResult
		want IntegrationStatus
	}{
		{"clean completed", EvalResult{RawStatus: "COMPLETED"}, StatusCompleted},
		{"critical dominates regression", EvalResult{RawStatus: "COMPLETED", NewCriticalFailures: 2, Regressed: 1}, StatusCompletedWithCriticalFailure},
		{"regression only", EvalResult{RawStatus: "COMPLETED", Regressed: 1}, StatusCompletedWithRegression},
		{"errored scenario never passes", EvalResult{RawStatus: "COMPLETED", Errored: 1}, StatusError},
		{"incomplete scenario never passes", EvalResult{RawStatus: "COMPLETED", Incomplete: 1}, StatusIncomplete},
		{"errored beats critical", EvalResult{RawStatus: "COMPLETED", Errored: 1, NewCriticalFailures: 3}, StatusError},
		{"raw error", EvalResult{RawStatus: "ERROR"}, StatusError},
		{"raw failed", EvalResult{RawStatus: "failed"}, StatusError},
		{"raw cancelled", EvalResult{RawStatus: "CANCELLED"}, StatusIncomplete},
		{"raw pending", EvalResult{RawStatus: "PENDING"}, StatusPending},
		{"raw running", EvalResult{RawStatus: "  Running "}, StatusPending},
		{"empty", EvalResult{RawStatus: ""}, StatusPending},
		{"unknown upstream is pending", EvalResult{RawStatus: "WAT"}, StatusPending},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := MapEvalStatus(tc.in); got != tc.want {
				t.Fatalf("MapEvalStatus(%+v) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestDecide(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		status IntegrationStatus
		policy Policy
		want   GateDecision
	}{
		{"completed passes", StatusCompleted, Policy{}, DecisionPass},
		{"critical blocks", StatusCompletedWithCriticalFailure, Policy{}, DecisionBlock},
		{"regression warns by default", StatusCompletedWithRegression, Policy{}, DecisionWarn},
		{"regression blocks under policy", StatusCompletedWithRegression, Policy{RegressionBlocks: true}, DecisionBlock},
		{"error is error", StatusError, Policy{}, DecisionError},
		{"incomplete is incomplete", StatusIncomplete, Policy{}, DecisionIncomplete},
		{"pending is incomplete", StatusPending, Policy{}, DecisionIncomplete},
		{"not configured warns when optional", StatusNotConfigured, Policy{}, DecisionWarn},
		{"not configured blocks when required", StatusNotConfigured, Policy{AssuranceRequired: true}, DecisionBlock},
		{"blocked blocks", StatusBlocked, Policy{}, DecisionBlock},
		{"unknown never passes", IntegrationStatus("???"), Policy{}, DecisionIncomplete},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := Decide(tc.status, tc.policy)
			if got != tc.want {
				t.Fatalf("Decide(%q, %+v) = %q, want %q", tc.status, tc.policy, got, tc.want)
			}
			// Fail-safe cross-check: only a clean COMPLETED may ever pass.
			if got.Passing() && tc.status != StatusCompleted {
				t.Fatalf("Decide(%q) passed but status is not COMPLETED", tc.status)
			}
		})
	}
}

func TestGateDecisionValidate(t *testing.T) {
	t.Parallel()
	for _, d := range []GateDecision{DecisionPass, DecisionWarn, DecisionBlock, DecisionIncomplete, DecisionError} {
		if err := d.Validate(); err != nil {
			t.Errorf("valid decision %q rejected: %v", d, err)
		}
	}
	if err := GateDecision("NOPE").Validate(); err == nil {
		t.Error("unknown decision accepted")
	}
}

func TestIntegrationStatusValidate(t *testing.T) {
	t.Parallel()
	valid := []IntegrationStatus{
		StatusCompleted, StatusCompletedWithCriticalFailure, StatusCompletedWithRegression,
		StatusError, StatusIncomplete, StatusPending, StatusNotConfigured, StatusBlocked,
	}
	for _, s := range valid {
		if err := s.Validate(); err != nil {
			t.Errorf("valid status %q rejected: %v", s, err)
		}
	}
	if err := IntegrationStatus("NOPE").Validate(); err == nil {
		t.Error("unknown status accepted")
	}
}

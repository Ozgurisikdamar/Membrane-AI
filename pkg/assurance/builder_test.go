package assurance

import (
	"context"
	"testing"
	"time"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
)

func corr() CorrelationContext {
	return CorrelationContext{
		SchemaVersion:    CorrelationSchemaVersion,
		OrganizationID:   "org-demo",
		ProjectID:        "proj-demo",
		CodeSubmissionID: "sub-1",
		Repository:       "Ozgurisikdamar/demo",
		CommitSHA:        "aedc1ed4014a684868ef3a339ea1d6d5df8235e0",
		ReleaseID:        "rel-1",
		AgentID:          "support-refund-agent",
		AgentVersion:     "1.3.0",
		CorrelationID:    "corr-1",
	}
}

// The real reproduced runs, as the deterministic fake would return them.
func ref130() EvalResultRef {
	return EvalResultRef{
		EvaluationRunID:   "01a18c71-faf5-70b5-a459-7052a0b23f37",
		AgentTwinProject:  "01a18c71-9d01-7b04-932b-2c14a50d0953",
		BaselineVersion:   "1.2.4",
		CandidateVersion:  "1.3.0",
		Seed:              "42",
		RawStatus:         "COMPLETED",
		Counts:            EvalResult{NewCriticalFailures: 2, Regressed: 1},
		ScenarioSelection: []string{"refund-tool-success-lie", "refund-timeout-after-mutation"},
		PerScenario: []ScenarioOutcome{
			{Scenario: "refund-tool-success-lie", Status: "NEW_CRITICAL_FAILURE", Severity: "critical"},
			{Scenario: "refund-timeout-after-mutation", Status: "NEW_CRITICAL_FAILURE", Severity: "critical"},
		},
		SimulationPairIDs: []string{"01a18c71-fa92", "01a18c71-faaa"},
	}
}

func ref131() EvalResultRef {
	return EvalResultRef{
		EvaluationRunID:   "01a18c73-c690-73bd-8d2c-711c059ba059",
		AgentTwinProject:  "01a18c71-9d01-7b04-932b-2c14a50d0953",
		BaselineVersion:   "1.2.4",
		CandidateVersion:  "1.3.1",
		Seed:              "42",
		RawStatus:         "COMPLETED",
		Counts:            EvalResult{},
		ScenarioSelection: []string{"refund-happy-path"},
		PerScenario:       []ScenarioOutcome{{Scenario: "refund-happy-path", Status: "UNCHANGED", Severity: "high"}},
	}
}

func baseInput(ref EvalResultRef, code GateDecision) BuildInput {
	return BuildInput{
		Correlation:  corr(),
		CodeVerdict:  code,
		CodeFindings: map[string]int{"blocking": 0},
		CodeRuleset:  "scan-v1",
		Agent:        ref,
		Now:          time.Date(2026, 10, 30, 10, 54, 2, 0, time.UTC),
		SnapshotHash: "sha256:abc",
	}
}

func TestBuildReleaseEvidence_CriticalBlocks(t *testing.T) {
	t.Parallel()
	ev, err := BuildReleaseEvidence(baseInput(ref130(), DecisionPass))
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if ev.OverallStatus != StatusCompletedWithCriticalFailure {
		t.Fatalf("status = %s", ev.OverallStatus)
	}
	if ev.GateDecision != DecisionBlock {
		t.Fatalf("gate = %s, want BLOCK", ev.GateDecision)
	}
	if !contains(ev.TriggeredRules, "AGENT_CRITICAL_SCENARIO") {
		t.Fatalf("missing AGENT_CRITICAL_SCENARIO: %v", ev.TriggeredRules)
	}
	if len(ev.CriticalFailures) != 2 {
		t.Fatalf("critical failures = %d, want 2", len(ev.CriticalFailures))
	}
}

func TestBuildReleaseEvidence_CleanPasses(t *testing.T) {
	t.Parallel()
	ev, err := BuildReleaseEvidence(baseInput(ref131(), DecisionPass))
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if ev.GateDecision != DecisionPass {
		t.Fatalf("gate = %s, want PASS", ev.GateDecision)
	}
}

func TestBuildReleaseEvidence_CodeBlockForcesBlock(t *testing.T) {
	t.Parallel()
	// Agent is clean (1.3.1) but code governance blocks → combined BLOCK.
	ev, err := BuildReleaseEvidence(baseInput(ref131(), DecisionBlock))
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if ev.GateDecision != DecisionBlock {
		t.Fatalf("gate = %s, want BLOCK", ev.GateDecision)
	}
	if !contains(ev.TriggeredRules, "CODE_CRITICAL") {
		t.Fatalf("missing CODE_CRITICAL: %v", ev.TriggeredRules)
	}
}

func TestBuildReleaseEvidence_ErrorNeverPasses(t *testing.T) {
	t.Parallel()
	ref := ref131()
	ref.RawStatus = "ERROR"
	ev, err := BuildReleaseEvidence(baseInput(ref, DecisionPass))
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if ev.GateDecision == DecisionPass {
		t.Fatalf("ERROR status produced PASS")
	}
}

func TestFakeClientRoundTrip(t *testing.T) {
	t.Parallel()
	c := NewFakeClient(ref130(), ref131())
	got, err := c.GetEvalResult(context.Background(), ref130().EvaluationRunID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.CandidateVersion != "1.3.0" {
		t.Fatalf("candidate = %s", got.CandidateVersion)
	}
	if _, err := c.GetEvalResult(context.Background(), "nope"); errs.KindOf(err) != errs.KindNotFound {
		t.Fatalf("expected not-found, got %v", err)
	}
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

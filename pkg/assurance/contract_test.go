package assurance

import (
	"testing"
	"time"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
)

func validCorrelation() CorrelationContext {
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

func TestCorrelationValidate(t *testing.T) {
	t.Parallel()
	if err := validCorrelation().Validate(); err != nil {
		t.Fatalf("valid correlation rejected: %v", err)
	}

	bad := validCorrelation()
	bad.SchemaVersion = "9.9.9"
	if err := bad.Validate(); errs.KindOf(err) != errs.KindValidation {
		t.Errorf("wrong schema version should be a validation error, got %v", err)
	}

	missing := validCorrelation()
	missing.CommitSHA = ""
	if err := missing.Validate(); err == nil {
		t.Error("missing commit sha accepted")
	}

	nothex := validCorrelation()
	nothex.CommitSHA = "not-a-sha-zzz"
	if err := nothex.Validate(); err == nil {
		t.Error("non-hex commit sha accepted")
	}
}

func validMapping() MappingRecordV1 {
	now := time.Date(2026, 10, 30, 10, 50, 0, 0, time.UTC)
	return MappingRecordV1{
		SchemaVersion:      MappingSchemaVersion,
		MappingID:          "map-1",
		MembraneOrgID:      "org-demo",
		MembraneProjectID:  "proj-demo",
		AgentTwinOrgID:     "at-org",
		AgentTwinProjectID: "at-proj",
		AgentTwinBaseURL:   "https://assurance.agenttwin.example",
		CredentialRef:      "agenttwin/demo-api-key",
		Status:             "active",
		CreatedAt:          now,
		UpdatedAt:          now,
		Actor:              "owner@demo",
		AuditRef:           "audit-1",
	}
}

func TestMappingValidate(t *testing.T) {
	t.Parallel()
	allow := NewAllowlist(false, "assurance.agenttwin.example")

	if err := validMapping().Validate(allow); err != nil {
		t.Fatalf("valid mapping rejected: %v", err)
	}

	badVer := validMapping()
	badVer.SchemaVersion = "0.9.0"
	if err := badVer.Validate(allow); err == nil {
		t.Error("wrong schema version accepted")
	}

	missing := validMapping()
	missing.AgentTwinProjectID = "  "
	if err := missing.Validate(allow); err == nil {
		t.Error("missing agenttwin_project_id accepted")
	}

	offlist := validMapping()
	offlist.AgentTwinBaseURL = "https://evil.example"
	if err := offlist.Validate(allow); err == nil {
		t.Error("base url off the allowlist accepted")
	}

	inline := validMapping()
	inline.CredentialRef = "atk_demo0000_supersecretvalue"
	if err := inline.Validate(allow); err == nil {
		t.Error("inline secret in credential_ref accepted")
	}
}

func validEvidence() ReleaseEvidenceV1 {
	return ReleaseEvidenceV1{
		SchemaVersion:      EvidenceSchemaVersion,
		ReleaseID:          "rel-1",
		CorrelationID:      "corr-1",
		CreatedAt:          time.Date(2026, 10, 30, 10, 54, 2, 0, time.UTC),
		OrganizationID:     "org-demo",
		ProjectID:          "proj-demo",
		Repository:         "Ozgurisikdamar/demo",
		CodeSubmissionID:   "sub-1",
		CommitSHA:          "aedc1ed4014a684868ef3a339ea1d6d5df8235e0",
		CodeVerdict:        DecisionPass,
		RulesetVersion:     "scan-v1",
		FindingsSummary:    map[string]int{"blocking": 0},
		AgentID:            "support-refund-agent",
		BaselineVersion:    "1.2.4",
		CandidateVersion:   "1.3.0",
		AgentTwinProjectID: "at-proj",
		EvaluationRunID:    "run-1",
		ScenarioSelection:  []string{"refund-tool-success-lie"},
		RunSeed:            "42",
		OverallStatus:      StatusCompletedWithCriticalFailure,
		PerScenario:        []ScenarioOutcome{{Scenario: "refund-tool-success-lie", Status: "NEW_CRITICAL_FAILURE", Severity: "critical"}},
		GateDecision:       DecisionBlock,
		TriggeredRules:     []string{"AGENT_CRITICAL_SCENARIO"},
		SnapshotHash:       "sha256:abc",
	}
}

func TestEvidenceValidate(t *testing.T) {
	t.Parallel()
	if err := validEvidence().Validate(); err != nil {
		t.Fatalf("valid evidence rejected: %v", err)
	}

	// Required-field coverage: each cleared string field must fail.
	for _, mut := range []struct {
		name string
		fn   func(*ReleaseEvidenceV1)
	}{
		{"release_id", func(e *ReleaseEvidenceV1) { e.ReleaseID = "" }},
		{"correlation_id", func(e *ReleaseEvidenceV1) { e.CorrelationID = "" }},
		{"commit_sha", func(e *ReleaseEvidenceV1) { e.CommitSHA = "" }},
		{"evaluation_run_id", func(e *ReleaseEvidenceV1) { e.EvaluationRunID = "" }},
		{"snapshot_hash", func(e *ReleaseEvidenceV1) { e.SnapshotHash = "" }},
		{"created_at", func(e *ReleaseEvidenceV1) { e.CreatedAt = time.Time{} }},
		{"scenario_selection", func(e *ReleaseEvidenceV1) { e.ScenarioSelection = nil }},
	} {
		e := validEvidence()
		mut.fn(&e)
		if err := e.Validate(); err == nil {
			t.Errorf("cleared %s but evidence still valid", mut.name)
		}
	}
}

func TestEvidenceFailSafeInvariants(t *testing.T) {
	t.Parallel()

	// Invariant 1: PASS cannot accompany a non-clean status.
	badPass := validEvidence()
	badPass.GateDecision = DecisionPass // status is still COMPLETED_WITH_CRITICAL_FAILURE
	if err := badPass.Validate(); err == nil {
		t.Error("PASS with a critical-failure status was accepted (fail-safe breach)")
	}

	// A genuinely clean PASS is fine.
	cleanPass := validEvidence()
	cleanPass.OverallStatus = StatusCompleted
	cleanPass.GateDecision = DecisionPass
	cleanPass.TriggeredRules = nil
	if err := cleanPass.Validate(); err != nil {
		t.Errorf("clean PASS rejected: %v", err)
	}

	// Invariant 2: a BLOCK/WARN must cite a rule.
	unexplained := validEvidence()
	unexplained.TriggeredRules = nil // BLOCK with no rule
	if err := unexplained.Validate(); err == nil {
		t.Error("BLOCK with no triggered rule was accepted (unexplained decision)")
	}

	// ERROR/INCOMPLETE never equal PASS.
	for _, s := range []IntegrationStatus{StatusError, StatusIncomplete, StatusPending, StatusBlocked} {
		e := validEvidence()
		e.OverallStatus = s
		e.GateDecision = DecisionPass
		e.TriggeredRules = nil
		if err := e.Validate(); err == nil {
			t.Errorf("status %q with PASS was accepted", s)
		}
	}
}

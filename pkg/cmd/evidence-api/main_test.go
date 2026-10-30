package main

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/assurance"
)

const testKey = "test-api-key"

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// blockEvidence is a valid BLOCK record (critical code + agent findings).
func blockEvidence() assurance.ReleaseEvidenceV1 {
	return assurance.ReleaseEvidenceV1{
		SchemaVersion:      assurance.EvidenceSchemaVersion,
		ReleaseID:          "rel-1",
		CorrelationID:      "corr-1",
		CreatedAt:          time.Date(2026, 10, 30, 10, 0, 0, 0, time.UTC),
		OrganizationID:     "org-demo",
		ProjectID:          "proj-demo",
		Repository:         "Ozgurisikdamar/demo",
		CodeSubmissionID:   "sub-1",
		CommitSHA:          "aedc1ed4014a684868ef3a339ea1d6d5df8235e0",
		CodeVerdict:        assurance.DecisionBlock,
		RulesetVersion:     "scan-v1",
		FindingsSummary:    map[string]int{"blocking": 1},
		AgentID:            "support-refund-agent",
		BaselineVersion:    "1.2.4",
		CandidateVersion:   "1.3.0",
		AgentTwinProjectID: "at-proj",
		EvaluationRunID:    "run-1",
		ScenarioSelection:  []string{"refund-tool-success-lie"},
		RunSeed:            "42",
		OverallStatus:      assurance.StatusCompletedWithCriticalFailure,
		PerScenario:        []assurance.ScenarioOutcome{{Scenario: "refund-tool-success-lie", Status: "NEW_CRITICAL_FAILURE", Severity: "critical"}},
		GateDecision:       assurance.DecisionBlock,
		TriggeredRules:     []string{"CODE_CRITICAL", "AGENT_CRITICAL_SCENARIO"},
		SnapshotHash:       "sha256:abc",
	}
}

// passEvidence is a valid PASS record (clean code + completed eval).
func passEvidence() assurance.ReleaseEvidenceV1 {
	ev := blockEvidence()
	ev.CodeVerdict = assurance.DecisionPass
	ev.FindingsSummary = map[string]int{"blocking": 0}
	ev.CandidateVersion = "1.3.1"
	ev.ScenarioSelection = []string{"refund-happy-path"}
	ev.OverallStatus = assurance.StatusCompleted
	ev.PerScenario = []assurance.ScenarioOutcome{{Scenario: "refund-happy-path", Status: "UNCHANGED", Severity: "info"}}
	ev.GateDecision = assurance.DecisionPass
	ev.TriggeredRules = nil
	ev.SnapshotHash = "sha256:def"
	return ev
}

// seededServer returns a server backed by an in-memory store holding one
// BLOCK record for proj-a and one PASS record for proj-b.
func seededServer(t *testing.T) *server {
	t.Helper()
	store := assurance.NewMemStore()
	block := blockEvidence()
	block.ReleaseID = "rel-block"
	block.ProjectID = "proj-a"
	block.CreatedAt = time.Date(2026, 10, 30, 9, 0, 0, 0, time.UTC)
	pass := passEvidence()
	pass.ReleaseID = "rel-pass"
	pass.ProjectID = "proj-b"
	pass.CreatedAt = time.Date(2026, 10, 30, 10, 0, 0, 0, time.UTC)
	for _, ev := range []assurance.ReleaseEvidenceV1{block, pass} {
		if err := store.Put(ev); err != nil {
			t.Fatalf("seed Put %s: %v", ev.ReleaseID, err)
		}
	}
	return &server{store: store, apiKey: testKey, log: discardLogger()}
}

func do(t *testing.T, h http.HandlerFunc, method, target, key string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, nil)
	if key != "" {
		req.Header.Set("X-Membrane-Api-Key", key)
	}
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec
}

func TestAuth_RejectsMissingAndWrongKey(t *testing.T) {
	t.Parallel()
	s := seededServer(t)
	for _, key := range []string{"", "wrong-key"} {
		rec := do(t, s.auth(s.list), http.MethodGet, "/v1/releases", key)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("key=%q: status = %d, want 401", key, rec.Code)
		}
	}
}

func TestList_ReturnsNewestFirst(t *testing.T) {
	t.Parallel()
	s := seededServer(t)
	rec := do(t, s.auth(s.list), http.MethodGet, "/v1/releases", testKey)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var out []assurance.ReleaseEvidenceV1
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("got %d records, want 2", len(out))
	}
	if out[0].ReleaseID != "rel-pass" {
		t.Fatalf("newest = %q, want rel-pass", out[0].ReleaseID)
	}
}

func TestList_ProjectFilter(t *testing.T) {
	t.Parallel()
	s := seededServer(t)
	rec := do(t, s.auth(s.list), http.MethodGet, "/v1/releases?project=proj-a", testKey)
	var out []assurance.ReleaseEvidenceV1
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(out) != 1 || out[0].ReleaseID != "rel-block" {
		t.Fatalf("filter proj-a = %+v, want only rel-block", releaseIDs(out))
	}
}

func TestGet_RealCombinedVerdictAndNotFound(t *testing.T) {
	t.Parallel()
	s := seededServer(t)
	// Found: canonical JSON carries the real combined decision, not a placeholder.
	// The mux fills PathValue in production; set it directly for the unit test.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/releases/rel-block", nil)
	req.Header.Set("X-Membrane-Api-Key", testKey)
	req.SetPathValue("id", "rel-block")
	s.auth(s.get)(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var ev assurance.ReleaseEvidenceV1
	if err := json.Unmarshal(rec.Body.Bytes(), &ev); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if ev.GateDecision != assurance.DecisionBlock || ev.CodeVerdict != assurance.DecisionBlock {
		t.Fatalf("gate=%s code=%s, want BLOCK/BLOCK", ev.GateDecision, ev.CodeVerdict)
	}

	// Not found.
	nf := httptest.NewRecorder()
	nreq := httptest.NewRequest(http.MethodGet, "/v1/releases/missing", nil)
	nreq.Header.Set("X-Membrane-Api-Key", testKey)
	nreq.SetPathValue("id", "missing")
	s.auth(s.get)(nf, nreq)
	if nf.Code != http.StatusNotFound {
		t.Fatalf("missing status = %d, want 404", nf.Code)
	}
}

func releaseIDs(xs []assurance.ReleaseEvidenceV1) []string {
	out := make([]string, len(xs))
	for i, x := range xs {
		out[i] = x.ReleaseID
	}
	return out
}

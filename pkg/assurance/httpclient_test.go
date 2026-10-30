package assurance

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
)

const evalDetail130JSON = `{
  "run": {
    "id": "01a18c71-faf5-70b5-a459-7052a0b23f37",
    "project_id": "01a18c71-9d01-7b04-932b-2c14a50d0953",
    "agent_name": "support-refund-agent",
    "baseline_version": "1.2.4",
    "candidate_version": "1.3.0",
    "seed": 42,
    "status": "COMPLETED",
    "counts": {"NEW_CRITICAL_FAILURE": 2, "REGRESSED": 1, "IMPROVED": 0, "UNCHANGED": 6, "INCOMPLETE": 0},
    "baseline_run_id": "01a18c71-fa92",
    "candidate_run_id": "01a18c71-faaa"
  },
  "summary": {"new_critical_failures": [{"scenario_name": "refund-tool-success-lie", "expectations": ["outcomeVerified"]}], "regressed": ["refund-happy-path"]},
  "cases": [
    {"scenario_name": "refund-tool-success-lie", "severity": "critical", "classification": "NEW_CRITICAL_FAILURE"},
    {"scenario_name": "refund-happy-path", "severity": "high", "classification": "REGRESSED"}
  ]
}`

func devAllowlist() *Allowlist { return NewAllowlist(true) } // loopback for httptest

func newTestClient(t *testing.T, base string) *HTTPClient {
	t.Helper()
	c, err := NewHTTPClient(HTTPConfig{BaseURL: base, APIKey: "atk_test", Allowlist: devAllowlist()})
	if err != nil {
		t.Fatalf("NewHTTPClient: %v", err)
	}
	return c
}

func TestHTTPClient_Happy(t *testing.T) {
	t.Parallel()
	var gotKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("X-AgentTwin-Api-Key")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(evalDetail130JSON))
	}))
	defer srv.Close()

	got, err := newTestClient(t, srv.URL).GetEvalResult(context.Background(), "01a18c71-faf5-70b5-a459-7052a0b23f37")
	if err != nil {
		t.Fatalf("GetEvalResult: %v", err)
	}
	if gotKey != "atk_test" {
		t.Errorf("api key header = %q", gotKey)
	}
	if got.CandidateVersion != "1.3.0" || got.Seed != "42" {
		t.Errorf("mapping wrong: %+v", got)
	}
	if got.Counts.NewCriticalFailures != 2 || got.Counts.Regressed != 1 {
		t.Errorf("counts wrong: %+v", got.Counts)
	}
	if len(got.PerScenario) != 2 || len(got.SimulationPairIDs) != 2 {
		t.Errorf("scenarios/pairs wrong: %+v", got)
	}
	// The mapped result drives the same fail-safe gate as the live dashboard.
	if MapEvalStatus(got.Counts.withStatus(got.RawStatus)) != StatusCompletedWithCriticalFailure {
		t.Errorf("status mapping wrong")
	}
}

func TestHTTPClient_StatusErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		code int
		kind errs.Kind
	}{
		{"unauthorized", http.StatusUnauthorized, errs.KindUnauthorized},
		{"forbidden", http.StatusForbidden, errs.KindUnauthorized},
		{"not found", http.StatusNotFound, errs.KindNotFound},
		{"conflict", http.StatusConflict, errs.KindConflict},
		{"rate limited", http.StatusTooManyRequests, errs.KindUnavailable},
		{"server error", http.StatusInternalServerError, errs.KindUnavailable},
		{"bad gateway", http.StatusBadGateway, errs.KindUnavailable},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.code)
			}))
			defer srv.Close()
			_, err := newTestClient(t, srv.URL).GetEvalResult(context.Background(), "run-1")
			if errs.KindOf(err) != tc.kind {
				t.Fatalf("status %d: kind = %v, want %v", tc.code, errs.KindOf(err), tc.kind)
			}
		})
	}
}

func TestHTTPClient_MalformedJSON(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("{ not json"))
	}))
	defer srv.Close()
	_, err := newTestClient(t, srv.URL).GetEvalResult(context.Background(), "run-1")
	if errs.KindOf(err) != errs.KindInternal {
		t.Fatalf("malformed: kind = %v", errs.KindOf(err))
	}
}

func TestHTTPClient_Timeout(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(200 * time.Millisecond)
		_, _ = w.Write([]byte(evalDetail130JSON))
	}))
	defer srv.Close()
	c, err := NewHTTPClient(HTTPConfig{BaseURL: srv.URL, APIKey: "k", Allowlist: devAllowlist(), Timeout: 20 * time.Millisecond})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	if _, err := c.GetEvalResult(context.Background(), "run-1"); errs.KindOf(err) != errs.KindUnavailable {
		t.Fatalf("timeout: kind = %v, want unavailable", errs.KindOf(err))
	}
}

func TestHTTPClient_ConfigGuards(t *testing.T) {
	t.Parallel()
	if _, err := NewHTTPClient(HTTPConfig{APIKey: "k", Allowlist: devAllowlist()}); err == nil {
		t.Error("missing base URL accepted")
	}
	if _, err := NewHTTPClient(HTTPConfig{BaseURL: "https://x.example", Allowlist: devAllowlist()}); err == nil {
		t.Error("missing API key accepted")
	}
	if _, err := NewHTTPClient(HTTPConfig{BaseURL: "https://x.example", APIKey: "k"}); err == nil {
		t.Error("missing allowlist accepted")
	}
	// Base URL off the allowlist is rejected (SSRF guard).
	if _, err := NewHTTPClient(HTTPConfig{BaseURL: "https://evil.example", APIKey: "k", Allowlist: NewAllowlist(false, "good.example")}); err == nil {
		t.Error("off-allowlist base URL accepted")
	}
}

func TestHTTPClient_BuildEvidenceFromLiveShape(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(evalDetail130JSON))
	}))
	defer srv.Close()
	ref, err := newTestClient(t, srv.URL).GetEvalResult(context.Background(), "run-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	ev, err := BuildReleaseEvidence(BuildInput{
		Correlation:  corr(),
		CodeVerdict:  DecisionPass,
		CodeFindings: map[string]int{"blocking": 0},
		CodeRuleset:  "scan-v1",
		Agent:        ref,
		Now:          time.Now().UTC(),
		SnapshotHash: "sha256:x",
	})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if ev.GateDecision != DecisionBlock {
		t.Fatalf("combined gate = %s, want BLOCK", ev.GateDecision)
	}
}

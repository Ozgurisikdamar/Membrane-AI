package assurance

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
)

func evWith(gate GateDecision, status IntegrationStatus, rules []string) ReleaseEvidenceV1 {
	e := validEvidence() // BLOCK / critical by default
	e.GateDecision = gate
	e.OverallStatus = status
	e.TriggeredRules = rules
	if gate == DecisionPass {
		e.TriggeredRules = nil
	}
	return e
}

func TestMapSCMStatus(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		gate        GateDecision
		status      IntegrationStatus
		mode        SCMMode
		wantOutcome SCMOutcome
		wantState   string
		titleHas    string
	}{
		{"pass", DecisionPass, StatusCompleted, ModeEnforce, OutcomeSuccess, "success", "passed"},
		{"block enforce", DecisionBlock, StatusCompletedWithCriticalFailure, ModeEnforce, OutcomeFailure, "failure", "blocked"},
		{"block shadow", DecisionBlock, StatusCompletedWithCriticalFailure, ModeShadow, OutcomeNeutral, "pending", "shadow mode"},
		{"warn", DecisionWarn, StatusCompletedWithRegression, ModeEnforce, OutcomeNeutral, "pending", "warnings"},
		{"incomplete", DecisionIncomplete, StatusIncomplete, ModeEnforce, OutcomeNeutral, "pending", "incomplete"},
		{"error", DecisionError, StatusError, ModeEnforce, OutcomeNeutral, "pending", "errored"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			st := MapSCMStatus(evWith(tc.gate, tc.status, []string{"AGENT_CRITICAL_SCENARIO"}), tc.mode)
			if st.Outcome != tc.wantOutcome || st.State != tc.wantState {
				t.Fatalf("outcome/state = %s/%s, want %s/%s", st.Outcome, st.State, tc.wantOutcome, tc.wantState)
			}
			if !strings.Contains(strings.ToLower(st.Title), strings.ToLower(tc.titleHas)) {
				t.Errorf("title %q missing %q", st.Title, tc.titleHas)
			}
			// Fail-safe: only PASS is ever a success.
			if st.Outcome == OutcomeSuccess && tc.gate != DecisionPass {
				t.Errorf("non-PASS gate %s produced success", tc.gate)
			}
		})
	}
}

func TestIdempotentPublisher(t *testing.T) {
	t.Parallel()
	fake := &FakePublisher{}
	p := NewIdempotentPublisher(fake)
	s := MapSCMStatus(evWith(DecisionBlock, StatusCompletedWithCriticalFailure, []string{"AGENT_CRITICAL_SCENARIO"}), ModeShadow)

	d1, err := p.Publish(context.Background(), s)
	if err != nil || !d1 {
		t.Fatalf("first publish: delivered=%v err=%v", d1, err)
	}
	d2, err := p.Publish(context.Background(), s) // duplicate delivery
	if err != nil || d2 {
		t.Fatalf("duplicate publish: delivered=%v err=%v (want not delivered)", d2, err)
	}
	if len(fake.Published) != 1 {
		t.Fatalf("inner published %d times, want 1 (idempotent)", len(fake.Published))
	}
	// A changed state re-publishes.
	s2 := MapSCMStatus(evWith(DecisionPass, StatusCompleted, nil), ModeShadow)
	s2.Repository, s2.CommitSHA = s.Repository, s.CommitSHA
	if d3, _ := p.Publish(context.Background(), s2); !d3 {
		t.Fatalf("changed state should re-publish")
	}
	if len(fake.Published) != 2 {
		t.Fatalf("inner published %d times, want 2", len(fake.Published))
	}
}

func TestGitHubStatusPublisher(t *testing.T) {
	t.Parallel()
	status := MapSCMStatus(evWith(DecisionBlock, StatusCompletedWithCriticalFailure, []string{"AGENT_CRITICAL_SCENARIO"}), ModeEnforce)
	status.Repository, status.CommitSHA = "owner/repo", "deadbeef"

	t.Run("created", func(t *testing.T) {
		t.Parallel()
		var gotAuth, gotState, gotCtx string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotAuth = r.Header.Get("Authorization")
			var p ghStatusPayload
			_ = json.NewDecoder(r.Body).Decode(&p)
			gotState, gotCtx = p.State, p.Context
			w.WriteHeader(http.StatusCreated)
		}))
		defer srv.Close()
		pub, err := NewGitHubStatusPublisher(srv.URL, "ghp_test")
		if err != nil {
			t.Fatalf("new: %v", err)
		}
		if err := pub.Publish(context.Background(), status); err != nil {
			t.Fatalf("publish: %v", err)
		}
		if gotAuth != "Bearer ghp_test" || gotState != "failure" || gotCtx != StatusContext {
			t.Errorf("auth=%q state=%q ctx=%q", gotAuth, gotState, gotCtx)
		}
	})

	errCases := []struct {
		name string
		code int
		kind errs.Kind
	}{
		{"revoked token", http.StatusUnauthorized, errs.KindUnauthorized},
		{"forbidden", http.StatusForbidden, errs.KindUnauthorized},
		{"not found", http.StatusNotFound, errs.KindNotFound},
		{"server error", http.StatusInternalServerError, errs.KindUnavailable},
	}
	for _, tc := range errCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.code)
			}))
			defer srv.Close()
			pub, _ := NewGitHubStatusPublisher(srv.URL, "ghp_test")
			if err := pub.Publish(context.Background(), status); errs.KindOf(err) != tc.kind {
				t.Fatalf("status %d: kind = %v, want %v", tc.code, errs.KindOf(err), tc.kind)
			}
		})
	}

	t.Run("missing token + coords", func(t *testing.T) {
		t.Parallel()
		if _, err := NewGitHubStatusPublisher("https://api.github.com", ""); err == nil {
			t.Error("missing token accepted")
		}
		pub, _ := NewGitHubStatusPublisher("https://api.github.com", "t")
		bad := status
		bad.Repository = ""
		if err := pub.Publish(context.Background(), bad); errs.KindOf(err) != errs.KindValidation {
			t.Errorf("missing coords kind = %v", errs.KindOf(err))
		}
	})
}

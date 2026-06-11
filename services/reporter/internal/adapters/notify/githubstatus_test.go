package notify_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	pkgerrs "github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
	"github.com/Ozgurisikdamar/Membrane-AI/services/reporter/internal/adapters/notify"
	"github.com/Ozgurisikdamar/Membrane-AI/services/reporter/internal/domain"
)

func ghReport(outcome domain.Outcome) domain.Report {
	return domain.Report{
		SubmissionID: "s-1",
		Outcome:      outcome,
		Title:        "MEMBRANE.AI ✗ rejected — 2 finding(s) must be fixed",
		Repository:   "Ozgurisikdamar/Membrane-AI",
		CommitSHA:    "abc123def",
	}
}

func TestGitHubStatus_PostsStatus(t *testing.T) {
	var gotPath, gotAuth string
	var gotBody map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	n := notify.NewGitHubStatus(srv.URL, "tok-123")
	if err := n.Notify(context.Background(), ghReport(domain.OutcomeFailure)); err != nil {
		t.Fatal(err)
	}
	if gotPath != "/repos/Ozgurisikdamar/Membrane-AI/statuses/abc123def" {
		t.Fatalf("path = %s", gotPath)
	}
	if gotAuth != "Bearer tok-123" {
		t.Fatalf("auth = %s", gotAuth)
	}
	if gotBody["state"] != "failure" || gotBody["context"] != "membrane-ai/governance" {
		t.Fatalf("body = %v", gotBody)
	}
}

func TestGitHubStatus_OutcomeMapping(t *testing.T) {
	var state string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b map[string]string
		_ = json.NewDecoder(r.Body).Decode(&b)
		state = b["state"]
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()
	n := notify.NewGitHubStatus(srv.URL, "t")

	for outcome, want := range map[domain.Outcome]string{
		domain.OutcomeSuccess: "success",
		domain.OutcomeFailure: "failure",
		domain.OutcomeNeutral: "pending",
	} {
		if err := n.Notify(context.Background(), ghReport(outcome)); err != nil {
			t.Fatal(err)
		}
		if state != want {
			t.Fatalf("outcome %s → state %s, want %s", outcome, state, want)
		}
	}
}

func TestGitHubStatus_SkipsWithoutCoordinates(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	defer srv.Close()
	n := notify.NewGitHubStatus(srv.URL, "t")

	r := ghReport(domain.OutcomeFailure)
	r.CommitSHA = ""
	if err := n.Notify(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("must skip silently when commit sha is missing")
	}
}

func TestGitHubStatus_BadRepoIsValidation(t *testing.T) {
	n := notify.NewGitHubStatus("http://127.0.0.1:1", "t")
	r := ghReport(domain.OutcomeFailure)
	r.Repository = "just-a-name"
	err := n.Notify(context.Background(), r)
	if pkgerrs.KindOf(err) != pkgerrs.KindValidation {
		t.Fatalf("kind = %v, want validation", pkgerrs.KindOf(err))
	}
}

func TestGitHubStatus_Non201IsUnavailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()
	err := notify.NewGitHubStatus(srv.URL, "bad").Notify(context.Background(), ghReport(domain.OutcomeSuccess))
	if pkgerrs.KindOf(err) != pkgerrs.KindUnavailable {
		t.Fatalf("kind = %v, want unavailable", pkgerrs.KindOf(err))
	}
	if !strings.Contains(err.Error(), "401") {
		t.Fatalf("err = %v", err)
	}
}

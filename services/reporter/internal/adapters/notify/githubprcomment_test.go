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

func prReport() domain.Report {
	return domain.Report{
		SubmissionID: "s-1",
		Outcome:      domain.OutcomeFailure,
		Title:        "MEMBRANE.AI ✗ rejected — 1 finding(s) must be fixed",
		Body:         "• [analyzer] aws-access-key-id (blocking): rotate it",
		Repository:   "Ozgurisikdamar/Membrane-AI",
		CommitSHA:    "abc123",
		PRNumber:     42,
	}
}

func TestPRComment_PostsMarkdown(t *testing.T) {
	var gotPath string
	var gotBody map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	if err := notify.NewGitHubPRComment(srv.URL, "tok").Notify(context.Background(), prReport()); err != nil {
		t.Fatal(err)
	}
	if gotPath != "/repos/Ozgurisikdamar/Membrane-AI/issues/42/comments" {
		t.Fatalf("path = %s", gotPath)
	}
	body := gotBody["body"]
	for _, want := range []string{"**MEMBRANE.AI ✗ rejected", "```", "aws-access-key-id", "submission `s-1`"} {
		if !strings.Contains(body, want) {
			t.Fatalf("comment missing %q:\n%s", want, body)
		}
	}
}

func TestPRComment_SkipsWithoutPR(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	defer srv.Close()
	n := notify.NewGitHubPRComment(srv.URL, "tok")

	r := prReport()
	r.PRNumber = 0
	if err := n.Notify(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	r = prReport()
	r.Repository = ""
	if err := n.Notify(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("must skip silently without PR coordinates")
	}
}

func TestPRComment_BadRepoIsValidation(t *testing.T) {
	r := prReport()
	r.Repository = "no-slash"
	err := notify.NewGitHubPRComment("http://127.0.0.1:1", "t").Notify(context.Background(), r)
	if pkgerrs.KindOf(err) != pkgerrs.KindValidation {
		t.Fatalf("kind = %v, want validation", pkgerrs.KindOf(err))
	}
}

func TestPRComment_Non201IsUnavailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()
	err := notify.NewGitHubPRComment(srv.URL, "t").Notify(context.Background(), prReport())
	if pkgerrs.KindOf(err) != pkgerrs.KindUnavailable {
		t.Fatalf("kind = %v, want unavailable", pkgerrs.KindOf(err))
	}
}

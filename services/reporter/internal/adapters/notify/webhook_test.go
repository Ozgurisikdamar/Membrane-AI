package notify_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	pkgerrs "github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
	"github.com/Ozgurisikdamar/Membrane-AI/services/reporter/internal/adapters/notify"
	"github.com/Ozgurisikdamar/Membrane-AI/services/reporter/internal/domain"
)

func report() domain.Report {
	return domain.Report{
		SubmissionID:   "s-1",
		OrganizationID: "o-1",
		Outcome:        domain.OutcomeFailure,
		Title:          "MEMBRANE.AI ✗ rejected — 1 finding(s) must be fixed",
		Body:           "• [analyzer] aws-access-key-id (blocking): rotate it",
	}
}

func TestWebhook_PostsSlackCompatibleJSON(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("content-type = %s", ct)
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	if err := notify.NewWebhook(srv.URL).Notify(context.Background(), report()); err != nil {
		t.Fatal(err)
	}
	if got["text"] == "" || got["outcome"] != "failure" || got["submission_id"] != "s-1" {
		t.Fatalf("payload = %v", got)
	}
}

func TestWebhook_Non2xxIsUnavailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	err := notify.NewWebhook(srv.URL).Notify(context.Background(), report())
	if pkgerrs.KindOf(err) != pkgerrs.KindUnavailable {
		t.Fatalf("kind = %v, want unavailable", pkgerrs.KindOf(err))
	}
}

func TestWebhook_TransportErrorIsUnavailable(t *testing.T) {
	err := notify.NewWebhook("http://127.0.0.1:1").Notify(context.Background(), report())
	if pkgerrs.KindOf(err) != pkgerrs.KindUnavailable {
		t.Fatalf("kind = %v, want unavailable", pkgerrs.KindOf(err))
	}
}

func TestLogger_NeverFails(t *testing.T) {
	log := slog.New(slog.NewJSONHandler(io.Discard, nil))
	if err := notify.NewLogger(log).Notify(context.Background(), report()); err != nil {
		t.Fatal(err)
	}
	if notify.NewLogger(log).Name() != "log" {
		t.Fatal("name")
	}
}

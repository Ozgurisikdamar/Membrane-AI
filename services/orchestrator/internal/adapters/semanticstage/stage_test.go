package semanticstage_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	pkgerrs "github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/adapters/semanticstage"
	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/domain"
)

const orgUUID = "0b7e3f6a-1f2d-4c5b-9e8d-2a1b3c4d5e6f"

type fakeFetcher struct {
	gold    []semanticstage.GoldContext
	err     error
	called  bool
	gotOrg  string
	gotDiff string
}

func (f *fakeFetcher) Fetch(_ context.Context, orgID, _, diff string) ([]semanticstage.GoldContext, error) {
	f.called = true
	f.gotOrg = orgID
	f.gotDiff = diff
	return f.gold, f.err
}

func sub(t *testing.T, org, diff string) domain.Submission {
	t.Helper()
	s, err := domain.NewSubmission("s-1", org, "r", "f.go", "go", diff, "ide", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// fakeSemantic spins an httptest server mimicking /v1/semantic/evaluate and
// captures the request it received.
func fakeSemantic(t *testing.T, status int, respBody string) (*httptest.Server, *map[string]any) {
	t.Helper()
	captured := map[string]any{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/semantic/evaluate" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&captured)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(respBody))
	}))
	t.Cleanup(srv.Close)
	return srv, &captured
}

func TestAnalyze_MapsFindingsAndSendsMaskedDiff(t *testing.T) {
	srv, captured := fakeSemantic(t, http.StatusOK,
		`{"findings":[{"rule":"local-heuristic:auth","severity":"info","message":"m"},
		              {"rule":"weird","severity":"new-kind","message":"u"}],
		  "tier":"local","escalated":false}`)
	fetcher := &fakeFetcher{gold: []semanticstage.GoldContext{{FilePath: "gold.go", Code: "c"}}}
	stage := semanticstage.New(srv.URL, fetcher)

	out, err := stage.Analyze(context.Background(), sub(t, orgUUID, "+masked [MASKED:x]"))
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Findings) != 2 {
		t.Fatalf("findings = %+v", out.Findings)
	}
	if out.Findings[0].Stage != "semantic" || out.Findings[0].Severity != domain.SeverityInfo {
		t.Fatalf("first = %+v", out.Findings[0])
	}
	if out.Findings[1].Severity != domain.SeverityWarning {
		t.Fatalf("unknown severity must fail safe as warning: %+v", out.Findings[1])
	}
	if (*captured)["masked_diff"] != "+masked [MASKED:x]" {
		t.Fatalf("masked diff not forwarded: %v", (*captured)["masked_diff"])
	}
	goldCtx, _ := (*captured)["gold_context"].([]any)
	if len(goldCtx) != 1 {
		t.Fatalf("gold context not forwarded: %v", (*captured)["gold_context"])
	}
	if !fetcher.called || fetcher.gotOrg != orgUUID {
		t.Fatalf("fetcher: called=%v org=%s", fetcher.called, fetcher.gotOrg)
	}
}

func TestAnalyze_NonUUIDOrgSkipsRAG(t *testing.T) {
	srv, captured := fakeSemantic(t, http.StatusOK, `{"findings":[],"tier":"local","escalated":false}`)
	fetcher := &fakeFetcher{}
	stage := semanticstage.New(srv.URL, fetcher)

	if _, err := stage.Analyze(context.Background(), sub(t, "demo-org", "+x")); err != nil {
		t.Fatal(err)
	}
	if fetcher.called {
		t.Fatal("fetcher must be skipped for non-UUID orgs (gold index is UUID-keyed)")
	}
	if gc, ok := (*captured)["gold_context"].([]any); ok && len(gc) != 0 {
		t.Fatalf("gold context should be empty: %v", gc)
	}
}

func TestAnalyze_FetcherErrorIsBestEffort(t *testing.T) {
	srv, _ := fakeSemantic(t, http.StatusOK, `{"findings":[],"tier":"local","escalated":false}`)
	stage := semanticstage.New(srv.URL, &fakeFetcher{err: errors.New("resolver down")})

	if _, err := stage.Analyze(context.Background(), sub(t, orgUUID, "+x")); err != nil {
		t.Fatalf("context fetch failure must not fail the stage: %v", err)
	}
}

func TestAnalyze_Non200IsUnavailable(t *testing.T) {
	srv, _ := fakeSemantic(t, http.StatusServiceUnavailable, `{"detail":"premium not configured"}`)
	stage := semanticstage.New(srv.URL, nil)

	_, err := stage.Analyze(context.Background(), sub(t, orgUUID, "+x"))
	if pkgerrs.KindOf(err) != pkgerrs.KindUnavailable {
		t.Fatalf("kind = %v, want unavailable", pkgerrs.KindOf(err))
	}
}

func TestAnalyze_TransportErrorIsUnavailable(t *testing.T) {
	stage := semanticstage.New("http://127.0.0.1:1", nil) // nothing listens
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_, err := stage.Analyze(ctx, sub(t, orgUUID, "+x"))
	if pkgerrs.KindOf(err) != pkgerrs.KindUnavailable {
		t.Fatalf("kind = %v, want unavailable", pkgerrs.KindOf(err))
	}
}

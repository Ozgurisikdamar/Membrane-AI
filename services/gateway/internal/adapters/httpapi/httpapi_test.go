package httpapi_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Ozgurisikdamar/Membrane-AI/services/gateway/internal/adapters/httpapi"
	"github.com/Ozgurisikdamar/Membrane-AI/services/gateway/internal/app"
	"github.com/Ozgurisikdamar/Membrane-AI/services/gateway/internal/domain"
)

type noAudit struct{}

func (noAudit) Record(context.Context, string, string, domain.Verdict) error { return nil }

func server(t *testing.T) *httptest.Server {
	t.Helper()
	gw := app.NewGateway(domain.DefaultPolicy(), noAudit{})
	srv := httptest.NewServer(httpapi.NewMux(gw))
	t.Cleanup(srv.Close)
	return srv
}

func post(t *testing.T, url, body string) (int, string) {
	t.Helper()
	resp, err := http.Post(url, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(raw)
}

func TestToolCall_DangerousArgDenied(t *testing.T) {
	srv := server(t)
	code, body := post(t, srv.URL+"/v1/gateway/tool-call",
		`{"tool":"bash","args":{"cmd":"rm -rf /"}}`)
	if code != http.StatusOK {
		t.Fatalf("status=%d body=%s", code, body)
	}
	if !strings.Contains(body, `"decision":"deny"`) {
		t.Fatalf("expected deny, got %s", body)
	}
}

func TestPackage_TyposquatDenied(t *testing.T) {
	srv := server(t)
	code, body := post(t, srv.URL+"/v1/gateway/package", `{"ecosystem":"npm","name":"lodashh"}`)
	if code != http.StatusOK || !strings.Contains(body, `"decision":"deny"`) {
		t.Fatalf("expected deny for typosquat, status=%d body=%s", code, body)
	}
}

func TestShadowAI_UnsanctionedReviewed(t *testing.T) {
	srv := server(t)
	code, body := post(t, srv.URL+"/v1/gateway/shadow-ai", `{"agent":"some-random-agent","source":"ide"}`)
	if code != http.StatusOK || !strings.Contains(body, `"decision":"review"`) {
		t.Fatalf("expected review for shadow AI, status=%d body=%s", code, body)
	}
}

func TestToolCall_MissingToolIsBadRequest(t *testing.T) {
	srv := server(t)
	code, _ := post(t, srv.URL+"/v1/gateway/tool-call", `{"args":{}}`)
	if code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", code)
	}
}

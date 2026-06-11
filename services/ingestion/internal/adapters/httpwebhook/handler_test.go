package httpwebhook_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Ozgurisikdamar/Membrane-AI/services/ingestion/internal/adapters/httpwebhook"
	"github.com/Ozgurisikdamar/Membrane-AI/services/ingestion/internal/adapters/idgen"
	"github.com/Ozgurisikdamar/Membrane-AI/services/ingestion/internal/adapters/memory"
	"github.com/Ozgurisikdamar/Membrane-AI/services/ingestion/internal/app"
)

func newHandler(secret string) (*httpwebhook.Handler, *memory.Publisher) {
	pub := memory.NewPublisher()
	uc := app.NewEnqueueSubmission(pub, idgen.UUID{}, idgen.SystemClock{})
	return httpwebhook.New(uc, secret), pub
}

const validBody = `{"organization_id":"org-1","repository":"r","file_path":"a.go","language":"go","diff":"d"}`

func TestWebhook_AcceptsValid(t *testing.T) {
	h, pub := newHandler("")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(validBody)))
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202 (%s)", rec.Code, rec.Body.String())
	}
	if len(pub.Events()) != 1 {
		t.Fatalf("expected 1 published event, got %d", len(pub.Events()))
	}
}

func TestWebhook_Rejects(t *testing.T) {
	h, _ := newHandler("")
	tests := []struct {
		name, method, body string
		want               int
	}{
		{"GET not allowed", http.MethodGet, "", http.StatusMethodNotAllowed},
		{"bad json", http.MethodPost, "{not json", http.StatusBadRequest},
		{"missing fields", http.MethodPost, `{"organization_id":"o"}`, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(tt.method, "/webhook", strings.NewReader(tt.body)))
			if rec.Code != tt.want {
				t.Fatalf("status = %d, want %d", rec.Code, tt.want)
			}
		})
	}
}

func TestWebhook_SignatureVerification(t *testing.T) {
	secret := "topsecret"
	h, pub := newHandler(secret)

	// wrong signature -> 401
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(validBody))
	req.Header.Set("X-Hub-Signature-256", "sha256=deadbeef")
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("bad sig status = %d, want 401", rec.Code)
	}

	// correct signature -> 202
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(validBody))
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(validBody))
	req.Header.Set("X-Hub-Signature-256", sig)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("good sig status = %d, want 202 (%s)", rec.Code, rec.Body.String())
	}
	if len(pub.Events()) != 1 {
		t.Fatalf("expected 1 event after valid signed request, got %d", len(pub.Events()))
	}
}

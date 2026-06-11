package health_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/health"
)

func TestLive_AlwaysOK(t *testing.T) {
	h := health.New(0)
	rec := httptest.NewRecorder()
	h.Live(rec, httptest.NewRequest(http.MethodGet, "/livez", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("livez = %d, want 200", rec.Code)
	}
}

func TestReady_AllPass(t *testing.T) {
	h := health.New(0)
	h.Register("kafka", func(context.Context) error { return nil })
	rec := httptest.NewRecorder()
	h.Ready(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("readyz = %d, want 200", rec.Code)
	}
	var body struct {
		Status string            `json:"status"`
		Checks map[string]string `json:"checks"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Status != "ready" || body.Checks["kafka"] != "ok" {
		t.Fatalf("unexpected body: %+v", body)
	}
}

func TestReady_OneFails(t *testing.T) {
	h := health.New(0)
	h.Register("kafka", func(context.Context) error { return nil })
	h.Register("redis", func(context.Context) error { return errors.New("conn refused") })
	rec := httptest.NewRecorder()
	h.Ready(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("readyz = %d, want 503", rec.Code)
	}
	var body struct {
		Status string            `json:"status"`
		Checks map[string]string `json:"checks"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Status != "not_ready" || body.Checks["redis"] == "ok" {
		t.Fatalf("unexpected body: %+v", body)
	}
}

func TestMux_Routes(t *testing.T) {
	h := health.New(0)
	srv := httptest.NewServer(h.Mux())
	defer srv.Close()
	for _, path := range []string{"/livez", "/readyz"} {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s = %d, want 200", path, resp.StatusCode)
		}
	}
}

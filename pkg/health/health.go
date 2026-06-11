// Package health exposes Kubernetes-style liveness and readiness endpoints.
// Liveness is always OK while the process runs; readiness runs registered
// dependency checks (Kafka, Redis, DB, …) and reports 503 if any fail.
package health

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// Check reports whether a dependency is ready. Return nil for healthy.
type Check func(ctx context.Context) error

// Handler holds named readiness checks and serves the health endpoints.
type Handler struct {
	mu      sync.RWMutex
	checks  map[string]Check
	timeout time.Duration
}

// New returns a Handler with a per-check timeout (use 0 for the 2s default).
func New(timeout time.Duration) *Handler {
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	return &Handler{checks: make(map[string]Check), timeout: timeout}
}

// Register adds (or replaces) a readiness check by name.
func (h *Handler) Register(name string, c Check) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.checks[name] = c
}

// Live always reports the process is alive.
func (h *Handler) Live(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Ready runs all checks and returns 200 when all pass, else 503 with per-check
// detail.
func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	checks := make(map[string]Check, len(h.checks))
	for n, c := range h.checks {
		checks[n] = c
	}
	h.mu.RUnlock()

	results := make(map[string]string, len(checks))
	ok := true
	for name, c := range checks {
		ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
		err := c(ctx)
		cancel()
		if err != nil {
			ok = false
			results[name] = err.Error()
		} else {
			results[name] = "ok"
		}
	}

	status := http.StatusOK
	overall := "ready"
	if !ok {
		status = http.StatusServiceUnavailable
		overall = "not_ready"
	}
	writeJSON(w, status, map[string]any{"status": overall, "checks": results})
}

// Mux returns an http.ServeMux wired to /livez and /readyz.
func (h *Handler) Mux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/livez", h.Live)
	mux.HandleFunc("/readyz", h.Ready)
	return mux
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

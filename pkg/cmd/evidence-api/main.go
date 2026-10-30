// Command evidence-api is a minimal, read-only HTTP service that serves the
// persisted combined release-evidence records (pkg/assurance.EvidenceStore) so
// the Membrane dashboard can show real release-decision history over HTTP — the
// read surface Membrane otherwise lacks (D0 gap). It is authenticated
// (X-Membrane-Api-Key), read-only, and serves canonical JSON. Tenant scoping in
// production must derive the org from the authenticated key; the optional
// ?project= filter here is a convenience, not an authorization boundary.
package main

import (
	"crypto/subtle"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/assurance"
	"github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "evidence-api: "+err.Error())
		os.Exit(1)
	}
}

func run() error {
	port := flag.String("port", envOr("MEMBRANE_EVIDENCE_PORT", "8007"), "listen port")
	dir := flag.String("store", os.Getenv("MEMBRANE_EVIDENCE_STORE_DIR"), "evidence store directory")
	flag.Parse()

	if *dir == "" {
		return fmt.Errorf("--store or $MEMBRANE_EVIDENCE_STORE_DIR is required")
	}
	apiKey := os.Getenv("MEMBRANE_EVIDENCE_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("$MEMBRANE_EVIDENCE_API_KEY is required (read API must be authenticated)")
	}
	store, err := assurance.NewFileStore(*dir)
	if err != nil {
		return err
	}

	log := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	srv := &server{store: store, apiKey: apiKey, log: log}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) })
	mux.HandleFunc("GET /v1/releases", srv.auth(srv.list))
	mux.HandleFunc("GET /v1/releases/{id}", srv.auth(srv.get))

	httpSrv := &http.Server{
		Addr:              ":" + *port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Info("evidence-api listening", "port", *port, "store", *dir)
	return httpSrv.ListenAndServe()
}

type server struct {
	store  assurance.EvidenceStore
	apiKey string
	log    *slog.Logger
}

// auth enforces a constant-time API-key check. The key is never logged.
func (s *server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		got := r.Header.Get("X-Membrane-Api-Key")
		if subtle.ConstantTimeCompare([]byte(got), []byte(s.apiKey)) != 1 {
			writeErr(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next(w, r)
	}
}

func (s *server) list(w http.ResponseWriter, r *http.Request) {
	all, err := s.store.List()
	if err != nil {
		s.fail(w, err)
		return
	}
	project := r.URL.Query().Get("project")
	out := all[:0:0]
	for _, ev := range all {
		if project == "" || ev.ProjectID == project {
			out = append(out, ev)
		}
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *server) get(w http.ResponseWriter, r *http.Request) {
	ev, err := s.store.Get(r.PathValue("id"))
	if err != nil {
		s.fail(w, err)
		return
	}
	b, err := assurance.RenderJSON(ev)
	if err != nil {
		s.fail(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(b)
}

func (s *server) fail(w http.ResponseWriter, err error) {
	switch errs.KindOf(err) {
	case errs.KindNotFound:
		writeErr(w, http.StatusNotFound, "not found")
	case errs.KindValidation:
		writeErr(w, http.StatusBadRequest, "bad request")
	default:
		s.log.Error("evidence-api failure", "err", err.Error())
		writeErr(w, http.StatusInternalServerError, "internal error")
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

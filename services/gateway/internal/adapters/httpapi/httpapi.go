// Package httpapi exposes the gateway use-cases over HTTP+JSON (same contract
// style as the semantic service, D-023). Pydantic-free: decode at the edge,
// map to/from the domain, return the verdict.
package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/Ozgurisikdamar/Membrane-AI/services/gateway/internal/app"
	"github.com/Ozgurisikdamar/Membrane-AI/services/gateway/internal/domain"
)

// maxBody bounds request bodies (governance payloads are small).
const maxBody = 1 << 20 // 1 MiB

type verdictResponse struct {
	Decision string   `json:"decision"`
	Reasons  []string `json:"reasons"`
}

func toResponse(v domain.Verdict) verdictResponse {
	return verdictResponse{Decision: string(v.Decision), Reasons: v.Reasons}
}

type toolCallRequest struct {
	Tool   string            `json:"tool"`
	Args   map[string]string `json:"args"`
	Server string            `json:"server"`
	Agent  string            `json:"agent"`
}

type packageRequest struct {
	Ecosystem string `json:"ecosystem"`
	Name      string `json:"name"`
	Version   string `json:"version"`
}

type shadowRequest struct {
	Agent  string `json:"agent"`
	Source string `json:"source"`
	User   string `json:"user"`
}

// NewMux builds the gateway HTTP handler around the injected use-case.
func NewMux(gw *app.Gateway) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /v1/gateway/tool-call", func(w http.ResponseWriter, r *http.Request) {
		var req toolCallRequest
		if !decode(w, r, &req) {
			return
		}
		if req.Tool == "" {
			http.Error(w, "tool is required", http.StatusBadRequest)
			return
		}
		v, err := gw.GovernToolCall(r.Context(), domain.ToolCall{
			Tool: req.Tool, Args: req.Args, Server: req.Server, Agent: req.Agent,
		})
		respond(w, v, err)
	})

	mux.HandleFunc("POST /v1/gateway/package", func(w http.ResponseWriter, r *http.Request) {
		var req packageRequest
		if !decode(w, r, &req) {
			return
		}
		if req.Ecosystem == "" || req.Name == "" {
			http.Error(w, "ecosystem and name are required", http.StatusBadRequest)
			return
		}
		v, err := gw.ScreenPackage(r.Context(), domain.Package{
			Ecosystem: req.Ecosystem, Name: req.Name, Version: req.Version,
		})
		respond(w, v, err)
	})

	mux.HandleFunc("POST /v1/gateway/shadow-ai", func(w http.ResponseWriter, r *http.Request) {
		var req shadowRequest
		if !decode(w, r, &req) {
			return
		}
		if req.Agent == "" {
			http.Error(w, "agent is required", http.StatusBadRequest)
			return
		}
		v, err := gw.RecordShadow(r.Context(), domain.ShadowSignal{
			Agent: req.Agent, Source: req.Source, User: req.User,
		})
		respond(w, v, err)
	})

	return mux
}

func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBody))
	if err := dec.Decode(dst); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return false
	}
	return true
}

func respond(w http.ResponseWriter, v domain.Verdict, err error) {
	if err != nil {
		http.Error(w, "audit failed", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(toResponse(v))
}

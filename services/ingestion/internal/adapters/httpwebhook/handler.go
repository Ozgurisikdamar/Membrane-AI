// Package httpwebhook adapts source-control webhooks to the ingestion use-case.
// It optionally verifies an HMAC-SHA256 signature, parses the payload into a
// domain submission, and returns 202 on acceptance.
package httpwebhook

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	pkgerrs "github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
	"github.com/Ozgurisikdamar/Membrane-AI/services/ingestion/internal/app"
	"github.com/Ozgurisikdamar/Membrane-AI/services/ingestion/internal/domain"
)

// maxBody bounds the request body to protect the service.
const maxBody = 2 << 20 // 2 MiB

// signatureHeader is the GitHub-style HMAC header this handler verifies.
const signatureHeader = "X-Hub-Signature-256"

// Enqueuer is the use-case this adapter drives.
type Enqueuer interface {
	Handle(ctx context.Context, sub domain.Submission) (app.EnqueueResult, error)
}

// Handler is an http.Handler for inbound webhooks.
type Handler struct {
	enqueue Enqueuer
	secret  string // when empty, signature verification is disabled
}

// New returns a webhook handler. Pass an empty secret to disable HMAC checks.
func New(enqueue Enqueuer, secret string) *Handler {
	return &Handler{enqueue: enqueue, secret: secret}
}

type payload struct {
	OrganizationID string `json:"organization_id"`
	Repository     string `json:"repository"`
	FilePath       string `json:"file_path"`
	Language       string `json:"language"`
	Diff           string `json:"diff"`
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBody))
	if err != nil {
		writeError(w, http.StatusBadRequest, "cannot read body")
		return
	}
	if h.secret != "" && !validSignature(h.secret, body, r.Header.Get(signatureHeader)) {
		writeError(w, http.StatusUnauthorized, "invalid signature")
		return
	}
	var p payload
	if err := json.Unmarshal(body, &p); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	sub, err := domain.NewSubmission(p.OrganizationID, p.Repository, p.FilePath, p.Language, p.Diff, domain.OriginWebhook)
	if err != nil {
		writeError(w, httpStatus(pkgerrs.KindOf(err)), err.Error())
		return
	}
	res, err := h.enqueue.Handle(r.Context(), sub)
	if err != nil {
		writeError(w, httpStatus(pkgerrs.KindOf(err)), err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{
		"submission_id": res.SubmissionID,
		"status":        res.Status,
	})
}

// validSignature checks an "sha256=<hex>" HMAC of body using secret.
func validSignature(secret string, body []byte, header string) bool {
	const prefix = "sha256="
	if !strings.HasPrefix(header, prefix) {
		return false
	}
	want, err := hex.DecodeString(strings.TrimPrefix(header, prefix))
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hmac.Equal(want, mac.Sum(nil))
}

func httpStatus(k pkgerrs.Kind) int {
	switch k {
	case pkgerrs.KindValidation:
		return http.StatusBadRequest
	case pkgerrs.KindNotFound:
		return http.StatusNotFound
	case pkgerrs.KindConflict:
		return http.StatusConflict
	case pkgerrs.KindUnauthorized:
		return http.StatusUnauthorized
	case pkgerrs.KindUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

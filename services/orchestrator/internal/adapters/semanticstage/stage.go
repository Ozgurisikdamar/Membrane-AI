// Package semanticstage implements the AnalysisStage port against the semantic
// service's HTTP contract (D-023). It runs AFTER the analyzer in the chain, so
// the diff it forwards is already secret-masked (D-024). When a gold-context
// fetcher is wired, the organization's best implementations are attached to
// the request (RAG); context retrieval is best-effort — its failure never
// fails the stage.
package semanticstage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/domain"
	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/ports"
)

const op = "orchestrator.adapters.semanticstage"

// maxResponseBytes bounds the semantic service response.
const maxResponseBytes = 4 << 20 // 4 MiB

// GoldContext is one RAG snippet forwarded to the semantic service.
type GoldContext struct {
	FilePath             string `json:"file_path"`
	Code                 string `json:"code"`
	ArchitecturalContext string `json:"architectural_context"`
}

// ContextFetcher retrieves gold-codebase context for a submission (implemented
// by the resolver client adapter; nil = no RAG).
type ContextFetcher interface {
	Fetch(ctx context.Context, orgID, language, diff string) ([]GoldContext, error)
}

// evaluateRequest mirrors the semantic service's EvaluateRequest (D-023).
type evaluateRequest struct {
	SubmissionID   string        `json:"submission_id"`
	OrganizationID string        `json:"organization_id"`
	Language       string        `json:"language"`
	MaskedDiff     string        `json:"masked_diff"`
	GoldContext    []GoldContext `json:"gold_context"`
}

// evaluateResponse mirrors the semantic service's EvaluateResponse.
type evaluateResponse struct {
	Findings []struct {
		Rule     string `json:"rule"`
		Severity string `json:"severity"`
		Message  string `json:"message"`
	} `json:"findings"`
	Tier      string `json:"tier"`
	Escalated bool   `json:"escalated"`
}

// Stage calls the semantic service as an (advisory) step of the Saga — wrap it
// with app.Optional at the composition root.
type Stage struct {
	url     string // full evaluate endpoint URL
	client  *http.Client
	fetcher ContextFetcher // nil = skip RAG
}

// New returns the stage. baseURL is e.g. "http://localhost:8005"; fetcher may
// be nil. The Saga's per-stage deadline arrives via ctx, so the http.Client
// itself sets no timeout.
func New(baseURL string, fetcher ContextFetcher) *Stage {
	return &Stage{
		url:     baseURL + "/v1/semantic/evaluate",
		client:  &http.Client{},
		fetcher: fetcher,
	}
}

// Name implements ports.AnalysisStage.
func (s *Stage) Name() string { return "semantic" }

// Analyze forwards the (masked) diff plus optional gold context and maps the
// semantic findings into the domain.
func (s *Stage) Analyze(ctx context.Context, sub domain.Submission) (ports.StageResult, error) {
	req := evaluateRequest{
		SubmissionID:   sub.SubmissionID,
		OrganizationID: sub.OrganizationID,
		Language:       sub.Language,
		MaskedDiff:     sub.Diff, // already masked by the analyzer stage (D-024)
		GoldContext:    s.fetchContext(ctx, sub),
	}
	body, err := json.Marshal(req)
	if err != nil {
		return ports.StageResult{}, errs.Internal(op, "marshal evaluate request", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, s.url, bytes.NewReader(body))
	if err != nil {
		return ports.StageResult{}, errs.Internal(op, "build request", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return ports.StageResult{}, errs.Unavailable(op, "semantic call failed", err)
	}
	defer func() { _ = resp.Body.Close() }()

	payload, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return ports.StageResult{}, errs.Unavailable(op, "read semantic response", err)
	}
	if resp.StatusCode != http.StatusOK {
		return ports.StageResult{}, errs.Unavailable(op,
			fmt.Sprintf("semantic service returned %d", resp.StatusCode), nil)
	}

	var out evaluateResponse
	if err := json.Unmarshal(payload, &out); err != nil {
		return ports.StageResult{}, errs.Internal(op, "decode semantic response", err)
	}

	findings := make([]domain.Finding, 0, len(out.Findings))
	for _, f := range out.Findings {
		findings = append(findings, domain.Finding{
			Stage:    s.Name(),
			Rule:     f.Rule,
			Severity: severityFromWire(f.Severity),
			Message:  f.Message,
		})
	}
	return ports.StageResult{Findings: findings}, nil
}

// fetchContext retrieves RAG context best-effort: no fetcher, a non-UUID org
// (the gold index is UUID-keyed, D-020) or a fetch error simply mean "no
// context" — the semantic tier still runs.
func (s *Stage) fetchContext(ctx context.Context, sub domain.Submission) []GoldContext {
	if s.fetcher == nil {
		return nil
	}
	if _, err := uuid.Parse(sub.OrganizationID); err != nil {
		return nil
	}
	gold, err := s.fetcher.Fetch(ctx, sub.OrganizationID, sub.Language, sub.Diff)
	if err != nil {
		return nil
	}
	return gold
}

func severityFromWire(s string) domain.Severity {
	switch s {
	case "blocking":
		return domain.SeverityBlocking
	case "warning":
		return domain.SeverityWarning
	case "info":
		return domain.SeverityInfo
	default:
		// Unknown severities fail safe as warnings — surfaced, never dropped.
		return domain.SeverityWarning
	}
}

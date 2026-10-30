package assurance

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
)

// HTTPConfig configures the AgentTwin HTTP adapter. The base URL and API key are
// operator configuration (secret/environment), never user- or agent-supplied
// (work order §7.2). The base URL is checked against the Allowlist at
// construction (SSRF defense).
type HTTPConfig struct {
	BaseURL   string
	APIKey    string
	Org       string // optional X-AgentTwin-Org
	Allowlist *Allowlist
	// Timeout bounds every call (default 8s). MaxResponseBytes bounds the
	// response body (default 8 MiB).
	Timeout          time.Duration
	MaxResponseBytes int64
}

// HTTPClient is the HTTP implementation of AgentTwinClient. It authenticates
// with X-AgentTwin-Api-Key, enforces a bounded timeout and response size,
// validates the destination against the allowlist, and maps AgentTwin failures
// into typed errors (errs.Kind*). Secrets are never logged or embedded in
// errors.
type HTTPClient struct {
	cfg    HTTPConfig
	client *http.Client
}

const defaultHTTPTimeout = 8 * time.Second
const defaultMaxResponseBytes = 8 << 20 // 8 MiB

// NewHTTPClient builds the adapter, rejecting a missing/at-allowlisted base URL
// or a missing credential up front.
func NewHTTPClient(cfg HTTPConfig) (*HTTPClient, error) {
	const op = Op + ".NewHTTPClient"
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, errs.Validation(op, "AgentTwin base URL is required", nil)
	}
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, errs.Validation(op, "AgentTwin API key is required", nil)
	}
	if cfg.Allowlist == nil {
		return nil, errs.Validation(op, "a destination allowlist is required", nil)
	}
	if err := cfg.Allowlist.CheckURL(cfg.BaseURL); err != nil {
		return nil, errs.Validation(op, "AgentTwin base URL rejected by allowlist", err)
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = defaultHTTPTimeout
	}
	if cfg.MaxResponseBytes <= 0 {
		cfg.MaxResponseBytes = defaultMaxResponseBytes
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	return &HTTPClient{cfg: cfg, client: &http.Client{Timeout: cfg.Timeout}}, nil
}

// AgentTwin wire shapes (the subset consumed; see the pinned OpenAPI snapshot in
// contracts/agenttwin/). Only references/summaries are read — no raw spans.
type atClassCounts struct {
	NewCriticalFailure int `json:"NEW_CRITICAL_FAILURE"`
	Regressed          int `json:"REGRESSED"`
	Improved           int `json:"IMPROVED"`
	Unchanged          int `json:"UNCHANGED"`
	Incomplete         int `json:"INCOMPLETE"`
}

type atEvalRun struct {
	ID               string        `json:"id"`
	ProjectID        string        `json:"project_id"`
	AgentName        string        `json:"agent_name"`
	BaselineVersion  string        `json:"baseline_version"`
	CandidateVersion string        `json:"candidate_version"`
	Seed             *int          `json:"seed"`
	Status           string        `json:"status"`
	Counts           atClassCounts `json:"counts"`
	BaselineRunID    *string       `json:"baseline_run_id"`
	CandidateRunID   *string       `json:"candidate_run_id"`
}

type atEvalSummary struct {
	NewCriticalFailures []struct {
		ScenarioName string `json:"scenario_name"`
	} `json:"new_critical_failures"`
	Regressed []string `json:"regressed"`
}

type atCase struct {
	ScenarioName   string `json:"scenario_name"`
	Severity       string `json:"severity"`
	Classification string `json:"classification"`
}

type atEvalDetail struct {
	Run     atEvalRun      `json:"run"`
	Summary *atEvalSummary `json:"summary"`
	Cases   []atCase       `json:"cases"`
}

// GetEvalResult fetches a paired evaluation run and maps it to EvalResultRef.
func (c *HTTPClient) GetEvalResult(ctx context.Context, evalRunID string) (EvalResultRef, error) {
	const op = Op + ".HTTPClient.GetEvalResult"
	if strings.TrimSpace(evalRunID) == "" {
		return EvalResultRef{}, errs.Validation(op, "evalRunID is required", nil)
	}
	url := c.cfg.BaseURL + "/api/v1/eval-runs/" + urlPathEscape(evalRunID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return EvalResultRef{}, errs.Internal(op, "build request", err)
	}
	req.Header.Set("X-AgentTwin-Api-Key", c.cfg.APIKey)
	req.Header.Set("Accept", "application/json")
	if c.cfg.Org != "" {
		req.Header.Set("X-AgentTwin-Org", c.cfg.Org)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return EvalResultRef{}, errs.Unavailable(op, "agenttwin request failed", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if err := statusError(op, resp.StatusCode); err != nil {
		return EvalResultRef{}, err
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, c.cfg.MaxResponseBytes))
	if err != nil {
		return EvalResultRef{}, errs.Unavailable(op, "read response", err)
	}
	var d atEvalDetail
	if err := json.Unmarshal(body, &d); err != nil {
		return EvalResultRef{}, errs.Internal(op, "decode response", err)
	}
	return mapEvalDetail(d), nil
}

// statusError maps an AgentTwin HTTP status to a typed error, or nil for 2xx.
func statusError(op string, code int) error {
	switch {
	case code >= 200 && code < 300:
		return nil
	case code == http.StatusUnauthorized, code == http.StatusForbidden:
		return errs.Unauthorized(op, "agenttwin rejected credentials ("+strconv.Itoa(code)+")", nil)
	case code == http.StatusNotFound:
		return errs.NotFound(op, "evaluation run not found", nil)
	case code == http.StatusConflict:
		return errs.Conflict(op, "agenttwin conflict (409)", nil)
	case code == http.StatusTooManyRequests:
		return errs.Unavailable(op, "agenttwin rate limited (429)", nil)
	case code >= 500:
		return errs.Unavailable(op, "agenttwin server error ("+strconv.Itoa(code)+")", nil)
	default:
		return errs.Internal(op, "unexpected agenttwin status ("+strconv.Itoa(code)+")", nil)
	}
}

func mapEvalDetail(d atEvalDetail) EvalResultRef {
	seed := "unset"
	if d.Run.Seed != nil {
		seed = strconv.Itoa(*d.Run.Seed)
	}
	scenarios := make([]string, 0, len(d.Cases))
	per := make([]ScenarioOutcome, 0, len(d.Cases))
	for _, cs := range d.Cases {
		scenarios = append(scenarios, cs.ScenarioName)
		per = append(per, ScenarioOutcome{Scenario: cs.ScenarioName, Status: cs.Classification, Severity: cs.Severity})
	}
	pairs := make([]string, 0, 2)
	if d.Run.BaselineRunID != nil {
		pairs = append(pairs, *d.Run.BaselineRunID)
	}
	if d.Run.CandidateRunID != nil {
		pairs = append(pairs, *d.Run.CandidateRunID)
	}
	return EvalResultRef{
		EvaluationRunID:  d.Run.ID,
		AgentTwinProject: d.Run.ProjectID,
		BaselineVersion:  d.Run.BaselineVersion,
		CandidateVersion: d.Run.CandidateVersion,
		Seed:             seed,
		RawStatus:        d.Run.Status,
		Counts: EvalResult{
			NewCriticalFailures: d.Run.Counts.NewCriticalFailure,
			Regressed:           d.Run.Counts.Regressed,
			Incomplete:          d.Run.Counts.Incomplete,
		},
		ScenarioSelection: scenarios,
		PerScenario:       per,
		SimulationPairIDs: pairs,
	}
}

// urlPathEscape escapes a single path segment without pulling in net/url for a
// one-liner; AgentTwin run IDs are UUIDs, but callers may pass arbitrary text.
func urlPathEscape(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		ch := s[i]
		switch {
		case ch >= 'a' && ch <= 'z', ch >= 'A' && ch <= 'Z', ch >= '0' && ch <= '9',
			ch == '-', ch == '_', ch == '.', ch == '~':
			b.WriteByte(ch)
		default:
			const hex = "0123456789ABCDEF"
			b.WriteByte('%')
			b.WriteByte(hex[ch>>4])
			b.WriteByte(hex[ch&0xf])
		}
	}
	return b.String()
}

// compile-time check that the adapter satisfies the port.
var _ AgentTwinClient = (*HTTPClient)(nil)

package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
	"github.com/Ozgurisikdamar/Membrane-AI/services/reporter/internal/domain"
)

const opGitHub = "reporter.adapters.notify.GitHubStatus"

// statusContext is the check name shown on the commit/PR.
const statusContext = "membrane-ai/governance"

// GitHubStatus posts commit statuses (POST /repos/{owner}/{repo}/statuses/{sha}).
// Reports without repository or commit SHA are skipped silently — there is
// nothing to attach a status to (D-025).
//
// Outcome mapping: success→success, failure→failure, neutral→pending
// ("pending" renders as a yellow dot — attention without hard-failing the
// check while a human reviews).
type GitHubStatus struct {
	baseURL string // default https://api.github.com; overridable for tests/GHE
	token   string
	client  *http.Client
}

// NewGitHubStatus returns the notifier. baseURL "" means api.github.com.
func NewGitHubStatus(baseURL, token string) *GitHubStatus {
	if baseURL == "" {
		baseURL = "https://api.github.com"
	}
	return &GitHubStatus{baseURL: strings.TrimRight(baseURL, "/"), token: token, client: &http.Client{Timeout: notifyTimeout}}
}

// Name implements ports.Notifier.
func (*GitHubStatus) Name() string { return "github-status" }

type statusPayload struct {
	State       string `json:"state"`
	Description string `json:"description"`
	Context     string `json:"context"`
}

// Notify implements ports.Notifier.
func (g *GitHubStatus) Notify(ctx context.Context, r domain.Report) error {
	if r.Repository == "" || r.CommitSHA == "" {
		return nil // no coordinates → nothing to report; counts as delivered
	}
	owner, repo, ok := strings.Cut(r.Repository, "/")
	if !ok || owner == "" || repo == "" {
		return errs.Validation(opGitHub, "repository must be owner/repo, got "+r.Repository, nil)
	}

	payload, err := json.Marshal(statusPayload{
		State:       stateFor(r.Outcome),
		Description: truncate(r.Title, 140), // GitHub caps description at 140 chars
		Context:     statusContext,
	})
	if err != nil {
		return errs.Internal(opGitHub, "marshal status", err)
	}

	url := fmt.Sprintf("%s/repos/%s/%s/statuses/%s", g.baseURL, owner, repo, r.CommitSHA)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return errs.Internal(opGitHub, "build request", err)
	}
	req.Header.Set("Authorization", "Bearer "+g.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return errs.Unavailable(opGitHub, "post status", err)
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	if resp.StatusCode != http.StatusCreated {
		return errs.Unavailable(opGitHub, fmt.Sprintf("github returned %d", resp.StatusCode), nil)
	}
	return nil
}

func stateFor(o domain.Outcome) string {
	switch o {
	case domain.OutcomeSuccess:
		return "success"
	case domain.OutcomeFailure:
		return "failure"
	default:
		return "pending"
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

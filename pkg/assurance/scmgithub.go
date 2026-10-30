package assurance

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
)

// GitHubStatusPublisher publishes a combined release status as a GitHub commit
// status (POST /repos/{owner}/{repo}/statuses/{sha}). It reuses the Membrane
// reporter's outbound-HTTP conventions (bounded client timeout,
// NewRequestWithContext) and maps failures to typed errors. The token comes
// from configuration/secret and is never logged.
type GitHubStatusPublisher struct {
	baseURL string
	token   string
	client  *http.Client
}

const scmTimeout = 15 * time.Second

// NewGitHubStatusPublisher builds the publisher. baseURL defaults to the public
// GitHub API; override for GitHub Enterprise or a test server.
func NewGitHubStatusPublisher(baseURL, token string) (*GitHubStatusPublisher, error) {
	const op = Op + ".NewGitHubStatusPublisher"
	if strings.TrimSpace(token) == "" {
		return nil, errs.Validation(op, "a GitHub token is required", nil)
	}
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "https://api.github.com"
	}
	return &GitHubStatusPublisher{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		client:  &http.Client{Timeout: scmTimeout},
	}, nil
}

type ghStatusPayload struct {
	State       string `json:"state"`
	Description string `json:"description"`
	Context     string `json:"context"`
}

// Publish posts the commit status, returning typed errors (401/403→Unauthorized
// for a revoked/invalid token, 404→NotFound, other non-201→Unavailable).
func (g *GitHubStatusPublisher) Publish(ctx context.Context, s SCMStatus) error {
	const op = Op + ".GitHubStatusPublisher.Publish"
	if err := s.validate(); err != nil {
		return err
	}
	owner, repo, _ := strings.Cut(s.Repository, "/")

	payload, err := json.Marshal(ghStatusPayload{
		State:       s.State,
		Description: truncate140(s.Title),
		Context:     s.Context,
	})
	if err != nil {
		return errs.Internal(op, "marshal status", err)
	}
	url := g.baseURL + "/repos/" + owner + "/" + repo + "/statuses/" + urlPathEscape(s.CommitSHA)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return errs.Internal(op, "build request", err)
	}
	req.Header.Set("Authorization", "Bearer "+g.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return errs.Unavailable(op, "post status", err)
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))

	switch resp.StatusCode {
	case http.StatusCreated:
		return nil
	case http.StatusUnauthorized, http.StatusForbidden:
		return errs.Unauthorized(op, "github rejected the token ("+strconv.Itoa(resp.StatusCode)+")", nil)
	case http.StatusNotFound:
		return errs.NotFound(op, "repository or commit not found", nil)
	default:
		return errs.Unavailable(op, "github returned "+strconv.Itoa(resp.StatusCode), nil)
	}
}

func truncate140(s string) string {
	const n = 140
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

var _ Publisher = (*GitHubStatusPublisher)(nil)

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

const opPRComment = "reporter.adapters.notify.GitHubPRComment"

// GitHubPRComment posts the verdict as a pull-request comment
// (POST /repos/{owner}/{repo}/issues/{pr}/comments). Reports without a
// repository or PR number are skipped silently — there is no PR to comment on.
type GitHubPRComment struct {
	baseURL string // default https://api.github.com; overridable for tests/GHE
	token   string
	client  *http.Client
}

// NewGitHubPRComment returns the notifier. baseURL "" means api.github.com.
func NewGitHubPRComment(baseURL, token string) *GitHubPRComment {
	if baseURL == "" {
		baseURL = "https://api.github.com"
	}
	return &GitHubPRComment{baseURL: strings.TrimRight(baseURL, "/"), token: token, client: &http.Client{Timeout: notifyTimeout}}
}

// Name implements ports.Notifier.
func (*GitHubPRComment) Name() string { return "github-pr-comment" }

type prCommentPayload struct {
	Body string `json:"body"`
}

// Notify implements ports.Notifier.
func (g *GitHubPRComment) Notify(ctx context.Context, r domain.Report) error {
	if r.Repository == "" || r.PRNumber <= 0 {
		return nil // no PR coordinates → nothing to comment on; counts as delivered
	}
	owner, repo, ok := strings.Cut(r.Repository, "/")
	if !ok || owner == "" || repo == "" {
		return errs.Validation(opPRComment, "repository must be owner/repo, got "+r.Repository, nil)
	}

	payload, err := json.Marshal(prCommentPayload{Body: renderComment(r)})
	if err != nil {
		return errs.Internal(opPRComment, "marshal comment", err)
	}

	url := fmt.Sprintf("%s/repos/%s/%s/issues/%d/comments", g.baseURL, owner, repo, r.PRNumber)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return errs.Internal(opPRComment, "build request", err)
	}
	req.Header.Set("Authorization", "Bearer "+g.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return errs.Unavailable(opPRComment, "post comment", err)
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	if resp.StatusCode != http.StatusCreated {
		return errs.Unavailable(opPRComment, fmt.Sprintf("github returned %d", resp.StatusCode), nil)
	}
	return nil
}

// renderComment formats the report as PR-comment markdown: bold title, the
// finding list in a fenced block (keeps alignment, prevents accidental
// markdown injection from finding messages).
func renderComment(r domain.Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "**%s**\n\n", r.Title)
	b.WriteString("```\n")
	b.WriteString(r.Body)
	b.WriteString("\n```\n")
	fmt.Fprintf(&b, "_submission `%s` · MEMBRANE.AI governance_\n", r.SubmissionID)
	return b.String()
}

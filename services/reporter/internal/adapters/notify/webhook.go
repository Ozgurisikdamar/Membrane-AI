// Package notify hosts Notifier implementations.
package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
	"github.com/Ozgurisikdamar/Membrane-AI/services/reporter/internal/domain"
)

const opWebhook = "reporter.adapters.notify.Webhook"

// Webhook POSTs reports as JSON to a configured URL. The payload carries a
// Slack-compatible "text" field plus structured fields for generic consumers
// (Teams, SIEM collectors, custom hooks).
type Webhook struct {
	url    string
	client *http.Client
}

// NewWebhook returns the notifier (ctx deadlines bound each call).
func NewWebhook(url string) *Webhook {
	return &Webhook{url: url, client: &http.Client{}}
}

// Name implements ports.Notifier.
func (*Webhook) Name() string { return "webhook" }

type webhookPayload struct {
	Text           string `json:"text"` // Slack-compatible summary
	SubmissionID   string `json:"submission_id"`
	OrganizationID string `json:"organization_id"`
	Outcome        string `json:"outcome"`
	Body           string `json:"body"`
}

// Notify implements ports.Notifier.
func (w *Webhook) Notify(ctx context.Context, r domain.Report) error {
	payload, err := json.Marshal(webhookPayload{
		Text:           r.Title,
		SubmissionID:   r.SubmissionID,
		OrganizationID: r.OrganizationID,
		Outcome:        string(r.Outcome),
		Body:           r.Body,
	})
	if err != nil {
		return errs.Internal(opWebhook, "marshal payload", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.url, bytes.NewReader(payload))
	if err != nil {
		return errs.Internal(opWebhook, "build request", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.client.Do(req)
	if err != nil {
		return errs.Unavailable(opWebhook, "post webhook", err)
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return errs.Unavailable(opWebhook, fmt.Sprintf("webhook returned %d", resp.StatusCode), nil)
	}
	return nil
}

// Logger writes every report to structured logs — always-on, never fails.
type Logger struct {
	log *slog.Logger
}

// NewLogger returns the log notifier.
func NewLogger(log *slog.Logger) *Logger { return &Logger{log: log} }

// Name implements ports.Notifier.
func (*Logger) Name() string { return "log" }

// Notify implements ports.Notifier.
func (l *Logger) Notify(_ context.Context, r domain.Report) error {
	l.log.Info("verdict report",
		"submission_id", r.SubmissionID,
		"organization_id", r.OrganizationID,
		"outcome", string(r.Outcome),
		"title", r.Title,
	)
	return nil
}

package kafkabus

import (
	"context"
	"encoding/json"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/domain"
)

const opPublisher = "orchestrator.adapters.kafkabus.Publisher"

// verdictEnvelope is the on-the-wire JSON shape of code.verdict.v1.
type verdictEnvelope struct {
	SubmissionID   string           `json:"submission_id"`
	OrganizationID string           `json:"organization_id"`
	Decision       string           `json:"decision"`
	Source         string           `json:"source"`
	RulesetVersion string           `json:"ruleset_version"`
	Findings       []verdictFinding `json:"findings"`
	EvaluatedAt    time.Time        `json:"evaluated_at"`
}

type verdictFinding struct {
	Stage    string `json:"stage"`
	Rule     string `json:"rule"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

// Publisher produces verdicts to the verdict topic, keyed by organization ID
// (same per-tenant ordering guarantee as submissions).
type Publisher struct {
	client *kgo.Client
	topic  string
}

// NewPublisher dials the brokers and returns a verdict publisher.
func NewPublisher(brokers []string, topic string) (*Publisher, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ProducerLinger(5*time.Millisecond),
		kgo.RequiredAcks(kgo.AllISRAcks()),
	)
	if err != nil {
		return nil, errs.Unavailable(opPublisher, "create kafka client", err)
	}
	return &Publisher{client: client, topic: topic}, nil
}

// Publish marshals and produces the verdict synchronously.
func (p *Publisher) Publish(ctx context.Context, v domain.Verdict) error {
	findings := make([]verdictFinding, 0, len(v.Findings))
	for _, f := range v.Findings {
		findings = append(findings, verdictFinding{
			Stage: f.Stage, Rule: f.Rule, Severity: string(f.Severity), Message: f.Message,
		})
	}
	payload, err := json.Marshal(verdictEnvelope{
		SubmissionID:   v.SubmissionID,
		OrganizationID: v.OrganizationID,
		Decision:       string(v.Decision),
		Source:         string(v.Source),
		RulesetVersion: v.RulesetVersion,
		Findings:       findings,
		EvaluatedAt:    v.EvaluatedAt,
	})
	if err != nil {
		return errs.Internal(opPublisher, "marshal verdict", err)
	}
	rec := &kgo.Record{Topic: p.topic, Key: []byte(v.OrganizationID), Value: payload}
	if err := p.client.ProduceSync(ctx, rec).FirstErr(); err != nil {
		return errs.Unavailable(opPublisher, "produce record", err)
	}
	return nil
}

// Close flushes and shuts down the client.
func (p *Publisher) Close() { p.client.Close() }

// Package kafka implements the EventPublisher port against Redpanda/Kafka using
// franz-go. Events are published keyed by organization ID so a tenant's stream
// stays strictly ordered (ARCHITECTURE §3).
package kafka

import (
	"context"
	"encoding/json"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
	"github.com/Ozgurisikdamar/Membrane-AI/services/ingestion/internal/ports"
)

const op = "ingestion.adapters.kafka"

// envelope is the on-the-wire JSON shape of a published event. Kept explicit so
// the contract is reviewable and stable independent of internal structs.
type envelope struct {
	SubmissionID   string    `json:"submission_id"`
	OrganizationID string    `json:"organization_id"`
	Repository     string    `json:"repository"`
	FilePath       string    `json:"file_path"`
	Language       string    `json:"language"`
	Origin         string    `json:"origin"`
	Diff           string    `json:"diff"`
	OccurredAt     time.Time `json:"occurred_at"`
}

// Publisher produces submission events to a Kafka topic.
type Publisher struct {
	client *kgo.Client
	topic  string
}

// NewPublisher dials the brokers and returns a Publisher for topic.
func NewPublisher(brokers []string, topic string) (*Publisher, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ProducerLinger(5*time.Millisecond),
		kgo.RequiredAcks(kgo.AllISRAcks()),
	)
	if err != nil {
		return nil, errs.Unavailable(op, "create kafka client", err)
	}
	return &Publisher{client: client, topic: topic}, nil
}

// Publish marshals the event and produces it synchronously, keyed by org ID.
func (p *Publisher) Publish(ctx context.Context, e ports.Event) error {
	payload, err := json.Marshal(envelope{
		SubmissionID:   e.SubmissionID,
		OrganizationID: e.OrganizationID,
		Repository:     e.Submission.Repository,
		FilePath:       e.Submission.FilePath,
		Language:       e.Submission.Language,
		Origin:         string(e.Submission.Origin),
		Diff:           e.Submission.Diff,
		OccurredAt:     e.OccurredAt,
	})
	if err != nil {
		return errs.Internal(op, "marshal event", err)
	}
	rec := &kgo.Record{Topic: p.topic, Key: []byte(e.OrganizationID), Value: payload}
	if err := p.client.ProduceSync(ctx, rec).FirstErr(); err != nil {
		return errs.Unavailable(op, "produce record", err)
	}
	return nil
}

// Ping checks broker connectivity for readiness probes.
func (p *Publisher) Ping(ctx context.Context) error {
	if err := p.client.Ping(ctx); err != nil {
		return errs.Unavailable(op, "ping brokers", err)
	}
	return nil
}

// Close flushes and shuts down the client.
func (p *Publisher) Close() { p.client.Close() }

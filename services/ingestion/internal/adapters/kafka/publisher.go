// Package kafka implements the EventPublisher port against Redpanda/Kafka using
// franz-go. Events are published keyed by organization ID so a tenant's stream
// stays strictly ordered (ARCHITECTURE §3). The wire schema lives in
// pkg/envelope — shared with every consumer.
package kafka

import (
	"context"
	"encoding/json"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/envelope"
	"github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
	"github.com/Ozgurisikdamar/Membrane-AI/pkg/observability"
	"github.com/Ozgurisikdamar/Membrane-AI/services/ingestion/internal/ports"
)

const op = "ingestion.adapters.kafka"

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
// It opens the originating trace span and stamps W3C trace context into the
// record headers so downstream consumers join the same distributed trace (D-032).
func (p *Publisher) Publish(ctx context.Context, e ports.Event) error {
	ctx, end := observability.Start(ctx, "ingestion", "ingestion.publish")
	defer end()

	payload, err := json.Marshal(envelope.SubmissionV1{
		SubmissionID:   e.SubmissionID,
		OrganizationID: e.OrganizationID,
		Repository:     e.Submission.Repository,
		FilePath:       e.Submission.FilePath,
		Language:       e.Submission.Language,
		Origin:         string(e.Submission.Origin),
		Diff:           e.Submission.Diff,
		OccurredAt:     e.OccurredAt,
		CommitSHA:      e.Submission.CommitSHA,
		PRNumber:       e.Submission.PRNumber,
	})
	if err != nil {
		return errs.Internal(op, "marshal event", err)
	}
	rec := &kgo.Record{Topic: p.topic, Key: []byte(e.OrganizationID), Value: payload}
	for k, v := range observability.InjectHeaders(ctx) {
		rec.Headers = append(rec.Headers, kgo.RecordHeader{Key: k, Value: []byte(v)})
	}
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

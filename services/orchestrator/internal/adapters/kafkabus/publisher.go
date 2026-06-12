package kafkabus

import (
	"context"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
	"github.com/Ozgurisikdamar/Membrane-AI/pkg/observability"
	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/adapters/codec"
	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/domain"
)

const opPublisher = "orchestrator.adapters.kafkabus.Publisher"

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

// Publish marshals and produces the verdict synchronously (direct path; the
// outbox relay uses PublishRecord with pre-encoded payloads instead).
func (p *Publisher) Publish(ctx context.Context, v domain.Verdict) error {
	payload, err := codec.EncodeVerdict(v)
	if err != nil {
		return err
	}
	return p.PublishRecord(ctx, p.topic, []byte(v.OrganizationID), payload, observability.InjectHeaders(ctx))
}

// PublishRecord produces a raw record; used by the outbox relay, which reads
// already-encoded payloads (and the captured trace headers) from the outbox
// table. headers are written as record headers so consumers rejoin the trace.
func (p *Publisher) PublishRecord(ctx context.Context, topic string, key, value []byte, headers map[string]string) error {
	rec := &kgo.Record{Topic: topic, Key: key, Value: value}
	for k, v := range headers {
		rec.Headers = append(rec.Headers, kgo.RecordHeader{Key: k, Value: []byte(v)})
	}
	if err := p.client.ProduceSync(ctx, rec).FirstErr(); err != nil {
		return errs.Unavailable(opPublisher, "produce record", err)
	}
	return nil
}

// Close flushes and shuts down the client.
func (p *Publisher) Close() { p.client.Close() }

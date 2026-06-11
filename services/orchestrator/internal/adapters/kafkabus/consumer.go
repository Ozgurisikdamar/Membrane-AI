// Package kafkabus contains the Kafka adapters of the orchestrator: the
// consumer that drives the Saga from code.submission.v1 and the publisher that
// emits verdicts to code.verdict.v1. Consumption is at-least-once; the Blake3
// verdict cache makes reprocessing cheap and idempotent.
package kafkabus

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/domain"
)

const opConsumer = "orchestrator.adapters.kafkabus.Consumer"

// submissionEnvelope mirrors the JSON the ingestion service produces
// (services/ingestion/internal/adapters/kafka). Keep the two in sync until the
// envelope moves to a shared schema package (ROADMAP P1, outbox item).
type submissionEnvelope struct {
	SubmissionID   string    `json:"submission_id"`
	OrganizationID string    `json:"organization_id"`
	Repository     string    `json:"repository"`
	FilePath       string    `json:"file_path"`
	Language       string    `json:"language"`
	Origin         string    `json:"origin"`
	Diff           string    `json:"diff"`
	OccurredAt     time.Time `json:"occurred_at"`
}

// Processor is the use-case the consumer drives (narrow interface on the
// consumer side).
type Processor interface {
	Handle(ctx context.Context, sub domain.Submission) (domain.Verdict, error)
}

// Consumer polls code.submission.v1 and feeds each record into the Saga.
type Consumer struct {
	client *kgo.Client
	proc   Processor
	log    *slog.Logger
}

// NewConsumer joins the consumer group on the submission topic.
func NewConsumer(brokers []string, topic, group string, proc Processor, log *slog.Logger) (*Consumer, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ConsumeTopics(topic),
		kgo.ConsumerGroup(group),
	)
	if err != nil {
		return nil, errs.Unavailable(opConsumer, "create kafka client", err)
	}
	return &Consumer{client: client, proc: proc, log: log}, nil
}

// Run polls until ctx is canceled. Malformed records are logged and skipped
// (poison messages must not wedge the partition); processing errors are logged
// and the record is retried by the at-least-once redelivery semantics.
func (c *Consumer) Run(ctx context.Context) error {
	for {
		fetches := c.client.PollFetches(ctx)
		if ctx.Err() != nil {
			return nil
		}
		if errs := fetches.Errors(); len(errs) > 0 {
			for _, fe := range errs {
				c.log.Error("kafka fetch error", "topic", fe.Topic, "err", fe.Err)
			}
			continue
		}
		fetches.EachRecord(func(rec *kgo.Record) {
			c.handleRecord(ctx, rec)
		})
	}
}

func (c *Consumer) handleRecord(ctx context.Context, rec *kgo.Record) {
	var env submissionEnvelope
	if err := json.Unmarshal(rec.Value, &env); err != nil {
		c.log.Error("skipping malformed submission record", "offset", rec.Offset, "err", err)
		return
	}
	sub, err := domain.NewSubmission(
		env.SubmissionID, env.OrganizationID, env.Repository,
		env.FilePath, env.Language, env.Diff, env.Origin, env.OccurredAt,
	)
	if err != nil {
		c.log.Error("skipping invalid submission", "submission_id", env.SubmissionID, "err", err)
		return
	}
	verdict, err := c.proc.Handle(ctx, sub)
	if err != nil {
		c.log.Error("saga failed; record will be redelivered", "submission_id", sub.SubmissionID, "err", err)
		return
	}
	c.log.Info("verdict published",
		"submission_id", verdict.SubmissionID,
		"decision", string(verdict.Decision),
		"source", string(verdict.Source),
	)
}

// Ping checks broker connectivity for readiness probes.
func (c *Consumer) Ping(ctx context.Context) error {
	if err := c.client.Ping(ctx); err != nil {
		return errs.Unavailable(opConsumer, "ping brokers", err)
	}
	return nil
}

// Close leaves the group and releases the client.
func (c *Consumer) Close() { c.client.Close() }

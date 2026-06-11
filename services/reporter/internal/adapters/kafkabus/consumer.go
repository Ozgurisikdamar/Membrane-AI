// Package kafkabus consumes code.verdict.v1 and drives the dispatch use-case.
// Consumption is at-least-once; the DeliveryLog makes notifications idempotent
// per destination.
package kafkabus

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/envelope"
	"github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
	"github.com/Ozgurisikdamar/Membrane-AI/services/reporter/internal/domain"
)

const op = "reporter.adapters.kafkabus"

// Dispatcher is the use-case the consumer drives.
type Dispatcher interface {
	Handle(ctx context.Context, v domain.Verdict) error
}

// Consumer polls the verdict topic and dispatches each record.
type Consumer struct {
	client *kgo.Client
	disp   Dispatcher
	log    *slog.Logger
}

// NewConsumer joins the consumer group on the verdict topic.
func NewConsumer(brokers []string, topic, group string, disp Dispatcher, log *slog.Logger) (*Consumer, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ConsumeTopics(topic),
		kgo.ConsumerGroup(group),
	)
	if err != nil {
		return nil, errs.Unavailable(op, "create kafka client", err)
	}
	return &Consumer{client: client, disp: disp, log: log}, nil
}

// Run polls until ctx is canceled. Malformed/invalid records are logged and
// skipped (poison messages must not wedge the partition); delivery failures are
// logged and retried via at-least-once redelivery.
func (c *Consumer) Run(ctx context.Context) error {
	for {
		fetches := c.client.PollFetches(ctx)
		if ctx.Err() != nil {
			return nil
		}
		if errsList := fetches.Errors(); len(errsList) > 0 {
			for _, fe := range errsList {
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
	var env envelope.VerdictV1
	if err := json.Unmarshal(rec.Value, &env); err != nil {
		c.log.Error("skipping malformed verdict record", "offset", rec.Offset, "err", err)
		return
	}
	findings := make([]domain.Finding, 0, len(env.Findings))
	for _, f := range env.Findings {
		findings = append(findings, domain.Finding{
			Stage: f.Stage, Rule: f.Rule, Severity: f.Severity, Message: f.Message,
		})
	}
	v := domain.Verdict{
		SubmissionID:   env.SubmissionID,
		OrganizationID: env.OrganizationID,
		Decision:       env.Decision,
		Source:         env.Source,
		RulesetVersion: env.RulesetVersion,
		Findings:       findings,
	}
	if err := c.disp.Handle(ctx, v); err != nil {
		if errs.KindOf(err) == errs.KindValidation {
			c.log.Error("skipping invalid verdict", "submission_id", env.SubmissionID, "err", err)
			return
		}
		c.log.Error("dispatch failed; record will be redelivered",
			"submission_id", env.SubmissionID, "err", err)
		return
	}
	c.log.Info("verdict reported", "submission_id", env.SubmissionID, "decision", env.Decision)
}

// Ping checks broker connectivity for readiness probes.
func (c *Consumer) Ping(ctx context.Context) error {
	if err := c.client.Ping(ctx); err != nil {
		return errs.Unavailable(op, "ping brokers", err)
	}
	return nil
}

// Close leaves the group and releases the client.
func (c *Consumer) Close() { c.client.Close() }

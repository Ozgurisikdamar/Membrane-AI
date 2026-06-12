package outboxstore

import (
	"context"
	"log/slog"
	"time"
)

// RecordPublisher delivers one raw record to the bus (implemented by
// kafkabus.Publisher.PublishRecord). headers carries the W3C trace context
// captured when the verdict was enqueued, so the consumer rejoins the trace
// across the asynchronous outbox hop (D-032).
type RecordPublisher func(ctx context.Context, topic string, key, value []byte, headers map[string]string) error

// Shipper is the slice of Store the relay needs (narrow interface so the relay
// is unit-testable with a fake).
type Shipper interface {
	PublishPending(ctx context.Context, batch int, publish RecordPublisher) (int, error)
}

// Relay periodically ships pending outbox rows to the bus (D-021: the relay
// runs in-process; FOR UPDATE SKIP LOCKED makes concurrent replicas safe).
type Relay struct {
	shipper  Shipper
	publish  RecordPublisher
	interval time.Duration
	batch    int
	log      *slog.Logger
}

// NewRelay wires a relay. interval ≤ 0 defaults to 500 ms; batch ≤ 0 to 100.
func NewRelay(s Shipper, p RecordPublisher, interval time.Duration, batch int, log *slog.Logger) *Relay {
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}
	if batch <= 0 {
		batch = 100
	}
	return &Relay{shipper: s, publish: p, interval: interval, batch: batch, log: log}
}

// Run ticks until ctx is canceled. Errors are logged and retried next tick —
// the outbox keeps rows pending, so nothing is lost.
func (r *Relay) Run(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			n, err := r.shipper.PublishPending(ctx, r.batch, r.publish)
			if err != nil {
				if ctx.Err() == nil {
					r.log.Error("outbox relay tick failed; will retry", "err", err)
				}
				continue
			}
			if n > 0 {
				r.log.Debug("outbox relay shipped", "count", n)
			}
		}
	}
}

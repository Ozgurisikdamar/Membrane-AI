// Package redislog is the Redis-backed DeliveryLog: cross-replica idempotency
// for at-least-once verdict consumption. Where memorylog dedupes within one
// process, this dedupes across every reporter replica sharing the Redis (the
// multi-replica deployment story). Entries carry a TTL so the dedup set never
// grows unbounded — long after a verdict is delivered, re-delivery is
// vanishingly unlikely.
package redislog

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
)

const op = "reporter.adapters.redislog"

// keyPrefix namespaces delivery markers inside the shared Redis.
const keyPrefix = "membrane:reporter:delivered:"

// Log is a Redis-backed ports.DeliveryLog.
type Log struct {
	client *redis.Client
	ttl    time.Duration
}

// New dials Redis at addr. ttl <= 0 defaults to 72h (matches the verdict cache
// window — a verdict older than that won't be reprocessed in practice).
func New(addr string, ttl time.Duration) *Log {
	if ttl <= 0 {
		ttl = 72 * time.Hour
	}
	return &Log{client: redis.NewClient(&redis.Options{Addr: addr}), ttl: ttl}
}

// NewWithClient wires an existing client (used by tests with a fake server).
func NewWithClient(client *redis.Client, ttl time.Duration) *Log {
	if ttl <= 0 {
		ttl = 72 * time.Hour
	}
	return &Log{client: client, ttl: ttl}
}

func key(submissionID, notifier string) string {
	return keyPrefix + submissionID + "\x00" + notifier
}

// MarkIfNew implements ports.DeliveryLog via SET NX: the marker is created
// atomically and the call returns true exactly once per (submission, notifier)
// across all replicas; subsequent calls see the existing key and return false.
func (l *Log) MarkIfNew(ctx context.Context, submissionID, notifier string) (bool, error) {
	created, err := l.client.SetNX(ctx, key(submissionID, notifier), "1", l.ttl).Result()
	if err != nil {
		return false, errs.Unavailable(op, "redis setnx", err)
	}
	return created, nil
}

// Unmark implements ports.DeliveryLog: drop the marker so a failed notify can
// be retried by the next redelivery.
func (l *Log) Unmark(ctx context.Context, submissionID, notifier string) error {
	if err := l.client.Del(ctx, key(submissionID, notifier)).Err(); err != nil {
		return errs.Unavailable(op, "redis del", err)
	}
	return nil
}

// Ping checks connectivity for readiness probes.
func (l *Log) Ping(ctx context.Context) error {
	if err := l.client.Ping(ctx).Err(); err != nil {
		return errs.Unavailable(op, "redis ping", err)
	}
	return nil
}

// Close releases the client.
func (l *Log) Close() error { return l.client.Close() }

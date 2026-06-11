// Package rediscache implements the VerdictCache port on Redis with a TTL
// (volatile entries, D-007: 72h default so active sprint branches stay warm).
package rediscache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/domain"
)

const op = "orchestrator.adapters.rediscache"

// keyPrefix namespaces verdict entries inside the shared Redis.
const keyPrefix = "membrane:verdict:"

// Cache is a Redis-backed VerdictCache.
type Cache struct {
	client *redis.Client
	ttl    time.Duration
}

// New dials Redis at addr and returns the cache. ttl <= 0 defaults to 72h.
func New(addr string, ttl time.Duration) *Cache {
	if ttl <= 0 {
		ttl = 72 * time.Hour
	}
	return &Cache{client: redis.NewClient(&redis.Options{Addr: addr}), ttl: ttl}
}

// Get returns the cached verdict and true on a hit; a clean miss is (zero,
// false, nil).
func (c *Cache) Get(ctx context.Context, key string) (domain.Verdict, bool, error) {
	raw, err := c.client.Get(ctx, keyPrefix+key).Bytes()
	if errors.Is(err, redis.Nil) {
		return domain.Verdict{}, false, nil
	}
	if err != nil {
		return domain.Verdict{}, false, errs.Unavailable(op, "redis get", err)
	}
	var v domain.Verdict
	if err := json.Unmarshal(raw, &v); err != nil {
		// A corrupt entry behaves like a miss; the pipeline will overwrite it.
		return domain.Verdict{}, false, nil
	}
	return v, true, nil
}

// Set stores the verdict under key with the configured TTL.
func (c *Cache) Set(ctx context.Context, key string, v domain.Verdict) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return errs.Internal(op, "marshal verdict", err)
	}
	if err := c.client.Set(ctx, keyPrefix+key, raw, c.ttl).Err(); err != nil {
		return errs.Unavailable(op, "redis set", err)
	}
	return nil
}

// Ping checks connectivity for readiness probes.
func (c *Cache) Ping(ctx context.Context) error {
	if err := c.client.Ping(ctx).Err(); err != nil {
		return errs.Unavailable(op, "redis ping", err)
	}
	return nil
}

// Close releases the client.
func (c *Cache) Close() error { return c.client.Close() }

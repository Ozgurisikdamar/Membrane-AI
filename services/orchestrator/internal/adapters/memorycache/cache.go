// Package memorycache is the in-process VerdictCache used by tests and the
// local in-memory run mode (no Redis required).
package memorycache

import (
	"context"
	"sync"

	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/domain"
)

// Cache stores verdicts in a map. Safe for concurrent use. No TTL: entries
// live for the process lifetime, which is acceptable for dev/test only.
type Cache struct {
	mu    sync.RWMutex
	store map[string]domain.Verdict
}

// New returns an empty cache.
func New() *Cache { return &Cache{store: map[string]domain.Verdict{}} }

// Get returns the cached verdict and whether it was present.
func (c *Cache) Get(_ context.Context, key string) (domain.Verdict, bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.store[key]
	return v, ok, nil
}

// Set stores v under key.
func (c *Cache) Set(_ context.Context, key string, v domain.Verdict) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store[key] = v
	return nil
}

// Package memorylog is the in-memory DeliveryLog: per-instance idempotency for
// at-least-once consumption. Sufficient for a single replica; a Redis-backed
// implementation replaces it when the reporter scales out (ROADMAP).
package memorylog

import (
	"context"
	"sync"
)

// Log records delivered (submission, notifier) pairs in memory.
type Log struct {
	mu   sync.Mutex
	seen map[string]struct{}
}

// New returns an empty delivery log.
func New() *Log { return &Log{seen: make(map[string]struct{})} }

func key(submissionID, notifier string) string { return submissionID + "\x00" + notifier }

// MarkIfNew implements ports.DeliveryLog.
func (l *Log) MarkIfNew(_ context.Context, submissionID, notifier string) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	k := key(submissionID, notifier)
	if _, ok := l.seen[k]; ok {
		return false, nil
	}
	l.seen[k] = struct{}{}
	return true, nil
}

// Unmark implements ports.DeliveryLog.
func (l *Log) Unmark(_ context.Context, submissionID, notifier string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.seen, key(submissionID, notifier))
	return nil
}

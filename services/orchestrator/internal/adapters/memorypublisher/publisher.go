// Package memorypublisher is the in-process VerdictPublisher for tests and the
// local in-memory run mode.
package memorypublisher

import (
	"context"
	"sync"

	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/domain"
)

// Publisher records published verdicts in memory. Safe for concurrent use.
type Publisher struct {
	mu       sync.Mutex
	verdicts []domain.Verdict
}

// New returns an empty publisher.
func New() *Publisher { return &Publisher{} }

// Publish appends the verdict. It never fails.
func (p *Publisher) Publish(_ context.Context, v domain.Verdict) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.verdicts = append(p.verdicts, v)
	return nil
}

// Verdicts returns a copy of everything published so far.
func (p *Publisher) Verdicts() []domain.Verdict {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]domain.Verdict, len(p.verdicts))
	copy(out, p.verdicts)
	return out
}

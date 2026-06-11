// Package memory provides an in-process EventPublisher. It backs unit tests and
// the local "in-memory" run mode (no Kafka required).
package memory

import (
	"context"
	"sync"

	"github.com/Ozgurisikdamar/Membrane-AI/services/ingestion/internal/ports"
)

// Publisher records published events in memory. Safe for concurrent use.
type Publisher struct {
	mu     sync.Mutex
	events []ports.Event
}

// NewPublisher returns an empty in-memory publisher.
func NewPublisher() *Publisher { return &Publisher{} }

// Publish appends the event. It never fails.
func (p *Publisher) Publish(_ context.Context, e ports.Event) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.events = append(p.events, e)
	return nil
}

// Events returns a copy of everything published so far.
func (p *Publisher) Events() []ports.Event {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]ports.Event, len(p.events))
	copy(out, p.events)
	return out
}

// Package ports declares the interfaces the ingestion application depends on.
// Interfaces are defined here (the consumer side) and implemented by adapters,
// so use-cases depend on abstractions, never concrete I/O (SOLID D/I).
package ports

import (
	"context"
	"time"

	"github.com/Ozgurisikdamar/Membrane-AI/services/ingestion/internal/domain"
)

// Event is the message published to the bus for each accepted submission.
type Event struct {
	SubmissionID   string
	OrganizationID string // partition key — keeps a tenant's events ordered
	Submission     domain.Submission
	OccurredAt     time.Time
}

// EventPublisher publishes accepted submissions onto the event bus.
type EventPublisher interface {
	Publish(ctx context.Context, e Event) error
}

// IDGen produces unique submission identifiers.
type IDGen interface {
	NewID() string
}

// Clock returns the current time; injected so use-cases are deterministic in tests.
type Clock interface {
	Now() time.Time
}

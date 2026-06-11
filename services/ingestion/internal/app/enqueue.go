// Package app holds ingestion use-cases. Use-cases orchestrate domain objects
// and ports; they contain no transport or storage detail.
package app

import (
	"context"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
	"github.com/Ozgurisikdamar/Membrane-AI/services/ingestion/internal/domain"
	"github.com/Ozgurisikdamar/Membrane-AI/services/ingestion/internal/ports"
)

// StatusQueued is the acknowledgement returned once a submission is on the bus.
const StatusQueued = "QUEUED_FOR_IMMUNE_PROCESSING"

const opEnqueue = "ingestion.app.EnqueueSubmission"

// EnqueueResult is what callers get back after a successful enqueue.
type EnqueueResult struct {
	SubmissionID string
	Status       string
}

// EnqueueSubmission accepts a validated submission, stamps it with an ID and
// time, and publishes it to the event bus.
type EnqueueSubmission struct {
	publisher ports.EventPublisher
	ids       ports.IDGen
	clock     ports.Clock
}

// NewEnqueueSubmission wires the use-case. All dependencies are required.
func NewEnqueueSubmission(p ports.EventPublisher, ids ports.IDGen, clk ports.Clock) *EnqueueSubmission {
	return &EnqueueSubmission{publisher: p, ids: ids, clock: clk}
}

// Handle publishes sub and returns its assigned ID. A publish failure is
// reported as KindUnavailable (retryable by the caller).
func (uc *EnqueueSubmission) Handle(ctx context.Context, sub domain.Submission) (EnqueueResult, error) {
	event := ports.Event{
		SubmissionID:   uc.ids.NewID(),
		OrganizationID: sub.OrganizationID,
		Submission:     sub,
		OccurredAt:     uc.clock.Now(),
	}
	if err := uc.publisher.Publish(ctx, event); err != nil {
		return EnqueueResult{}, errs.Unavailable(opEnqueue, "failed to publish submission", err)
	}
	return EnqueueResult{SubmissionID: event.SubmissionID, Status: StatusQueued}, nil
}

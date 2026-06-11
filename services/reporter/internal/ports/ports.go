// Package ports declares the interfaces the reporter application depends on
// (consumer-side, SOLID I/D).
package ports

import (
	"context"

	"github.com/Ozgurisikdamar/Membrane-AI/services/reporter/internal/domain"
)

// Notifier delivers a report to one destination (chat webhook, commit status,
// SIEM, …). Name must be stable: it keys idempotent delivery.
type Notifier interface {
	Name() string
	Notify(ctx context.Context, r domain.Report) error
}

// DeliveryLog records which (submission, notifier) pairs already succeeded so
// at-least-once consumption never double-notifies a destination.
type DeliveryLog interface {
	// MarkIfNew returns true exactly once per key; false when already delivered.
	MarkIfNew(ctx context.Context, submissionID, notifier string) (bool, error)
	// Unmark forgets a delivery (called when a notify attempt fails so the
	// redelivery can retry that destination).
	Unmark(ctx context.Context, submissionID, notifier string) error
}

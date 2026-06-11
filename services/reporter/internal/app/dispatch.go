// Package app holds the reporter use-cases.
package app

import (
	"context"
	"errors"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
	"github.com/Ozgurisikdamar/Membrane-AI/services/reporter/internal/domain"
	"github.com/Ozgurisikdamar/Membrane-AI/services/reporter/internal/ports"
)

const opDispatch = "reporter.app.DispatchVerdict"

// DispatchVerdict renders a consumed verdict and fans it out to every notifier,
// idempotently per (submission, notifier): redeliveries retry only the
// destinations that have not succeeded yet.
type DispatchVerdict struct {
	notifiers  []ports.Notifier
	deliveries ports.DeliveryLog
}

// NewDispatchVerdict wires the use-case.
func NewDispatchVerdict(deliveries ports.DeliveryLog, notifiers ...ports.Notifier) *DispatchVerdict {
	return &DispatchVerdict{notifiers: notifiers, deliveries: deliveries}
}

// Handle renders and delivers. It returns an error when ANY destination failed
// (so the consumer leaves the record for redelivery); already-delivered
// destinations are skipped on retry.
func (uc *DispatchVerdict) Handle(ctx context.Context, v domain.Verdict) error {
	report, err := domain.NewReport(v)
	if err != nil {
		return err // validation error: poison message, caller logs and skips
	}

	var failures []error
	for _, n := range uc.notifiers {
		fresh, err := uc.deliveries.MarkIfNew(ctx, report.SubmissionID, n.Name())
		if err != nil {
			failures = append(failures, err)
			continue
		}
		if !fresh {
			continue // already delivered to this destination
		}
		if err := n.Notify(ctx, report); err != nil {
			// Forget the claim so the redelivery retries this destination.
			_ = uc.deliveries.Unmark(ctx, report.SubmissionID, n.Name())
			failures = append(failures, err)
		}
	}
	if len(failures) > 0 {
		return errs.Unavailable(opDispatch, "one or more notifications failed", errors.Join(failures...))
	}
	return nil
}

// Package logaudit records governance decisions as structured logs — the
// default audit sink. A SIEM/Kafka sink can replace it behind the same port.
package logaudit

import (
	"context"
	"log/slog"

	"github.com/Ozgurisikdamar/Membrane-AI/services/gateway/internal/domain"
)

// Sink writes audit records to a slog logger.
type Sink struct{ log *slog.Logger }

// New returns a log-backed audit sink.
func New(log *slog.Logger) *Sink { return &Sink{log: log} }

// Record implements ports.AuditSink. Deny/review are logged at Warn so they
// surface in alerting; allow at Info for the usage inventory.
func (s *Sink) Record(ctx context.Context, kind, subject string, v domain.Verdict) error {
	attrs := []any{"kind", kind, "subject", subject, "decision", string(v.Decision), "reasons", v.Reasons}
	if v.Decision == domain.DecisionAllow {
		s.log.InfoContext(ctx, "governance decision", attrs...)
	} else {
		s.log.WarnContext(ctx, "governance decision", attrs...)
	}
	return nil
}

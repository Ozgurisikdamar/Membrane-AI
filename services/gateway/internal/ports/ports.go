// Package ports declares the interfaces the gateway application depends on
// (consumer-side, SOLID I/D).
package ports

import (
	"context"

	"github.com/Ozgurisikdamar/Membrane-AI/services/gateway/internal/domain"
)

// AuditSink records every governance decision for the audit trail. Decisions
// must be observable even when allowed — that record IS the shadow-AI / tool
// usage inventory.
type AuditSink interface {
	Record(ctx context.Context, kind, subject string, v domain.Verdict) error
}

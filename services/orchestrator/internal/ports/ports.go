// Package ports declares the interfaces the orchestrator application depends
// on. Defined on the consumer side so use-cases depend on abstractions, never
// concrete I/O (SOLID D/I).
package ports

import (
	"context"
	"time"

	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/domain"
)

// VerdictCache stores consolidated verdicts keyed by domain.CacheKey.
type VerdictCache interface {
	// Get returns the cached verdict and true on a hit; false on a miss.
	Get(ctx context.Context, key string) (domain.Verdict, bool, error)
	// Set stores the verdict under key (with the adapter's configured TTL).
	Set(ctx context.Context, key string, v domain.Verdict) error
}

// AnalysisStage is one Strategy in the Saga's analysis chain (AST, vector
// context, semantic consensus, …). Implementations must honor ctx deadlines.
type AnalysisStage interface {
	Name() string
	Analyze(ctx context.Context, sub domain.Submission) ([]domain.Finding, error)
}

// VerdictPublisher emits the consolidated verdict for downstream consumers
// (reporter, audit).
type VerdictPublisher interface {
	Publish(ctx context.Context, v domain.Verdict) error
}

// Clock supplies time; injected for deterministic tests.
type Clock interface {
	Now() time.Time
}

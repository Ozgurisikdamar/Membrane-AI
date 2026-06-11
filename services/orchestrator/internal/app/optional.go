package app

import (
	"context"

	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/domain"
	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/ports"
)

// optionalStage decorates an advisory stage (Decorator pattern, D-024): when
// the inner stage fails, the Saga must NOT discard the findings of earlier,
// required stages by falling back — instead the failure surfaces as a warning
// finding and the pipeline continues.
type optionalStage struct {
	inner ports.AnalysisStage
}

// Optional wraps stage so its errors degrade to a warning finding instead of
// triggering the Saga's deterministic fallback. Use it for advisory tiers
// (e.g. the semantic stage) whose unavailability must not erase the verdict of
// required stages.
func Optional(stage ports.AnalysisStage) ports.AnalysisStage {
	return &optionalStage{inner: stage}
}

// Name implements ports.AnalysisStage.
func (o *optionalStage) Name() string { return o.inner.Name() }

// Analyze implements ports.AnalysisStage; it never returns an error.
func (o *optionalStage) Analyze(ctx context.Context, sub domain.Submission) (ports.StageResult, error) {
	out, err := o.inner.Analyze(ctx, sub)
	if err != nil {
		return ports.StageResult{Findings: []domain.Finding{{
			Stage:    o.inner.Name(),
			Rule:     "stage-unavailable",
			Severity: domain.SeverityWarning,
			Message:  "advisory stage failed; verdict produced without it: " + err.Error(),
		}}}, nil
	}
	return out, nil
}

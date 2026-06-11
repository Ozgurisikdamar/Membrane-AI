// Package app holds the analyzer use-cases.
package app

import (
	"context"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
	"github.com/Ozgurisikdamar/Membrane-AI/services/analyzer/internal/domain"
)

const opAnalyze = "analyzer.app.AnalyzeDiff"

// AnalyzeDiff runs the registered detector chain over an input and assembles
// the result, including the masked diff (Strategy composition).
type AnalyzeDiff struct {
	detectors []domain.Detector
}

// NewAnalyzeDiff wires the use-case with an ordered detector chain.
func NewAnalyzeDiff(detectors ...domain.Detector) *AnalyzeDiff {
	return &AnalyzeDiff{detectors: detectors}
}

// Handle analyzes in and returns findings plus the masked diff. A detector
// error aborts the analysis (the caller's Saga decides how to degrade).
func (uc *AnalyzeDiff) Handle(ctx context.Context, in domain.Input) (domain.Result, error) {
	var findings []domain.Finding
	masked := in.Diff
	for _, det := range uc.detectors {
		out, err := det.Detect(ctx, in)
		if err != nil {
			return domain.Result{}, errs.Internal(opAnalyze, "detector "+det.Name()+" failed", err)
		}
		findings = append(findings, out...)
		if m, ok := det.(domain.Masker); ok {
			masked = m.Mask(masked)
		}
	}
	return domain.Result{Findings: findings, MaskedDiff: masked}, nil
}

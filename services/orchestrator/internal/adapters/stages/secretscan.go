// Package stages hosts AnalysisStage implementations (the Strategy catalog).
// SecretScan is the deterministic stage: a fast credential screen used as the
// Saga's fallback — it must stay dependency-light and fast. The detection
// logic lives in pkg/scan (single source of truth shared with the analyzer
// service and the Code Sweeper CLI).
package stages

import (
	"context"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/scan"
	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/domain"
	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/ports"
)

// SecretScan is a deterministic, in-process credential screen.
type SecretScan struct {
	detector *scan.SecretDetector
}

// NewSecretScan returns the stage with the default detector set.
func NewSecretScan() *SecretScan { return &SecretScan{detector: scan.NewSecretDetector()} }

// Name implements ports.AnalysisStage.
func (s *SecretScan) Name() string { return "secret-scan" }

// Analyze screens the diff for leaked credentials. Every match is a blocking
// finding: leaked secrets are never acceptable to merge.
func (s *SecretScan) Analyze(ctx context.Context, sub domain.Submission) (ports.StageResult, error) {
	fs, err := s.detector.Detect(ctx, sub.Diff, sub.Language)
	if err != nil {
		return ports.StageResult{}, err
	}
	findings := make([]domain.Finding, 0, len(fs))
	for _, f := range fs {
		findings = append(findings, domain.Finding{
			Stage:    s.Name(),
			Rule:     f.Rule,
			Severity: domain.Severity(f.Severity),
			Message:  f.Message,
		})
	}
	return ports.StageResult{Findings: findings}, nil
}

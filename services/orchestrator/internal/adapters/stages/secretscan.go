// Package stages hosts AnalysisStage implementations (the Strategy catalog).
// SecretScan is the deterministic stage: a fast, regex-based screen for
// obviously leaked credentials in a diff. It doubles as the Saga's fallback —
// it must stay dependency-free and fast. The full AST analyzer service
// replaces/extends this per ROADMAP P1 ("Static analyzer").
package stages

import (
	"context"
	"regexp"

	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/domain"
	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/ports"
)

// secretPattern pairs a rule name with a compiled detector.
type secretPattern struct {
	rule string
	re   *regexp.Regexp
}

// Detectors are deliberately high-precision (low false-positive) patterns:
// false positives erode developer trust (report §False-positive fatigue).
var defaultPatterns = []secretPattern{
	{"private-key-block", regexp.MustCompile(`-----BEGIN (?:RSA |EC |OPENSSH |DSA |PGP )?PRIVATE KEY`)},
	{"aws-access-key-id", regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`)},
	{"github-token", regexp.MustCompile(`\bgh[pousr]_[A-Za-z0-9]{20,}\b`)},
	{"slack-token", regexp.MustCompile(`\bxox[baprs]-[A-Za-z0-9-]{10,}\b`)},
	// No leading \b: prefixed identifiers (db_password, user_token, …) must match.
	{"generic-assigned-secret", regexp.MustCompile(`(?i)(?:password|passwd|secret|api[_-]?key|token)\s*[:=]\s*["'][^"']{8,}["']`)},
}

// SecretScan is a deterministic, dependency-free credential screen.
type SecretScan struct {
	patterns []secretPattern
}

// NewSecretScan returns the stage with the default detector set.
func NewSecretScan() *SecretScan { return &SecretScan{patterns: defaultPatterns} }

// Name implements ports.AnalysisStage.
func (s *SecretScan) Name() string { return "secret-scan" }

// Analyze screens the diff for leaked credentials. Every match is a blocking
// finding: leaked secrets are never acceptable to merge.
func (s *SecretScan) Analyze(ctx context.Context, sub domain.Submission) (ports.StageResult, error) {
	if err := ctx.Err(); err != nil {
		return ports.StageResult{}, err
	}
	var findings []domain.Finding
	for _, p := range s.patterns {
		if p.re.MatchString(sub.Diff) {
			findings = append(findings, domain.Finding{
				Stage:    s.Name(),
				Rule:     p.rule,
				Severity: domain.SeverityBlocking,
				Message:  "potential credential detected in diff (" + p.rule + "); remove and rotate it",
			})
		}
	}
	return ports.StageResult{Findings: findings}, nil
}

package domain

import (
	"context"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/scan"
)

// Detector is one deterministic analysis Strategy. Implementations must be
// pure (no I/O) and honor ctx cancellation on long inputs.
//
// The actual pattern logic lives in pkg/scan — the single source of truth
// shared with the orchestrator's fallback stage and the Code Sweeper CLI.
// AST-based detectors require full-file content and land with the
// context-resolver integration (DECISIONS D-019).
type Detector interface {
	Name() string
	Detect(ctx context.Context, in Input) ([]Finding, error)
}

// Masker is implemented by detectors that can also rewrite the diff to hide
// what they detected (e.g. secret values) from downstream stages.
type Masker interface {
	Mask(diff string) string
}

func fromScan(fs []scan.Finding) []Finding {
	if len(fs) == 0 {
		return nil
	}
	out := make([]Finding, 0, len(fs))
	for _, f := range fs {
		out = append(out, Finding{
			Rule:     f.Rule,
			Severity: Severity(f.Severity), // identical value sets by design
			Message:  f.Message,
			Line:     f.Line,
		})
	}
	return out
}

// SecretDetector finds leaked credentials line by line and can mask them.
type SecretDetector struct {
	inner *scan.SecretDetector
}

// NewSecretDetector returns the detector with the default pattern set.
func NewSecretDetector() *SecretDetector {
	return &SecretDetector{inner: scan.NewSecretDetector()}
}

// Name implements Detector.
func (*SecretDetector) Name() string { return "secret" }

// Detect reports every credential match as a blocking finding with its
// 1-based diff line number.
func (d *SecretDetector) Detect(ctx context.Context, in Input) ([]Finding, error) {
	fs, err := d.inner.Detect(ctx, in.Diff, in.Language)
	if err != nil {
		return nil, err
	}
	return fromScan(fs), nil
}

// Mask replaces every detected secret value with a [MASKED:<rule>] placeholder
// so downstream (semantic/LLM) stages never see the raw credential.
func (d *SecretDetector) Mask(diff string) string { return d.inner.Mask(diff) }

// RiskyPatternDetector flags well-known insecure coding patterns.
type RiskyPatternDetector struct {
	inner *scan.RiskyPatternDetector
}

// NewRiskyPatternDetector returns the detector with the default rule set.
func NewRiskyPatternDetector() *RiskyPatternDetector {
	return &RiskyPatternDetector{inner: scan.NewRiskyPatternDetector()}
}

// Name implements Detector.
func (*RiskyPatternDetector) Name() string { return "risky-pattern" }

// Detect reports insecure patterns applicable to the input's language.
func (d *RiskyPatternDetector) Detect(ctx context.Context, in Input) ([]Finding, error) {
	fs, err := d.inner.Detect(ctx, in.Diff, in.Language)
	if err != nil {
		return nil, err
	}
	return fromScan(fs), nil
}

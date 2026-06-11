// Package scan is the single source of truth for MEMBRANE.AI's deterministic,
// line-based detectors (secrets + risky patterns) and secret masking. The
// analyzer service, the orchestrator's fallback stage and the Code Sweeper CLI
// all consume this package — patterns are defined exactly once.
//
// Detectors are pure (no I/O) and honor context cancellation on long inputs.
// AST-based detection stays out of this package by design (D-019): it needs
// full-file semantic context and lands with the resolver integration.
package scan

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// Severity grades a finding. Blocking findings reject a change.
type Severity string

// Finding severities, ordered by impact.
const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityBlocking Severity = "blocking"
)

// Finding is a single issue raised by a detector.
type Finding struct {
	Rule     string
	Severity Severity
	Message  string
	// Line is the 1-based line number within the scanned content (0 = unknown).
	Line int
}

// Detector is one deterministic analysis Strategy over text content.
type Detector interface {
	Name() string
	Detect(ctx context.Context, content, language string) ([]Finding, error)
}

// Masker is implemented by detectors that can rewrite content to hide what
// they detect (e.g. secret values) from downstream consumers.
type Masker interface {
	Mask(content string) string
}

// --- secret detector ---------------------------------------------------------

type secretPattern struct {
	rule string
	re   *regexp.Regexp
}

// High-precision patterns only: false positives erode developer trust
// (report §False-positive fatigue).
var secretPatterns = []secretPattern{
	{"private-key-block", regexp.MustCompile(`-----BEGIN (?:RSA |EC |OPENSSH |DSA |PGP )?PRIVATE KEY[^\n]*`)},
	{"aws-access-key-id", regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`)},
	{"github-token", regexp.MustCompile(`\bgh[pousr]_[A-Za-z0-9]{20,}\b`)},
	{"slack-token", regexp.MustCompile(`\bxox[baprs]-[A-Za-z0-9-]{10,}\b`)},
	// No leading \b: prefixed identifiers (db_password, user_token, …) must match.
	{"generic-assigned-secret", regexp.MustCompile(`(?i)(?:password|passwd|secret|api[_-]?key|token)\s*[:=]\s*["'][^"']{8,}["']`)},
}

// SecretDetector finds leaked credentials line by line and can mask them.
type SecretDetector struct{}

// NewSecretDetector returns the detector with the default pattern set.
func NewSecretDetector() *SecretDetector { return &SecretDetector{} }

// Name implements Detector.
func (*SecretDetector) Name() string { return "secret" }

// Detect reports every credential match as a blocking finding with its
// 1-based line number.
func (d *SecretDetector) Detect(ctx context.Context, content, _ string) ([]Finding, error) {
	var findings []Finding
	for i, line := range strings.Split(content, "\n") {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		for _, p := range secretPatterns {
			if p.re.MatchString(line) {
				findings = append(findings, Finding{
					Rule:     p.rule,
					Severity: SeverityBlocking,
					Message:  fmt.Sprintf("potential credential detected (%s); remove and rotate it", p.rule),
					Line:     i + 1,
				})
			}
		}
	}
	return findings, nil
}

// Mask replaces every detected secret value with a [MASKED:<rule>] placeholder
// so downstream (LLM) consumers never see the raw credential.
func (*SecretDetector) Mask(content string) string {
	masked := content
	for _, p := range secretPatterns {
		masked = p.re.ReplaceAllString(masked, "[MASKED:"+p.rule+"]")
	}
	return masked
}

// --- risky-pattern detector ---------------------------------------------------

type riskyPattern struct {
	rule     string
	severity Severity
	message  string
	re       *regexp.Regexp
	// language restricts the rule ("" = any language).
	language string
}

var riskyPatterns = []riskyPattern{
	{
		rule: "sql-string-concat", severity: SeverityWarning,
		message: "SQL built by string concatenation; use parameterized queries",
		re:      regexp.MustCompile(`(?i)\b(?:query|exec|prepare)\w*\(\s*"[^"]*"\s*\+`),
	},
	{
		rule: "exec-command-concat", severity: SeverityWarning,
		message: "command built from concatenated input; risk of command injection",
		re:      regexp.MustCompile(`exec\.Command\w*\([^)]*\+`), language: "go",
	},
	{
		rule: "insecure-tls-skip-verify", severity: SeverityWarning,
		message: "TLS certificate verification disabled (InsecureSkipVerify)",
		re:      regexp.MustCompile(`InsecureSkipVerify\s*:\s*true`), language: "go",
	},
}

// RiskyPatternDetector flags well-known insecure coding patterns.
type RiskyPatternDetector struct{}

// NewRiskyPatternDetector returns the detector with the default rule set.
func NewRiskyPatternDetector() *RiskyPatternDetector { return &RiskyPatternDetector{} }

// Name implements Detector.
func (*RiskyPatternDetector) Name() string { return "risky-pattern" }

// Detect reports insecure patterns applicable to the content's language.
func (d *RiskyPatternDetector) Detect(ctx context.Context, content, language string) ([]Finding, error) {
	language = strings.ToLower(strings.TrimSpace(language))
	var findings []Finding
	for i, line := range strings.Split(content, "\n") {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		for _, p := range riskyPatterns {
			if p.language != "" && p.language != language {
				continue
			}
			if p.re.MatchString(line) {
				findings = append(findings, Finding{
					Rule:     p.rule,
					Severity: p.severity,
					Message:  p.message,
					Line:     i + 1,
				})
			}
		}
	}
	return findings, nil
}

// Default returns the standard detector chain (secrets first, so masking
// composes before any later consumer).
func Default() []Detector {
	return []Detector{NewSecretDetector(), NewRiskyPatternDetector()}
}

// Run executes detectors in order over content, collecting findings and
// applying every Masker; it returns the findings and the masked content
// (identical to the input when nothing was masked).
func Run(ctx context.Context, content, language string, detectors ...Detector) ([]Finding, string, error) {
	var findings []Finding
	masked := content
	for _, d := range detectors {
		out, err := d.Detect(ctx, content, language)
		if err != nil {
			return nil, "", fmt.Errorf("detector %s: %w", d.Name(), err)
		}
		findings = append(findings, out...)
		if m, ok := d.(Masker); ok {
			masked = m.Mask(masked)
		}
	}
	return findings, masked, nil
}

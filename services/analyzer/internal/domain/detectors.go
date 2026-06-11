package domain

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// Detector is one deterministic analysis Strategy. Implementations must be
// pure (no I/O) and honor ctx cancellation on long inputs.
//
// AST-based detectors require full-file content, which arrives with the
// context-resolver integration (ROADMAP P1) — v1 detectors are line-scan based
// (see DECISIONS D-019).
type Detector interface {
	Name() string
	Detect(ctx context.Context, in Input) ([]Finding, error)
}

// Masker is implemented by detectors that can also rewrite the diff to hide
// what they detected (e.g. secret values) from downstream stages.
type Masker interface {
	Mask(diff string) string
}

// --- secret detector ---------------------------------------------------------

type secretPattern struct {
	rule string
	re   *regexp.Regexp
}

// High-precision detectors only: false positives erode developer trust.
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
// 1-based diff line number.
func (d *SecretDetector) Detect(ctx context.Context, in Input) ([]Finding, error) {
	var findings []Finding
	for i, line := range strings.Split(in.Diff, "\n") {
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
// so downstream (semantic/LLM) stages never see the raw credential.
func (*SecretDetector) Mask(diff string) string {
	masked := diff
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

// Detect reports insecure patterns applicable to the input's language.
func (d *RiskyPatternDetector) Detect(ctx context.Context, in Input) ([]Finding, error) {
	var findings []Finding
	for i, line := range strings.Split(in.Diff, "\n") {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		for _, p := range riskyPatterns {
			if p.language != "" && p.language != in.Language {
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

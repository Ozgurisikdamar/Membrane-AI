// Package domain holds the pure analyzer domain: the input under analysis,
// findings, and the deterministic detector engine. No I/O
// (ENGINEERING-STANDARDS §1).
package domain

import (
	"strings"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
)

// Input is the unit of analysis.
type Input struct {
	SubmissionID   string
	OrganizationID string
	Repository     string
	FilePath       string
	Language       string
	Diff           string
}

const opInput = "analyzer.domain.NewInput"

// NewInput validates the analysis input.
func NewInput(submissionID, orgID, repository, filePath, language, diff string) (Input, error) {
	if strings.TrimSpace(diff) == "" {
		return Input{}, errs.Validation(opInput, "diff is required", nil)
	}
	return Input{
		SubmissionID:   strings.TrimSpace(submissionID),
		OrganizationID: strings.TrimSpace(orgID),
		Repository:     strings.TrimSpace(repository),
		FilePath:       strings.TrimSpace(filePath),
		Language:       strings.ToLower(strings.TrimSpace(language)),
		Diff:           diff,
	}, nil
}

// Severity grades a finding.
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
	// Line is the 1-based line number within the diff text (0 = unknown).
	Line int
}

// Result is the outcome of analyzing one input.
type Result struct {
	Findings []Finding
	// MaskedDiff is the diff with every detected secret value replaced by a
	// [MASKED:<rule>] placeholder; equals the input diff when nothing matched.
	MaskedDiff string
}

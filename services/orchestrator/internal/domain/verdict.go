// Package domain holds the pure orchestrator domain: the submissions it
// consumes, the findings produced by analysis stages, and the verdicts the
// Saga consolidates. No I/O, no transport (ENGINEERING-STANDARDS §1).
package domain

import (
	"encoding/hex"
	"strings"
	"time"

	"lukechampine.com/blake3"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
)

// Submission is the consumed shape of a code.submission.v1 event.
type Submission struct {
	SubmissionID   string
	OrganizationID string
	Repository     string
	FilePath       string
	Language       string
	Diff           string
	Origin         string
	OccurredAt     time.Time
}

const opSubmission = "orchestrator.domain.NewSubmission"

// NewSubmission validates the consumed event payload.
func NewSubmission(submissionID, orgID, repository, filePath, language, diff, origin string, occurredAt time.Time) (Submission, error) {
	if strings.TrimSpace(submissionID) == "" {
		return Submission{}, errs.Validation(opSubmission, "submission_id is required", nil)
	}
	if strings.TrimSpace(orgID) == "" {
		return Submission{}, errs.Validation(opSubmission, "organization_id is required", nil)
	}
	if strings.TrimSpace(diff) == "" {
		return Submission{}, errs.Validation(opSubmission, "diff is required", nil)
	}
	return Submission{
		SubmissionID:   strings.TrimSpace(submissionID),
		OrganizationID: strings.TrimSpace(orgID),
		Repository:     strings.TrimSpace(repository),
		FilePath:       strings.TrimSpace(filePath),
		Language:       strings.TrimSpace(language),
		Diff:           diff,
		Origin:         strings.TrimSpace(origin),
		OccurredAt:     occurredAt,
	}, nil
}

// Severity grades a finding. Blocking findings reject the change.
type Severity string

// Finding severities, ordered by impact.
const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityBlocking Severity = "blocking"
)

// Finding is a single issue raised by an analysis stage.
type Finding struct {
	Stage    string
	Rule     string
	Severity Severity
	Message  string
}

// Decision is the consolidated outcome for a submission.
type Decision string

// Possible decisions.
const (
	DecisionApproved    Decision = "approved"
	DecisionNeedsReview Decision = "needs_review"
	DecisionRejected    Decision = "rejected"
)

// Source records which path produced the verdict.
type Source string

// Verdict sources.
const (
	SourceCache    Source = "cache"
	SourcePipeline Source = "pipeline"
	SourceFallback Source = "fallback"
)

// Verdict is the consolidated result for one submission.
type Verdict struct {
	SubmissionID   string
	OrganizationID string
	Decision       Decision
	Source         Source
	RulesetVersion string
	Findings       []Finding
	EvaluatedAt    time.Time
}

// Consolidate folds findings into a decision: any blocking finding rejects,
// any warning demands review, otherwise the change is approved.
func Consolidate(findings []Finding) Decision {
	decision := DecisionApproved
	for _, f := range findings {
		switch f.Severity {
		case SeverityBlocking:
			return DecisionRejected
		case SeverityWarning:
			decision = DecisionNeedsReview
		case SeverityInfo:
			// informational findings never change the decision
		}
	}
	return decision
}

// CacheKey derives the deterministic Blake3 verdict-cache key from the diff
// content and the active ruleset version (D-007): same diff + same rules ⇒
// same verdict, so the pipeline can be skipped entirely.
func CacheKey(diff, rulesetVersion string) string {
	h := blake3.New(32, nil)
	_, _ = h.Write([]byte(rulesetVersion))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(diff))
	return hex.EncodeToString(h.Sum(nil))
}

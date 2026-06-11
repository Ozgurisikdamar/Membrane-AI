// Package domain holds the pure ingestion domain: entities, value objects and
// their invariants. It performs no I/O and imports no adapter or transport code
// (see ENGINEERING-STANDARDS §1).
package domain

import (
	"strings"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
)

// Origin records where a submission entered the system.
type Origin string

// Submission origins.
const (
	OriginUnspecified Origin = "unspecified"
	OriginIDE         Origin = "ide"
	OriginWebhook     Origin = "webhook"
)

// maxDiffBytes bounds an accepted diff so a single submission cannot exhaust
// memory or the event bus message size.
const maxDiffBytes = 1 << 20 // 1 MiB

// Submission is a single code change to be governed. It is a value object:
// construct it only via NewSubmission so its invariants always hold.
type Submission struct {
	OrganizationID string
	Repository     string
	FilePath       string
	Language       string
	Diff           string
	Origin         Origin
	// CommitSHA and PRNumber are optional source coordinates (empty/0 when the
	// client cannot provide them); set them via WithSource.
	CommitSHA string
	PRNumber  int
}

// WithSource attaches optional source coordinates (used by commit-status and
// PR-comment reporting). A negative PR number is treated as absent.
func (s Submission) WithSource(commitSHA string, prNumber int) Submission {
	s.CommitSHA = strings.TrimSpace(commitSHA)
	if prNumber > 0 {
		s.PRNumber = prNumber
	}
	return s
}

const op = "ingestion.domain.NewSubmission"

// NewSubmission validates the inputs and returns a Submission, or a
// KindValidation error describing the first failed rule.
func NewSubmission(orgID, repository, filePath, language, diff string, origin Origin) (Submission, error) {
	orgID = strings.TrimSpace(orgID)
	if orgID == "" {
		return Submission{}, errs.Validation(op, "organization_id is required", nil)
	}
	if strings.TrimSpace(repository) == "" {
		return Submission{}, errs.Validation(op, "repository is required", nil)
	}
	if strings.TrimSpace(filePath) == "" {
		return Submission{}, errs.Validation(op, "file_path is required", nil)
	}
	if strings.TrimSpace(diff) == "" {
		return Submission{}, errs.Validation(op, "diff is required", nil)
	}
	if len(diff) > maxDiffBytes {
		return Submission{}, errs.Validation(op, "diff exceeds the maximum size", nil)
	}
	return Submission{
		OrganizationID: orgID,
		Repository:     strings.TrimSpace(repository),
		FilePath:       strings.TrimSpace(filePath),
		Language:       strings.TrimSpace(language),
		Diff:           diff,
		Origin:         normalizeOrigin(origin),
	}, nil
}

func normalizeOrigin(o Origin) Origin {
	switch o {
	case OriginIDE, OriginWebhook:
		return o
	default:
		return OriginUnspecified
	}
}

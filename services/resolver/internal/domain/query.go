// Package domain holds the pure resolver domain: the RAG query and its
// invariants, and the gold-codebase match. No I/O (ENGINEERING-STANDARDS §1).
package domain

import (
	"strings"

	"github.com/google/uuid"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
)

// Limit bounds for returned matches.
const (
	DefaultLimit = 3
	MaxLimit     = 10
)

const op = "resolver.domain.NewQuery"

// Query is a validated gold-codebase retrieval request.
type Query struct {
	OrganizationID string // canonical UUID string
	Language       string
	Diff           string
	Limit          int
}

// NewQuery validates inputs. The organization ID must be a UUID because the
// gold index is tenant-keyed by UUID (schema 0001); limit 0 means DefaultLimit
// and anything above MaxLimit is capped.
func NewQuery(orgID, language, diff string, limit int) (Query, error) {
	id, err := uuid.Parse(strings.TrimSpace(orgID))
	if err != nil {
		return Query{}, errs.Validation(op, "organization_id must be a UUID", err)
	}
	language = strings.ToLower(strings.TrimSpace(language))
	if language == "" {
		return Query{}, errs.Validation(op, "language is required", nil)
	}
	if strings.TrimSpace(diff) == "" {
		return Query{}, errs.Validation(op, "diff is required", nil)
	}
	switch {
	case limit <= 0:
		limit = DefaultLimit
	case limit > MaxLimit:
		limit = MaxLimit
	}
	return Query{OrganizationID: id.String(), Language: language, Diff: diff, Limit: limit}, nil
}

// GoldMatch is one retrieved gold-codebase snippet, ordered by similarity.
type GoldMatch struct {
	FilePath             string
	RawCodeContent       string
	ArchitecturalContext string
	// CosineDistance: lower is more similar.
	CosineDistance float64
}

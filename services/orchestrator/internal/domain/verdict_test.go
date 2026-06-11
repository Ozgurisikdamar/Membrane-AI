package domain_test

import (
	"testing"
	"time"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/domain"
)

func TestNewSubmission_ValidAndTrimmed(t *testing.T) {
	now := time.Now()
	got, err := domain.NewSubmission(" id ", " org ", " repo ", " f.go ", " go ", "diff", " ide ", now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.SubmissionID != "id" || got.OrganizationID != "org" || got.Origin != "ide" {
		t.Fatalf("fields not trimmed: %+v", got)
	}
}

func TestNewSubmission_Validation(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name          string
		id, org, diff string
	}{
		{"no submission id", "", "org", "d"},
		{"no org", "id", "", "d"},
		{"no diff", "id", "org", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := domain.NewSubmission(tt.id, tt.org, "r", "f", "go", tt.diff, "ide", now)
			if errs.KindOf(err) != errs.KindValidation {
				t.Fatalf("kind = %v, want validation", errs.KindOf(err))
			}
		})
	}
}

func TestConsolidate(t *testing.T) {
	tests := []struct {
		name     string
		findings []domain.Finding
		want     domain.Decision
	}{
		{"none → approved", nil, domain.DecisionApproved},
		{"info only → approved",
			[]domain.Finding{{Severity: domain.SeverityInfo}}, domain.DecisionApproved},
		{"warning → needs review",
			[]domain.Finding{{Severity: domain.SeverityInfo}, {Severity: domain.SeverityWarning}},
			domain.DecisionNeedsReview},
		{"blocking wins over warning",
			[]domain.Finding{{Severity: domain.SeverityWarning}, {Severity: domain.SeverityBlocking}},
			domain.DecisionRejected},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := domain.Consolidate(tt.findings); got != tt.want {
				t.Fatalf("Consolidate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCacheKey(t *testing.T) {
	a := domain.CacheKey("diff-a", "v1")
	if len(a) != 64 { // 32-byte blake3 → 64 hex chars
		t.Fatalf("key length = %d, want 64", len(a))
	}
	if a != domain.CacheKey("diff-a", "v1") {
		t.Fatal("CacheKey must be deterministic")
	}
	if a == domain.CacheKey("diff-b", "v1") {
		t.Fatal("different diffs must produce different keys")
	}
	if a == domain.CacheKey("diff-a", "v2") {
		t.Fatal("different ruleset versions must produce different keys")
	}
	// Domain separation: (rulesetVersion, diff) must not collide across the
	// boundary, e.g. ("v1","x") vs ("v","1x").
	if domain.CacheKey("1x", "v") == domain.CacheKey("x", "v1") {
		t.Fatal("boundary collision between ruleset and diff")
	}
}

package domain_test

import (
	"strings"
	"testing"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
	"github.com/Ozgurisikdamar/Membrane-AI/services/ingestion/internal/domain"
)

func TestNewSubmission_Valid(t *testing.T) {
	got, err := domain.NewSubmission(" org-1 ", " repo ", " a/b.go ", " go ", "diff", domain.OriginIDE)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.OrganizationID != "org-1" || got.Repository != "repo" || got.FilePath != "a/b.go" {
		t.Fatalf("fields not trimmed/set: %+v", got)
	}
	if got.Origin != domain.OriginIDE {
		t.Fatalf("origin = %v", got.Origin)
	}
}

func TestWithSource(t *testing.T) {
	base, err := domain.NewSubmission("o", "r", "f", "go", "d", domain.OriginWebhook)
	if err != nil {
		t.Fatal(err)
	}
	s := base.WithSource("  abc123  ", 42)
	if s.CommitSHA != "abc123" || s.PRNumber != 42 {
		t.Fatalf("source = %q/%d", s.CommitSHA, s.PRNumber)
	}
	neg := base.WithSource("", -1)
	if neg.CommitSHA != "" || neg.PRNumber != 0 {
		t.Fatalf("negative PR must be absent: %+v", neg)
	}
}

func TestNewSubmission_NormalizesUnknownOrigin(t *testing.T) {
	got, err := domain.NewSubmission("o", "r", "f", "go", "d", domain.Origin("weird"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Origin != domain.OriginUnspecified {
		t.Fatalf("origin = %v, want unspecified", got.Origin)
	}
}

func TestNewSubmission_Validation(t *testing.T) {
	tests := []struct {
		name                        string
		org, repo, file, lang, diff string
	}{
		{"no org", "", "r", "f", "go", "d"},
		{"no repo", "o", "", "f", "go", "d"},
		{"no file", "o", "r", "", "go", "d"},
		{"no diff", "o", "r", "f", "go", ""},
		{"diff too big", "o", "r", "f", "go", strings.Repeat("x", (1<<20)+1)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := domain.NewSubmission(tt.org, tt.repo, tt.file, tt.lang, tt.diff, domain.OriginWebhook)
			if err == nil {
				t.Fatal("expected validation error")
			}
			if errs.KindOf(err) != errs.KindValidation {
				t.Fatalf("kind = %v, want validation", errs.KindOf(err))
			}
		})
	}
}

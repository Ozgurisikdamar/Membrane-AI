package domain_test

import (
	"testing"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
	"github.com/Ozgurisikdamar/Membrane-AI/services/resolver/internal/domain"
)

const orgUUID = "0b7e3f6a-1f2d-4c5b-9e8d-2a1b3c4d5e6f"

func TestNewQuery_Valid(t *testing.T) {
	q, err := domain.NewQuery(" "+orgUUID+" ", " Go ", "+diff", 0)
	if err != nil {
		t.Fatal(err)
	}
	if q.OrganizationID != orgUUID {
		t.Fatalf("org = %q", q.OrganizationID)
	}
	if q.Language != "go" {
		t.Fatalf("language = %q, want normalized go", q.Language)
	}
	if q.Limit != domain.DefaultLimit {
		t.Fatalf("limit = %d, want default %d", q.Limit, domain.DefaultLimit)
	}
}

func TestNewQuery_LimitCap(t *testing.T) {
	q, err := domain.NewQuery(orgUUID, "go", "+d", 99)
	if err != nil {
		t.Fatal(err)
	}
	if q.Limit != domain.MaxLimit {
		t.Fatalf("limit = %d, want capped %d", q.Limit, domain.MaxLimit)
	}
}

func TestNewQuery_Validation(t *testing.T) {
	tests := []struct {
		name, org, lang, diff string
	}{
		{"bad uuid", "org-1", "go", "+d"},
		{"empty org", "", "go", "+d"},
		{"no language", orgUUID, " ", "+d"},
		{"no diff", orgUUID, "go", "  "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := domain.NewQuery(tt.org, tt.lang, tt.diff, 1)
			if errs.KindOf(err) != errs.KindValidation {
				t.Fatalf("kind = %v, want validation", errs.KindOf(err))
			}
		})
	}
}

package domain_test

import (
	"strings"
	"testing"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
	"github.com/Ozgurisikdamar/Membrane-AI/services/reporter/internal/domain"
)

func verdict(decision string, findings ...domain.Finding) domain.Verdict {
	return domain.Verdict{
		SubmissionID:   "s-1",
		OrganizationID: "o-1",
		Decision:       decision,
		Source:         "pipeline",
		RulesetVersion: "v1",
		Findings:       findings,
	}
}

func TestNewReport_OutcomeMapping(t *testing.T) {
	tests := []struct {
		decision string
		want     domain.Outcome
	}{
		{"approved", domain.OutcomeSuccess},
		{"rejected", domain.OutcomeFailure},
		{"needs_review", domain.OutcomeNeutral},
	}
	for _, tt := range tests {
		t.Run(tt.decision, func(t *testing.T) {
			r, err := domain.NewReport(verdict(tt.decision))
			if err != nil {
				t.Fatal(err)
			}
			if r.Outcome != tt.want {
				t.Fatalf("outcome = %v, want %v", r.Outcome, tt.want)
			}
			if !strings.Contains(r.Title, "MEMBRANE.AI") {
				t.Fatalf("title = %q", r.Title)
			}
		})
	}
}

func TestNewReport_Validation(t *testing.T) {
	if _, err := domain.NewReport(domain.Verdict{Decision: "approved"}); errs.KindOf(err) != errs.KindValidation {
		t.Fatalf("missing submission id: kind = %v", errs.KindOf(err))
	}
	if _, err := domain.NewReport(verdict("wat")); errs.KindOf(err) != errs.KindValidation {
		t.Fatalf("unknown decision: kind = %v", errs.KindOf(err))
	}
}

func TestNewReport_BodyRendersAndCapsFindings(t *testing.T) {
	many := make([]domain.Finding, 14)
	for i := range many {
		many[i] = domain.Finding{Stage: "analyzer", Rule: "r", Severity: "blocking", Message: "m"}
	}
	r, err := domain.NewReport(verdict("rejected", many...))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(r.Body, "… and 4 more finding(s)") {
		t.Fatalf("body must cap at 10 findings:\n%s", r.Body)
	}
	if !strings.Contains(r.Title, "14 finding(s)") {
		t.Fatalf("title = %q", r.Title)
	}

	empty, _ := domain.NewReport(verdict("approved"))
	if !strings.Contains(empty.Body, "no findings") {
		t.Fatalf("empty body = %q", empty.Body)
	}
}

package codec_test

import (
	"strings"
	"testing"
	"time"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/adapters/codec"
	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/domain"
)

func TestDecodeSubmission_Valid(t *testing.T) {
	payload := []byte(`{"submission_id":"s","organization_id":"o","repository":"r",` +
		`"file_path":"f","language":"go","origin":"ide","diff":"+d","occurred_at":"2026-06-11T12:00:00Z"}`)
	sub, err := codec.DecodeSubmission(payload)
	if err != nil {
		t.Fatal(err)
	}
	if sub.SubmissionID != "s" || sub.OrganizationID != "o" || sub.Diff != "+d" {
		t.Fatalf("sub = %+v", sub)
	}
}

func TestDecodeSubmission_Errors(t *testing.T) {
	tests := []struct {
		name    string
		payload string
	}{
		{"malformed json", `{not json`},
		{"missing diff", `{"submission_id":"s","organization_id":"o"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := codec.DecodeSubmission([]byte(tt.payload))
			if errs.KindOf(err) != errs.KindValidation {
				t.Fatalf("kind = %v, want validation", errs.KindOf(err))
			}
		})
	}
}

func TestEncodeVerdict(t *testing.T) {
	v := domain.Verdict{
		SubmissionID:   "s",
		OrganizationID: "o",
		Decision:       domain.DecisionRejected,
		Source:         domain.SourcePipeline,
		RulesetVersion: "v1",
		Findings:       []domain.Finding{{Stage: "analyzer", Rule: "x", Severity: domain.SeverityBlocking, Message: "m"}},
		EvaluatedAt:    time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC),
	}
	payload, err := codec.EncodeVerdict(v)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"decision":"rejected"`, `"source":"pipeline"`, `"rule":"x"`} {
		if !strings.Contains(string(payload), want) {
			t.Fatalf("payload missing %s: %s", want, payload)
		}
	}
}

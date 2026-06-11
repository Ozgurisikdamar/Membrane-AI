package envelope_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/envelope"
)

// The JSON field names are the cross-service contract: these golden strings
// pin them so an accidental tag rename fails loudly.

func TestSubmissionV1_GoldenJSON(t *testing.T) {
	at := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	got, err := json.Marshal(envelope.SubmissionV1{
		SubmissionID: "s", OrganizationID: "o", Repository: "r",
		FilePath: "f", Language: "go", Origin: "ide", Diff: "+d", OccurredAt: at,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"submission_id":"s","organization_id":"o","repository":"r","file_path":"f",` +
		`"language":"go","origin":"ide","diff":"+d","occurred_at":"2026-06-11T12:00:00Z"}`
	if string(got) != want {
		t.Fatalf("golden mismatch:\n got %s\nwant %s", got, want)
	}
}

func TestVerdictV1_GoldenJSON(t *testing.T) {
	at := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	got, err := json.Marshal(envelope.VerdictV1{
		SubmissionID: "s", OrganizationID: "o", Decision: "rejected", Source: "pipeline",
		RulesetVersion: "v1",
		Findings:       []envelope.FindingV1{{Stage: "analyzer", Rule: "x", Severity: "blocking", Message: "m"}},
		EvaluatedAt:    at,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"submission_id":"s","organization_id":"o","decision":"rejected","source":"pipeline",` +
		`"ruleset_version":"v1","findings":[{"stage":"analyzer","rule":"x","severity":"blocking","message":"m"}],` +
		`"evaluated_at":"2026-06-11T12:00:00Z"}`
	if string(got) != want {
		t.Fatalf("golden mismatch:\n got %s\nwant %s", got, want)
	}
}

func TestRoundTrip(t *testing.T) {
	in := envelope.VerdictV1{SubmissionID: "s", Decision: "approved", Findings: []envelope.FindingV1{}}
	b, _ := json.Marshal(in)
	var out envelope.VerdictV1
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if out.SubmissionID != "s" || out.Decision != "approved" {
		t.Fatalf("round trip lost data: %+v", out)
	}
}

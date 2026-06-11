package envelope_test

import (
	"encoding/json"
	"strings"
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

func TestOptionalSourceCoordinates_OmittedWhenEmpty(t *testing.T) {
	// The pre-enrichment golden strings above must stay valid: empty optional
	// fields may not appear on the wire.
	b, _ := json.Marshal(envelope.SubmissionV1{SubmissionID: "s"})
	if string(b) != `{"submission_id":"s","organization_id":"","repository":"","file_path":"",`+
		`"language":"","origin":"","diff":"","occurred_at":"0001-01-01T00:00:00Z"}` {
		t.Fatalf("optional fields leaked into the wire: %s", b)
	}
	v, _ := json.Marshal(envelope.VerdictV1{
		SubmissionID: "s", CommitSHA: "abc123", Repository: "owner/repo", PRNumber: 7,
		Findings: []envelope.FindingV1{},
	})
	for _, want := range []string{`"commit_sha":"abc123"`, `"repository":"owner/repo"`, `"pr_number":7`} {
		if !strings.Contains(string(v), want) {
			t.Fatalf("missing %s in %s", want, v)
		}
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

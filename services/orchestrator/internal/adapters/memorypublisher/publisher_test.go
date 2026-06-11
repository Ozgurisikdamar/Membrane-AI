package memorypublisher_test

import (
	"context"
	"testing"

	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/adapters/memorypublisher"
	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/domain"
)

func TestPublisher_RecordsAndCopies(t *testing.T) {
	p := memorypublisher.New()
	if err := p.Publish(context.Background(), domain.Verdict{SubmissionID: "a"}); err != nil {
		t.Fatal(err)
	}
	got := p.Verdicts()
	if len(got) != 1 || got[0].SubmissionID != "a" {
		t.Fatalf("verdicts = %+v", got)
	}
	got[0].SubmissionID = "mutated"
	if p.Verdicts()[0].SubmissionID != "a" {
		t.Fatal("Verdicts() leaked the internal slice")
	}
}

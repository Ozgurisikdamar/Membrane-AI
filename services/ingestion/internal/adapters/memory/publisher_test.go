package memory_test

import (
	"context"
	"testing"

	"github.com/Ozgurisikdamar/Membrane-AI/services/ingestion/internal/adapters/memory"
	"github.com/Ozgurisikdamar/Membrane-AI/services/ingestion/internal/ports"
)

func TestPublisher_RecordsAndCopies(t *testing.T) {
	p := memory.NewPublisher()
	if err := p.Publish(context.Background(), ports.Event{SubmissionID: "a"}); err != nil {
		t.Fatal(err)
	}
	if err := p.Publish(context.Background(), ports.Event{SubmissionID: "b"}); err != nil {
		t.Fatal(err)
	}
	got := p.Events()
	if len(got) != 2 || got[0].SubmissionID != "a" || got[1].SubmissionID != "b" {
		t.Fatalf("events = %+v", got)
	}
	// Events() must return a copy: mutating it must not affect the publisher.
	got[0].SubmissionID = "mutated"
	if p.Events()[0].SubmissionID != "a" {
		t.Fatal("Events() leaked the internal slice")
	}
}

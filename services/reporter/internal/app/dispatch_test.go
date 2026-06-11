package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
	"github.com/Ozgurisikdamar/Membrane-AI/services/reporter/internal/adapters/memorylog"
	"github.com/Ozgurisikdamar/Membrane-AI/services/reporter/internal/app"
	"github.com/Ozgurisikdamar/Membrane-AI/services/reporter/internal/domain"
)

type fakeNotifier struct {
	name  string
	calls int
	err   error
}

func (f *fakeNotifier) Name() string { return f.name }
func (f *fakeNotifier) Notify(context.Context, domain.Report) error {
	f.calls++
	return f.err
}

func okVerdict() domain.Verdict {
	return domain.Verdict{SubmissionID: "s-1", OrganizationID: "o", Decision: "approved"}
}

func TestHandle_NotifiesAllOnce(t *testing.T) {
	a, b := &fakeNotifier{name: "a"}, &fakeNotifier{name: "b"}
	uc := app.NewDispatchVerdict(memorylog.New(), a, b)

	if err := uc.Handle(context.Background(), okVerdict()); err != nil {
		t.Fatal(err)
	}
	// Redelivery: both already delivered → no new calls, no error.
	if err := uc.Handle(context.Background(), okVerdict()); err != nil {
		t.Fatal(err)
	}
	if a.calls != 1 || b.calls != 1 {
		t.Fatalf("calls a=%d b=%d, want 1/1 (idempotent)", a.calls, b.calls)
	}
}

func TestHandle_PartialFailureRetriesOnlyFailed(t *testing.T) {
	good := &fakeNotifier{name: "good"}
	bad := &fakeNotifier{name: "bad", err: errors.New("webhook 500")}
	uc := app.NewDispatchVerdict(memorylog.New(), good, bad)

	err := uc.Handle(context.Background(), okVerdict())
	if errs.KindOf(err) != errs.KindUnavailable {
		t.Fatalf("kind = %v, want unavailable", errs.KindOf(err))
	}

	bad.err = nil // destination recovers; redelivery arrives
	if err := uc.Handle(context.Background(), okVerdict()); err != nil {
		t.Fatal(err)
	}
	if good.calls != 1 {
		t.Fatalf("good notified %d times, want exactly 1", good.calls)
	}
	if bad.calls != 2 {
		t.Fatalf("bad notified %d times, want 2 (fail then retry)", bad.calls)
	}
}

func TestHandle_InvalidVerdictIsValidation(t *testing.T) {
	uc := app.NewDispatchVerdict(memorylog.New(), &fakeNotifier{name: "a"})
	err := uc.Handle(context.Background(), domain.Verdict{Decision: "approved"}) // no submission id
	if errs.KindOf(err) != errs.KindValidation {
		t.Fatalf("kind = %v, want validation", errs.KindOf(err))
	}
}

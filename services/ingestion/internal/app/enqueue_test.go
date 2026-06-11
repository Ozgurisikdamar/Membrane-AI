package app_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
	"github.com/Ozgurisikdamar/Membrane-AI/services/ingestion/internal/app"
	"github.com/Ozgurisikdamar/Membrane-AI/services/ingestion/internal/domain"
	"github.com/Ozgurisikdamar/Membrane-AI/services/ingestion/internal/ports"
)

type fakePublisher struct {
	got ports.Event
	err error
}

func (f *fakePublisher) Publish(_ context.Context, e ports.Event) error {
	f.got = e
	return f.err
}

type fixedIDGen struct{ id string }

func (g fixedIDGen) NewID() string { return g.id }

type fixedClock struct{ t time.Time }

func (c fixedClock) Now() time.Time { return c.t }

func mustSub(t *testing.T) domain.Submission {
	t.Helper()
	s, err := domain.NewSubmission("org-1", "repo", "a.go", "go", "diff", domain.OriginIDE)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestEnqueue_PublishesAndReturnsID(t *testing.T) {
	now := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	pub := &fakePublisher{}
	uc := app.NewEnqueueSubmission(pub, fixedIDGen{"sub-123"}, fixedClock{now})

	res, err := uc.Handle(context.Background(), mustSub(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.SubmissionID != "sub-123" || res.Status != app.StatusQueued {
		t.Fatalf("result = %+v", res)
	}
	if pub.got.SubmissionID != "sub-123" || pub.got.OrganizationID != "org-1" {
		t.Fatalf("published event = %+v", pub.got)
	}
	if !pub.got.OccurredAt.Equal(now) {
		t.Fatalf("OccurredAt = %v, want %v", pub.got.OccurredAt, now)
	}
}

func TestEnqueue_PublishFailureIsUnavailable(t *testing.T) {
	pub := &fakePublisher{err: errors.New("broker down")}
	uc := app.NewEnqueueSubmission(pub, fixedIDGen{"x"}, fixedClock{time.Now()})

	_, err := uc.Handle(context.Background(), mustSub(t))
	if err == nil {
		t.Fatal("expected error")
	}
	if errs.KindOf(err) != errs.KindUnavailable {
		t.Fatalf("kind = %v, want unavailable", errs.KindOf(err))
	}
}

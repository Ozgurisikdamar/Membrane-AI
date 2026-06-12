package outboxstore_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/adapters/outboxstore"
)

type fakeShipper struct {
	calls atomic.Int32
	err   error
}

func (f *fakeShipper) PublishPending(_ context.Context, _ int, _ outboxstore.RecordPublisher) (int, error) {
	f.calls.Add(1)
	if f.err != nil {
		return 0, f.err
	}
	return 1, nil
}

func discard() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestRelay_TicksAndStops(t *testing.T) {
	shipper := &fakeShipper{}
	relay := outboxstore.NewRelay(shipper, nil, 10*time.Millisecond, 5, discard())

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { relay.Run(ctx); close(done) }()

	deadline := time.After(2 * time.Second)
	for shipper.calls.Load() < 3 {
		select {
		case <-deadline:
			t.Fatalf("relay ticked %d times, want ≥3", shipper.calls.Load())
		case <-time.After(5 * time.Millisecond):
		}
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("relay did not stop on cancel")
	}
}

func TestRelay_KeepsTickingAfterError(t *testing.T) {
	shipper := &fakeShipper{err: errors.New("kafka down")}
	relay := outboxstore.NewRelay(shipper, nil, 10*time.Millisecond, 5, discard())

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { relay.Run(ctx); close(done) }()

	// Wait until the relay has demonstrably retried after errors (≥2 ticks),
	// with a generous deadline so scheduler hiccups cannot flake the test.
	deadline := time.After(5 * time.Second)
	for shipper.calls.Load() < 2 {
		select {
		case <-deadline:
			t.Fatalf("relay must retry after errors; ticks = %d", shipper.calls.Load())
		case <-time.After(5 * time.Millisecond):
		}
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("relay did not stop on cancel")
	}
}

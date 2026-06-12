package outboxstore_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/adapters/outboxstore"
	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/domain"
)

// TestStore_Integration runs only when MEMBRANE_ORCHESTRATOR_TEST_DSN points at
// a migrated dev database (task dev-up + task migrate). It proves the ACID
// write (audit + outbox in one tx), the relay claim/mark path, and that a
// failed publish keeps rows pending.
func TestStore_Integration(t *testing.T) {
	dsn := os.Getenv("MEMBRANE_ORCHESTRATOR_TEST_DSN")
	if dsn == "" {
		t.Skip("set MEMBRANE_ORCHESTRATOR_TEST_DSN to run the postgres integration test")
	}
	ctx := context.Background()
	topic := "test.verdict.v1"

	store, err := outboxstore.New(ctx, dsn, topic)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.Ping(ctx); err != nil {
		t.Fatal(err)
	}

	org := "itest-" + uuid.NewString()[:8]
	verdict := domain.Verdict{
		SubmissionID:   uuid.NewString(),
		OrganizationID: org,
		Decision:       domain.DecisionRejected,
		Source:         domain.SourcePipeline,
		RulesetVersion: "v1",
		Findings:       []domain.Finding{{Stage: "analyzer", Rule: "x", Severity: domain.SeverityBlocking, Message: "m"}},
		EvaluatedAt:    time.Now().UTC(),
	}
	t.Cleanup(func() { _ = outboxstore.CleanupForTest(context.Background(), store, org) })

	// 1. ACID write: audit row + outbox row.
	if err := store.Publish(ctx, verdict); err != nil {
		t.Fatal(err)
	}
	audits, pending, err := outboxstore.CountForTest(ctx, store, org)
	if err != nil {
		t.Fatal(err)
	}
	if audits != 1 || pending != 1 {
		t.Fatalf("after publish: audits=%d pending=%d, want 1/1", audits, pending)
	}

	// 2. Failed delivery keeps the row pending (tx rollback).
	boom := errors.New("broker down")
	if _, err := store.PublishPending(ctx, 10, func(context.Context, string, []byte, []byte, map[string]string) error {
		return boom
	}); err == nil {
		t.Fatal("expected relay error")
	}
	_, pending, _ = outboxstore.CountForTest(ctx, store, org)
	if pending != 1 {
		t.Fatalf("row must stay pending after failed publish; pending=%d", pending)
	}

	// 3. Successful delivery marks the row published exactly once.
	var gotTopic, gotKey string
	var gotPayload []byte
	n, err := store.PublishPending(ctx, 10, func(_ context.Context, topic string, key, value []byte, _ map[string]string) error {
		gotTopic, gotKey, gotPayload = topic, string(key), value
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("shipped = %d, want 1", n)
	}
	if gotTopic != topic || gotKey != org || len(gotPayload) == 0 {
		t.Fatalf("published topic=%s key=%s payloadLen=%d", gotTopic, gotKey, len(gotPayload))
	}
	_, pending, _ = outboxstore.CountForTest(ctx, store, org)
	if pending != 0 {
		t.Fatalf("pending after delivery = %d, want 0", pending)
	}

	// 4. Idle relay tick is a no-op.
	if n, err := store.PublishPending(ctx, 10, nil); err != nil || n != 0 {
		t.Fatalf("idle tick: n=%d err=%v", n, err)
	}
}

package redislog_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/Ozgurisikdamar/Membrane-AI/services/reporter/internal/adapters/redislog"
)

func newLog(t *testing.T) (*redislog.Log, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mr.Close)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return redislog.NewWithClient(client, time.Hour), mr
}

func TestMarkIfNew_DedupesAcrossCalls(t *testing.T) {
	log, _ := newLog(t)
	ctx := context.Background()

	first, err := log.MarkIfNew(ctx, "sub-1", "github-status")
	if err != nil || !first {
		t.Fatalf("first MarkIfNew = %v, %v; want true, nil", first, err)
	}
	second, err := log.MarkIfNew(ctx, "sub-1", "github-status")
	if err != nil || second {
		t.Fatalf("second MarkIfNew = %v, %v; want false, nil", second, err)
	}
	// A different notifier for the same submission is independent.
	other, err := log.MarkIfNew(ctx, "sub-1", "webhook")
	if err != nil || !other {
		t.Fatalf("other notifier MarkIfNew = %v, %v; want true, nil", other, err)
	}
}

func TestUnmark_AllowsRedelivery(t *testing.T) {
	log, _ := newLog(t)
	ctx := context.Background()

	if _, err := log.MarkIfNew(ctx, "sub-2", "webhook"); err != nil {
		t.Fatal(err)
	}
	if err := log.Unmark(ctx, "sub-2", "webhook"); err != nil {
		t.Fatal(err)
	}
	// After Unmark the key is gone, so the next attempt is "new" again.
	again, err := log.MarkIfNew(ctx, "sub-2", "webhook")
	if err != nil || !again {
		t.Fatalf("after Unmark MarkIfNew = %v, %v; want true, nil", again, err)
	}
}

func TestMarkIfNew_SetsTTL(t *testing.T) {
	log, mr := newLog(t)
	ctx := context.Background()
	if _, err := log.MarkIfNew(ctx, "sub-3", "webhook"); err != nil {
		t.Fatal(err)
	}
	// The marker must expire so the dedup set never grows unbounded.
	mr.FastForward(2 * time.Hour)
	again, err := log.MarkIfNew(ctx, "sub-3", "webhook")
	if err != nil || !again {
		t.Fatalf("after TTL expiry MarkIfNew = %v, %v; want true (key expired), nil", again, err)
	}
}

package memorycache_test

import (
	"context"
	"testing"

	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/adapters/memorycache"
	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/domain"
)

func TestCache_MissThenHit(t *testing.T) {
	c := memorycache.New()
	ctx := context.Background()

	if _, hit, err := c.Get(ctx, "k"); err != nil || hit {
		t.Fatalf("expected clean miss, hit=%v err=%v", hit, err)
	}
	want := domain.Verdict{Decision: domain.DecisionApproved, RulesetVersion: "v1"}
	if err := c.Set(ctx, "k", want); err != nil {
		t.Fatal(err)
	}
	got, hit, err := c.Get(ctx, "k")
	if err != nil || !hit {
		t.Fatalf("expected hit, hit=%v err=%v", hit, err)
	}
	if got.Decision != want.Decision || got.RulesetVersion != want.RulesetVersion {
		t.Fatalf("got %+v", got)
	}
}

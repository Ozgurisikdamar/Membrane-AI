package idgen_test

import (
	"testing"
	"time"

	"github.com/Ozgurisikdamar/Membrane-AI/services/ingestion/internal/adapters/idgen"
)

func TestUUID_NewID_UniqueNonEmpty(t *testing.T) {
	g := idgen.UUID{}
	a, b := g.NewID(), g.NewID()
	if a == "" || b == "" {
		t.Fatal("NewID returned empty")
	}
	if a == b {
		t.Fatal("NewID returned duplicate")
	}
	if len(a) != 36 { // canonical UUID length
		t.Fatalf("unexpected id format: %q", a)
	}
}

func TestSystemClock_NowIsUTCAndCurrent(t *testing.T) {
	now := idgen.SystemClock{}.Now()
	if now.Location() != time.UTC {
		t.Fatalf("Now location = %v, want UTC", now.Location())
	}
	if time.Since(now) > time.Minute {
		t.Fatalf("Now is not current: %v", now)
	}
}

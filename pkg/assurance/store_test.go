package assurance

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
)

func stores(t *testing.T) map[string]EvidenceStore {
	t.Helper()
	fs, err := NewFileStore(filepath.Join(t.TempDir(), "evidence"))
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}
	return map[string]EvidenceStore{"mem": NewMemStore(), "file": fs}
}

func TestEvidenceStore_RoundTripAndDeterministicRegeneration(t *testing.T) {
	t.Parallel()
	for name, s := range stores(t) {
		t.Run(name, func(t *testing.T) {
			ev := validEvidence()
			if err := s.Put(ev); err != nil {
				t.Fatalf("Put: %v", err)
			}
			got, err := s.Get(ev.ReleaseID)
			if err != nil {
				t.Fatalf("Get: %v", err)
			}
			// Regeneration is byte-identical from the stored record.
			orig, _ := RenderJSON(ev)
			round, _ := RenderJSON(got)
			if string(orig) != string(round) {
				t.Fatalf("JSON regeneration not deterministic")
			}
			if RenderMarkdown(ev) != RenderMarkdown(got) {
				t.Fatalf("Markdown regeneration not deterministic")
			}
		})
	}
}

func TestEvidenceStore_Immutable(t *testing.T) {
	t.Parallel()
	for name, s := range stores(t) {
		t.Run(name, func(t *testing.T) {
			ev := validEvidence()
			if err := s.Put(ev); err != nil {
				t.Fatalf("Put: %v", err)
			}
			// Identical re-Put is idempotent.
			if err := s.Put(ev); err != nil {
				t.Fatalf("idempotent re-Put failed: %v", err)
			}
			// A different record under the same release_id is a conflict.
			changed := validEvidence()
			changed.SnapshotHash = "sha256:different"
			if err := s.Put(changed); errs.KindOf(err) != errs.KindConflict {
				t.Fatalf("expected conflict on mutated re-Put, got %v", err)
			}
		})
	}
}

func TestEvidenceStore_ListNewestFirst(t *testing.T) {
	t.Parallel()
	for name, s := range stores(t) {
		t.Run(name, func(t *testing.T) {
			// Empty store lists nothing.
			if got, err := s.List(); err != nil || len(got) != 0 {
				t.Fatalf("empty List = %d records, %v", len(got), err)
			}
			base := validEvidence()
			// Three records, oldest to newest by CreatedAt.
			for i, at := range []time.Time{
				time.Date(2026, 10, 28, 9, 0, 0, 0, time.UTC),
				time.Date(2026, 10, 29, 9, 0, 0, 0, time.UTC),
				time.Date(2026, 10, 30, 9, 0, 0, 0, time.UTC),
			} {
				ev := base
				ev.ReleaseID = "rel-" + string(rune('a'+i))
				ev.CreatedAt = at
				if err := s.Put(ev); err != nil {
					t.Fatalf("Put %d: %v", i, err)
				}
			}
			got, err := s.List()
			if err != nil {
				t.Fatalf("List: %v", err)
			}
			if len(got) != 3 {
				t.Fatalf("List = %d records, want 3", len(got))
			}
			// Newest CreatedAt first.
			if got[0].CreatedAt.Before(got[1].CreatedAt) || got[1].CreatedAt.Before(got[2].CreatedAt) {
				t.Fatalf("List not newest-first: %v, %v, %v", got[0].CreatedAt, got[1].CreatedAt, got[2].CreatedAt)
			}
			if got[0].ReleaseID != "rel-c" {
				t.Fatalf("newest release = %q, want rel-c", got[0].ReleaseID)
			}
		})
	}
}

func TestEvidenceStore_Guards(t *testing.T) {
	t.Parallel()
	for name, s := range stores(t) {
		t.Run(name, func(t *testing.T) {
			if _, err := s.Get("missing"); errs.KindOf(err) != errs.KindNotFound {
				t.Errorf("missing Get kind = %v", errs.KindOf(err))
			}
			// Invalid evidence is refused.
			bad := validEvidence()
			bad.GateDecision = DecisionPass // contradicts the critical-failure status
			if err := s.Put(bad); errs.KindOf(err) != errs.KindValidation {
				t.Errorf("invalid Put kind = %v, want validation", errs.KindOf(err))
			}
		})
	}
}

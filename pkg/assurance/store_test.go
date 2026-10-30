package assurance

import (
	"path/filepath"
	"testing"

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

package assurance

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
)

// EvidenceStore persists release evidence immutably: a release_id maps to
// exactly one record for all time. Put is idempotent for an identical record
// and is a conflict for a different one under the same release_id (work order
// §3: "minimal immutable release evidence"). Get returns KindNotFound when
// absent. Implementations store the canonical RenderJSON bytes so a read-back
// regenerates the report deterministically.
type EvidenceStore interface {
	Put(ev ReleaseEvidenceV1) error
	Get(releaseID string) (ReleaseEvidenceV1, error)
}

// MemStore is an in-memory EvidenceStore for tests and ephemeral runs.
type MemStore struct {
	mu   sync.RWMutex
	data map[string][]byte
}

// NewMemStore builds an empty in-memory store.
func NewMemStore() *MemStore { return &MemStore{data: make(map[string][]byte)} }

// Put stores the record; a differing record under the same release_id is a conflict.
func (m *MemStore) Put(ev ReleaseEvidenceV1) error {
	const op = Op + ".MemStore.Put"
	if err := ev.Validate(); err != nil {
		return errs.Validation(op, "refusing to store invalid evidence", err)
	}
	b, err := RenderJSON(ev)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if existing, ok := m.data[ev.ReleaseID]; ok && !bytes.Equal(existing, b) {
		return errs.Conflict(op, "release evidence is immutable: "+ev.ReleaseID+" already stored with different content", nil)
	}
	m.data[ev.ReleaseID] = b
	return nil
}

// Get returns the stored record or KindNotFound.
func (m *MemStore) Get(releaseID string) (ReleaseEvidenceV1, error) {
	m.mu.RLock()
	b, ok := m.data[releaseID]
	m.mu.RUnlock()
	if !ok {
		return ReleaseEvidenceV1{}, errs.NotFound(Op+".MemStore.Get", "no evidence for release "+releaseID, nil)
	}
	return decodeEvidence(Op+".MemStore.Get", b)
}

// FileStore is a write-once filesystem EvidenceStore: one JSON file per
// release_id under a directory. It never overwrites a record with different
// content.
type FileStore struct {
	dir string
	mu  sync.Mutex
}

// NewFileStore builds a store rooted at dir (created if missing).
func NewFileStore(dir string) (*FileStore, error) {
	if strings.TrimSpace(dir) == "" {
		return nil, errs.Validation(Op+".NewFileStore", "dir is required", nil)
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, errs.Internal(Op+".NewFileStore", "create store dir", err)
	}
	return &FileStore{dir: dir}, nil
}

func (f *FileStore) path(releaseID string) string {
	return filepath.Join(f.dir, safeFileName(releaseID)+".json")
}

// Put writes the canonical record once; an identical re-Put is a no-op, a
// differing one is a conflict (immutability).
func (f *FileStore) Put(ev ReleaseEvidenceV1) error {
	const op = Op + ".FileStore.Put"
	if err := ev.Validate(); err != nil {
		return errs.Validation(op, "refusing to store invalid evidence", err)
	}
	b, err := RenderJSON(ev)
	if err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	p := f.path(ev.ReleaseID)
	if existing, rerr := os.ReadFile(p); rerr == nil { //nolint:gosec // G304: path derived from a sanitized release id under the store dir
		if !bytes.Equal(existing, b) {
			return errs.Conflict(op, "release evidence is immutable: "+ev.ReleaseID+" already stored with different content", nil)
		}
		return nil
	}
	if err := os.WriteFile(p, b, 0o600); err != nil {
		return errs.Internal(op, "write evidence", err)
	}
	return nil
}

// Get reads and decodes the stored record, or KindNotFound.
func (f *FileStore) Get(releaseID string) (ReleaseEvidenceV1, error) {
	const op = Op + ".FileStore.Get"
	b, err := os.ReadFile(f.path(releaseID)) //nolint:gosec // G304: path derived from a sanitized release id under the store dir
	if err != nil {
		if os.IsNotExist(err) {
			return ReleaseEvidenceV1{}, errs.NotFound(op, "no evidence for release "+releaseID, nil)
		}
		return ReleaseEvidenceV1{}, errs.Internal(op, "read evidence", err)
	}
	return decodeEvidence(op, b)
}

func decodeEvidence(op string, b []byte) (ReleaseEvidenceV1, error) {
	var ev ReleaseEvidenceV1
	if err := json.Unmarshal(b, &ev); err != nil {
		return ReleaseEvidenceV1{}, errs.Internal(op, "decode stored evidence", err)
	}
	return ev, nil
}

// safeFileName reduces a release id to a filesystem-safe token.
func safeFileName(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '-', c == '_', c == '.':
			b.WriteByte(c)
		default:
			b.WriteByte('_')
		}
	}
	out := b.String()
	if out == "" {
		return "release"
	}
	return out
}

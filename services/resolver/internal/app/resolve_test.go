package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
	"github.com/Ozgurisikdamar/Membrane-AI/services/resolver/internal/app"
	"github.com/Ozgurisikdamar/Membrane-AI/services/resolver/internal/domain"
)

const orgUUID = "0b7e3f6a-1f2d-4c5b-9e8d-2a1b3c4d5e6f"

type fakeEmbedder struct {
	vec []float32
	err error
}

func (f fakeEmbedder) Embed(context.Context, string) ([]float32, error) { return f.vec, f.err }
func (f fakeEmbedder) Dim() int                                         { return len(f.vec) }

type fakeIndex struct {
	gotOrg, gotLang string
	gotVec          []float32
	gotLimit        int
	matches         []domain.GoldMatch
	err             error
}

func (f *fakeIndex) Search(_ context.Context, org, lang string, vec []float32, limit int) ([]domain.GoldMatch, error) {
	f.gotOrg, f.gotLang, f.gotVec, f.gotLimit = org, lang, vec, limit
	return f.matches, f.err
}

func query(t *testing.T) domain.Query {
	t.Helper()
	q, err := domain.NewQuery(orgUUID, "go", "+diff", 2)
	if err != nil {
		t.Fatal(err)
	}
	return q
}

func TestHandle_EmbedsAndSearches(t *testing.T) {
	idx := &fakeIndex{matches: []domain.GoldMatch{{FilePath: "a.go", CosineDistance: 0.1}}}
	uc := app.NewResolveContext(fakeEmbedder{vec: []float32{1, 0}}, idx)

	got, err := uc.Handle(context.Background(), query(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].FilePath != "a.go" {
		t.Fatalf("matches = %+v", got)
	}
	if idx.gotOrg != orgUUID || idx.gotLang != "go" || idx.gotLimit != 2 || len(idx.gotVec) != 2 {
		t.Fatalf("index received org=%s lang=%s limit=%d vec=%v", idx.gotOrg, idx.gotLang, idx.gotLimit, idx.gotVec)
	}
}

func TestHandle_EmbedFailureIsInternal(t *testing.T) {
	uc := app.NewResolveContext(fakeEmbedder{err: errors.New("boom")}, &fakeIndex{})
	_, err := uc.Handle(context.Background(), query(t))
	if errs.KindOf(err) != errs.KindInternal {
		t.Fatalf("kind = %v, want internal", errs.KindOf(err))
	}
}

func TestHandle_SearchFailureIsUnavailable(t *testing.T) {
	uc := app.NewResolveContext(fakeEmbedder{vec: []float32{1}}, &fakeIndex{err: errors.New("db down")})
	_, err := uc.Handle(context.Background(), query(t))
	if errs.KindOf(err) != errs.KindUnavailable {
		t.Fatalf("kind = %v, want unavailable", errs.KindOf(err))
	}
}

package embed_test

import (
	"context"
	"math"
	"testing"

	"github.com/Ozgurisikdamar/Membrane-AI/services/resolver/internal/adapters/embed"
)

func TestNewStub_RejectsBadDim(t *testing.T) {
	if _, err := embed.NewStub(0); err == nil {
		t.Fatal("dim 0 must be rejected")
	}
}

func TestEmbed_DeterministicAndNormalized(t *testing.T) {
	s, err := embed.NewStub(128)
	if err != nil {
		t.Fatal(err)
	}
	a1, err := s.Embed(context.Background(), "func main() { fmt.Println(1) }")
	if err != nil {
		t.Fatal(err)
	}
	a2, _ := s.Embed(context.Background(), "func main() { fmt.Println(1) }")
	if len(a1) != 128 || s.Dim() != 128 {
		t.Fatalf("dim = %d", len(a1))
	}
	var norm float64
	for i := range a1 {
		if a1[i] != a2[i] {
			t.Fatal("embedding must be deterministic")
		}
		norm += float64(a1[i]) * float64(a1[i])
	}
	if math.Abs(norm-1) > 1e-5 {
		t.Fatalf("norm = %f, want 1", norm)
	}
}

func TestEmbed_SimilarTextsCloserThanDifferent(t *testing.T) {
	s, _ := embed.NewStub(512)
	ctx := context.Background()
	base, _ := s.Embed(ctx, "open the database connection pool with retries and timeouts configured")
	similar, _ := s.Embed(ctx, "open the database connection pool with retries and backoff configured")
	different, _ := s.Embed(ctx, "render the html template for the user profile page header")

	if cos(base, similar) <= cos(base, different) {
		t.Fatalf("similarity ordering broken: sim=%f diff=%f", cos(base, similar), cos(base, different))
	}
}

func TestEmbed_EmptyAndShortText(t *testing.T) {
	s, _ := embed.NewStub(64)
	v, err := s.Embed(context.Background(), "   ")
	if err != nil || len(v) != 64 {
		t.Fatalf("empty text: v=%d err=%v", len(v), err)
	}
	short, err := s.Embed(context.Background(), "one two") // below shingle size
	if err != nil {
		t.Fatal(err)
	}
	var nonzero bool
	for _, x := range short {
		if x != 0 {
			nonzero = true
			break
		}
	}
	if !nonzero {
		t.Fatal("short text must still produce a signal")
	}
}

func cos(a, b []float32) float64 {
	var dot float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
	}
	return dot // inputs are L2-normalized
}

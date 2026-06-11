package postgres_test

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"

	"github.com/Ozgurisikdamar/Membrane-AI/services/resolver/internal/adapters/embed"
	"github.com/Ozgurisikdamar/Membrane-AI/services/resolver/internal/adapters/postgres"
)

func TestVectorLiteral(t *testing.T) {
	got := postgres.VectorLiteral([]float32{1, -0.5, 0})
	want := "[1,-0.5,0]"
	if got != want {
		t.Fatalf("VectorLiteral = %q, want %q", got, want)
	}
}

// TestSearch_Integration runs only when MEMBRANE_RESOLVER_TEST_DSN points at a
// migrated dev database (task dev-up + task migrate). It seeds a tenant with
// two gold rows and asserts similarity ordering through the real halfvec query.
func TestSearch_Integration(t *testing.T) {
	dsn := os.Getenv("MEMBRANE_RESOLVER_TEST_DSN")
	if dsn == "" {
		t.Skip("set MEMBRANE_RESOLVER_TEST_DSN to run the postgres integration test")
	}
	ctx := context.Background()
	const dim = 3072

	idx, err := postgres.New(ctx, dsn, dim)
	if err != nil {
		t.Fatal(err)
	}
	defer idx.Close()
	if err := idx.Ping(ctx); err != nil {
		t.Fatal(err)
	}

	stub, err := embed.NewStub(dim)
	if err != nil {
		t.Fatal(err)
	}

	org := uuid.NewString()
	seed := func(path, content string) {
		vec, err := stub.Embed(ctx, content)
		if err != nil {
			t.Fatal(err)
		}
		if err := postgres.SeedForTest(ctx, idx, org, path, "go", content, postgres.VectorLiteral(vec)); err != nil {
			t.Fatalf("seed %s: %v", path, err)
		}
	}
	t.Cleanup(func() { _ = postgres.CleanupForTest(context.Background(), idx, org) })

	seed("db/pool.go", "open the database connection pool with retries and timeouts configured")
	seed("web/render.go", "render the html template for the user profile page header")

	query, err := stub.Embed(ctx, "open the database connection pool with retries and backoff configured")
	if err != nil {
		t.Fatal(err)
	}
	matches, err := idx.Search(ctx, org, "go", query, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 2 {
		t.Fatalf("matches = %d, want 2", len(matches))
	}
	if matches[0].FilePath != "db/pool.go" {
		t.Fatalf("nearest = %s, want db/pool.go (distances: %f vs %f)",
			matches[0].FilePath, matches[0].CosineDistance, matches[1].CosineDistance)
	}
	if matches[0].CosineDistance >= matches[1].CosineDistance {
		t.Fatalf("ordering broken: %f >= %f", matches[0].CosineDistance, matches[1].CosineDistance)
	}

	// Dimension mismatch is rejected before touching the database.
	if _, err := idx.Search(ctx, org, "go", []float32{1, 2}, 1); err == nil {
		t.Fatal("dimension mismatch must error")
	}
}

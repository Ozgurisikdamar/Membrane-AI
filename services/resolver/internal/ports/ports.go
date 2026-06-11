// Package ports declares the interfaces the resolver application depends on,
// defined on the consumer side (SOLID I/D).
package ports

import (
	"context"

	"github.com/Ozgurisikdamar/Membrane-AI/services/resolver/internal/domain"
)

// Embedder turns text into a dense vector. The production implementation will
// call the semantic service / embedding API; until then a deterministic stub
// stands in (DECISIONS D-020) — the swap point is exactly this port.
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
	// Dim returns the embedding dimensionality (must match the index schema).
	Dim() int
}

// GoldIndex searches the organization's gold-codebase index by vector
// similarity.
type GoldIndex interface {
	Search(ctx context.Context, orgID, language string, embedding []float32, limit int) ([]domain.GoldMatch, error)
}

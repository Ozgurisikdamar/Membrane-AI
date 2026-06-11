// Package embed provides Embedder adapters. Stub is a deterministic,
// dependency-free embedder that stands in until the semantic service exposes
// real embeddings (DECISIONS D-020): it hashes overlapping token shingles into
// a fixed-dimension bag-of-features vector and L2-normalizes it. Similar texts
// share shingles, so cosine distance is meaningful enough for plumbing, tests
// and demos — it is NOT a semantic embedding.
package embed

import (
	"context"
	"hash/fnv"
	"math"
	"strings"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
)

const op = "resolver.adapters.embed.Stub"

// shingleSize is the number of consecutive tokens hashed together.
const shingleSize = 3

// Stub is a deterministic hash-based embedder.
type Stub struct {
	dim int
}

// NewStub returns a Stub producing dim-dimensional vectors.
func NewStub(dim int) (*Stub, error) {
	if dim <= 0 {
		return nil, errs.Validation(op, "dimension must be positive", nil)
	}
	return &Stub{dim: dim}, nil
}

// Dim implements ports.Embedder.
func (s *Stub) Dim() int { return s.dim }

// Embed implements ports.Embedder deterministically and without I/O.
func (s *Stub) Embed(ctx context.Context, text string) ([]float32, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	vec := make([]float32, s.dim)
	tokens := strings.Fields(strings.ToLower(text))
	if len(tokens) == 0 {
		return vec, nil
	}
	for i := 0; i+shingleSize <= len(tokens) || (i == 0 && len(tokens) < shingleSize); i++ {
		end := i + shingleSize
		if end > len(tokens) {
			end = len(tokens)
		}
		h := fnv.New64a()
		_, _ = h.Write([]byte(strings.Join(tokens[i:end], " ")))
		sum := h.Sum64()
		bucket := int(sum % uint64(s.dim)) //nolint:gosec // dim is a small positive int
		// Sign from one hash bit de-correlates buckets (signed feature hashing).
		if sum&(1<<63) != 0 {
			vec[bucket]--
		} else {
			vec[bucket]++
		}
	}
	normalize(vec)
	return vec, nil
}

func normalize(v []float32) {
	var sum float64
	for _, x := range v {
		sum += float64(x) * float64(x)
	}
	if sum == 0 {
		return
	}
	n := float32(math.Sqrt(sum))
	for i := range v {
		v[i] /= n
	}
}

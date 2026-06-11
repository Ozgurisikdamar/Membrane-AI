// Package postgres implements the GoldIndex port against Aurora/PostgreSQL with
// pgvector. The similarity query uses the halfvec expression form so it hits
// the HNSW index created by migration 0001 (see the index comment there).
package postgres

import (
	"context"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
	"github.com/Ozgurisikdamar/Membrane-AI/services/resolver/internal/domain"
)

const op = "resolver.adapters.postgres"

// GoldIndex queries gold_codebase_index by vector similarity.
type GoldIndex struct {
	pool *pgxpool.Pool
	dim  int
}

// New connects a pool and returns the adapter. dim must match the schema's
// embedding dimensionality (3072 in migration 0001).
func New(ctx context.Context, databaseURL string, dim int) (*GoldIndex, error) {
	if dim <= 0 {
		return nil, errs.Validation(op, "dimension must be positive", nil)
	}
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, errs.Unavailable(op, "connect postgres", err)
	}
	return &GoldIndex{pool: pool, dim: dim}, nil
}

// searchSQL orders by the same halfvec expression the HNSW index is built on.
const searchSQL = `
SELECT file_path, raw_code_content, architectural_context,
       (embedding::halfvec(%DIM%) <=> $1::halfvec(%DIM%))::float8 AS cosine_distance
FROM gold_codebase_index
WHERE org_id = $2 AND language_tag = $3
ORDER BY embedding::halfvec(%DIM%) <=> $1::halfvec(%DIM%)
LIMIT $4`

// Search implements ports.GoldIndex.
func (g *GoldIndex) Search(ctx context.Context, orgID, language string, embedding []float32, limit int) ([]domain.GoldMatch, error) {
	if len(embedding) != g.dim {
		return nil, errs.Validation(op, "embedding dimension mismatch", nil)
	}
	sql := strings.ReplaceAll(searchSQL, "%DIM%", strconv.Itoa(g.dim))
	rows, err := g.pool.Query(ctx, sql, VectorLiteral(embedding), orgID, language, limit)
	if err != nil {
		return nil, errs.Unavailable(op, "similarity query", err)
	}
	defer rows.Close()

	var matches []domain.GoldMatch
	for rows.Next() {
		var m domain.GoldMatch
		if err := rows.Scan(&m.FilePath, &m.RawCodeContent, &m.ArchitecturalContext, &m.CosineDistance); err != nil {
			return nil, errs.Internal(op, "scan match", err)
		}
		matches = append(matches, m)
	}
	if err := rows.Err(); err != nil {
		return nil, errs.Unavailable(op, "iterate matches", err)
	}
	return matches, nil
}

// Ping checks connectivity for readiness probes.
func (g *GoldIndex) Ping(ctx context.Context) error {
	if err := g.pool.Ping(ctx); err != nil {
		return errs.Unavailable(op, "ping postgres", err)
	}
	return nil
}

// Close releases the pool.
func (g *GoldIndex) Close() { g.pool.Close() }

// VectorLiteral renders a pgvector text literal ("[x1,x2,…]"). Exported for
// tests and future writer adapters.
func VectorLiteral(v []float32) string {
	var b strings.Builder
	b.Grow(len(v)*10 + 2)
	b.WriteByte('[')
	for i, x := range v {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.FormatFloat(float64(x), 'f', -1, 32))
	}
	b.WriteByte(']')
	return b.String()
}

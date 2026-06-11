// Package app holds the resolver use-cases.
package app

import (
	"context"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
	"github.com/Ozgurisikdamar/Membrane-AI/services/resolver/internal/domain"
	"github.com/Ozgurisikdamar/Membrane-AI/services/resolver/internal/ports"
)

const opResolve = "resolver.app.ResolveContext"

// ResolveContext embeds the query diff and retrieves the organization's most
// similar gold-codebase snippets.
type ResolveContext struct {
	embedder ports.Embedder
	index    ports.GoldIndex
}

// NewResolveContext wires the use-case.
func NewResolveContext(e ports.Embedder, idx ports.GoldIndex) *ResolveContext {
	return &ResolveContext{embedder: e, index: idx}
}

// Handle runs the RAG retrieval for q.
func (uc *ResolveContext) Handle(ctx context.Context, q domain.Query) ([]domain.GoldMatch, error) {
	vec, err := uc.embedder.Embed(ctx, q.Diff)
	if err != nil {
		return nil, errs.Internal(opResolve, "embed query", err)
	}
	matches, err := uc.index.Search(ctx, q.OrganizationID, q.Language, vec, q.Limit)
	if err != nil {
		return nil, errs.Unavailable(opResolve, "gold-index search", err)
	}
	return matches, nil
}

package assurance

import (
	"context"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
)

// AgentTwinClient is the port the Membrane adapter depends on to read a paired
// evaluation result from AgentTwin. The HTTP adapter (Phase 2 runtime) and the
// deterministic FakeClient (tests/demo) both implement it; the aggregator
// (BuildReleaseEvidence) depends only on this interface, keeping the core pure
// (hexagonal; dependencies point inward).
type AgentTwinClient interface {
	// GetEvalResult returns the minimal, reference-only result for a paired
	// evaluation run, or a typed error (KindNotFound / KindUnavailable / ...).
	GetEvalResult(ctx context.Context, evalRunID string) (EvalResultRef, error)
}

// FakeClient is a deterministic in-memory AgentTwinClient for tests and the
// offline demo. It never performs I/O and must be labelled as a fake wherever
// used (it is not a real AgentTwin).
type FakeClient struct {
	Results map[string]EvalResultRef
}

// NewFakeClient builds a FakeClient seeded with the given results, keyed by
// evaluation run ID.
func NewFakeClient(results ...EvalResultRef) *FakeClient {
	m := make(map[string]EvalResultRef, len(results))
	for _, r := range results {
		m[r.EvaluationRunID] = r
	}
	return &FakeClient{Results: m}
}

// GetEvalResult returns the seeded result or a typed not-found error.
func (f *FakeClient) GetEvalResult(_ context.Context, evalRunID string) (EvalResultRef, error) {
	if r, ok := f.Results[evalRunID]; ok {
		return r, nil
	}
	return EvalResultRef{}, errs.NotFound(Op+".FakeClient.GetEvalResult", "no such evaluation run: "+evalRunID, nil)
}

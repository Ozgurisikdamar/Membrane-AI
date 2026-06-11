package app_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/app"
	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/domain"
	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/ports"
)

// --- fakes -----------------------------------------------------------------

type fakeCache struct {
	store    map[string]domain.Verdict
	getErr   error
	setErr   error
	setCalls int
}

func newFakeCache() *fakeCache { return &fakeCache{store: map[string]domain.Verdict{}} }

func (c *fakeCache) Get(_ context.Context, key string) (domain.Verdict, bool, error) {
	if c.getErr != nil {
		return domain.Verdict{}, false, c.getErr
	}
	v, ok := c.store[key]
	return v, ok, nil
}

func (c *fakeCache) Set(_ context.Context, key string, v domain.Verdict) error {
	c.setCalls++
	if c.setErr != nil {
		return c.setErr
	}
	c.store[key] = v
	return nil
}

type fakeStage struct {
	name       string
	findings   []domain.Finding
	maskedDiff string
	err        error
	delay      time.Duration
	seenDiff   *string // when set, records the diff this stage received
}

func (s fakeStage) Name() string { return s.name }

func (s fakeStage) Analyze(ctx context.Context, sub domain.Submission) (ports.StageResult, error) {
	if s.seenDiff != nil {
		*s.seenDiff = sub.Diff
	}
	if s.delay > 0 {
		select {
		case <-time.After(s.delay):
		case <-ctx.Done():
			return ports.StageResult{}, ctx.Err()
		}
	}
	return ports.StageResult{Findings: s.findings, MaskedDiff: s.maskedDiff}, s.err
}

type fakePublisher struct {
	published []domain.Verdict
	err       error
}

func (p *fakePublisher) Publish(_ context.Context, v domain.Verdict) error {
	if p.err != nil {
		return p.err
	}
	p.published = append(p.published, v)
	return nil
}

type fixedClock struct{ t time.Time }

func (c fixedClock) Now() time.Time { return c.t }

// --- helpers ----------------------------------------------------------------

var now = time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)

func sub(t *testing.T) domain.Submission {
	t.Helper()
	s, err := domain.NewSubmission("sub-1", "org-1", "repo", "a.go", "go", "some diff", "ide", now)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func newUC(cache ports.VerdictCache, stages []ports.AnalysisStage, fb ports.AnalysisStage, pub ports.VerdictPublisher, opts app.Options) *app.ProcessSubmission {
	return app.NewProcessSubmission(cache, stages, fb, pub, fixedClock{now}, opts)
}

// --- tests -------------------------------------------------------------------

func TestHandle_CacheHit_RepublishesWithoutPipeline(t *testing.T) {
	cache := newFakeCache()
	key := domain.CacheKey("some diff", "v1")
	cache.store[key] = domain.Verdict{
		Decision: domain.DecisionApproved, Source: domain.SourcePipeline, RulesetVersion: "v1",
	}
	exploding := fakeStage{name: "boom", err: errors.New("must not run")}
	pub := &fakePublisher{}
	uc := newUC(cache, []ports.AnalysisStage{exploding}, exploding, pub, app.Options{})

	v, err := uc.Handle(context.Background(), sub(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.Source != domain.SourceCache || v.Decision != domain.DecisionApproved {
		t.Fatalf("verdict = %+v", v)
	}
	if v.SubmissionID != "sub-1" || v.OrganizationID != "org-1" {
		t.Fatalf("cached verdict must be re-addressed to the live submission: %+v", v)
	}
	if len(pub.published) != 1 {
		t.Fatalf("published %d times", len(pub.published))
	}
	if cache.setCalls != 0 {
		t.Fatal("cache hit must not re-store")
	}
}

func TestHandle_Miss_PipelineApproves_AndCaches(t *testing.T) {
	cache := newFakeCache()
	stage := fakeStage{name: "ast", findings: []domain.Finding{{Stage: "ast", Severity: domain.SeverityInfo}}}
	pub := &fakePublisher{}
	uc := newUC(cache, []ports.AnalysisStage{stage}, fakeStage{name: "fb"}, pub, app.Options{})

	v, err := uc.Handle(context.Background(), sub(t))
	if err != nil {
		t.Fatal(err)
	}
	if v.Source != domain.SourcePipeline || v.Decision != domain.DecisionApproved {
		t.Fatalf("verdict = %+v", v)
	}
	if cache.setCalls != 1 {
		t.Fatalf("setCalls = %d, want 1", cache.setCalls)
	}
	// The cached copy must be address-free so a different submission with the
	// same diff can reuse it.
	cached := cache.store[domain.CacheKey("some diff", "v1")]
	if cached.SubmissionID != "" || cached.OrganizationID != "" {
		t.Fatalf("cached verdict must not carry submission identity: %+v", cached)
	}
}

func TestHandle_Miss_BlockingFindingRejects(t *testing.T) {
	stage := fakeStage{name: "ast", findings: []domain.Finding{{Stage: "ast", Severity: domain.SeverityBlocking}}}
	pub := &fakePublisher{}
	uc := newUC(newFakeCache(), []ports.AnalysisStage{stage}, fakeStage{name: "fb"}, pub, app.Options{})

	v, err := uc.Handle(context.Background(), sub(t))
	if err != nil {
		t.Fatal(err)
	}
	if v.Decision != domain.DecisionRejected {
		t.Fatalf("decision = %v, want rejected", v.Decision)
	}
}

func TestHandle_StageTimeout_FallsBack(t *testing.T) {
	slow := fakeStage{name: "slow", delay: 200 * time.Millisecond}
	fb := fakeStage{name: "fb", findings: []domain.Finding{{Stage: "fb", Severity: domain.SeverityWarning}}}
	pub := &fakePublisher{}
	uc := newUC(newFakeCache(), []ports.AnalysisStage{slow}, fb, pub,
		app.Options{StageDeadline: 30 * time.Millisecond})

	v, err := uc.Handle(context.Background(), sub(t))
	if err != nil {
		t.Fatal(err)
	}
	if v.Source != domain.SourceFallback {
		t.Fatalf("source = %v, want fallback", v.Source)
	}
	if v.Decision != domain.DecisionNeedsReview {
		t.Fatalf("decision = %v, want needs_review", v.Decision)
	}
}

func TestHandle_StageError_FallsBack_FallbackFails_NeedsReview(t *testing.T) {
	bad := fakeStage{name: "bad", err: errors.New("stage exploded")}
	fbBad := fakeStage{name: "fb", err: errors.New("fallback exploded")}
	pub := &fakePublisher{}
	uc := newUC(newFakeCache(), []ports.AnalysisStage{bad}, fbBad, pub, app.Options{})

	v, err := uc.Handle(context.Background(), sub(t))
	if err != nil {
		t.Fatal(err)
	}
	if v.Source != domain.SourceFallback || v.Decision != domain.DecisionNeedsReview {
		t.Fatalf("verdict = %+v", v)
	}
	if len(v.Findings) == 0 || v.Findings[len(v.Findings)-1].Rule != "fallback-error" {
		t.Fatalf("expected a fallback-error finding, got %+v", v.Findings)
	}
}

func TestHandle_FallbackVerdictIsNotCached(t *testing.T) {
	cache := newFakeCache()
	bad := fakeStage{name: "bad", err: errors.New("stage exploded")}
	fb := fakeStage{name: "fb"}
	uc := newUC(cache, []ports.AnalysisStage{bad}, fb, &fakePublisher{}, app.Options{})

	if _, err := uc.Handle(context.Background(), sub(t)); err != nil {
		t.Fatal(err)
	}
	if cache.setCalls != 0 {
		t.Fatal("fallback verdicts must not be cached (a degraded answer would stick for 72h)")
	}
}

func TestHandle_CacheReadErrorIsNonFatal(t *testing.T) {
	cache := newFakeCache()
	cache.getErr = errors.New("redis down")
	stage := fakeStage{name: "ast"}
	pub := &fakePublisher{}
	uc := newUC(cache, []ports.AnalysisStage{stage}, fakeStage{name: "fb"}, pub, app.Options{})

	v, err := uc.Handle(context.Background(), sub(t))
	if err != nil {
		t.Fatalf("cache read error must not fail the saga: %v", err)
	}
	if v.Source != domain.SourcePipeline {
		t.Fatalf("source = %v, want pipeline", v.Source)
	}
}

func TestHandle_CacheWriteErrorIsNonFatal(t *testing.T) {
	cache := newFakeCache()
	cache.setErr = errors.New("redis down")
	uc := newUC(cache, []ports.AnalysisStage{fakeStage{name: "ast"}}, fakeStage{name: "fb"}, &fakePublisher{}, app.Options{})

	if _, err := uc.Handle(context.Background(), sub(t)); err != nil {
		t.Fatalf("cache write error must not fail the saga: %v", err)
	}
}

func TestHandle_PublishFailureIsUnavailable(t *testing.T) {
	pub := &fakePublisher{err: errors.New("broker down")}
	uc := newUC(newFakeCache(), []ports.AnalysisStage{fakeStage{name: "ast"}}, fakeStage{name: "fb"}, pub, app.Options{})

	_, err := uc.Handle(context.Background(), sub(t))
	if errs.KindOf(err) != errs.KindUnavailable {
		t.Fatalf("kind = %v, want unavailable", errs.KindOf(err))
	}
}

func TestHandle_MaskedDiffFlowsToLaterStages(t *testing.T) {
	var semanticSaw string
	masker := fakeStage{name: "analyzer", maskedDiff: "+key := \"[MASKED:aws]\""}
	semantic := fakeStage{name: "semantic", seenDiff: &semanticSaw}
	pub := &fakePublisher{}
	uc := newUC(newFakeCache(), []ports.AnalysisStage{masker, semantic}, fakeStage{name: "fb"}, pub, app.Options{})

	if _, err := uc.Handle(context.Background(), sub(t)); err != nil {
		t.Fatal(err)
	}
	if semanticSaw != "+key := \"[MASKED:aws]\"" {
		t.Fatalf("semantic stage must receive the MASKED diff (D-024), got %q", semanticSaw)
	}
	// The cache key must still derive from the ORIGINAL diff: a resubmission of
	// the same raw diff has to hit the cache.
	if pub.published[0].Source != domain.SourcePipeline {
		t.Fatalf("source = %v", pub.published[0].Source)
	}
}

func TestHandle_SourceCoordinatesStampedAndNeverCached(t *testing.T) {
	cache := newFakeCache()
	pub := &fakePublisher{}
	uc := newUC(cache, []ports.AnalysisStage{fakeStage{name: "ast"}}, fakeStage{name: "fb"}, pub, app.Options{})

	s := sub(t).WithSource("abc123", 42)
	v, err := uc.Handle(context.Background(), s)
	if err != nil {
		t.Fatal(err)
	}
	if v.Repository != "repo" || v.CommitSHA != "abc123" || v.PRNumber != 42 {
		t.Fatalf("verdict missing source coords: %+v", v)
	}
	cached := cache.store[domain.CacheKey("some diff", "v1")]
	if cached.Repository != "" || cached.CommitSHA != "" || cached.PRNumber != 0 {
		t.Fatalf("cached verdict must not carry source coords (cross-repo reuse!): %+v", cached)
	}

	// Cache hit from a DIFFERENT repo/commit must be re-stamped with the live
	// submission's coordinates, never the original ones.
	other, _ := domain.NewSubmission("sub-2", "org-2", "other-repo", "b.go", "go", "some diff", "ide", now)
	other = other.WithSource("def456", 7)
	v2, err := uc.Handle(context.Background(), other)
	if err != nil {
		t.Fatal(err)
	}
	if v2.Source != domain.SourceCache || v2.Repository != "other-repo" || v2.CommitSHA != "def456" || v2.PRNumber != 7 {
		t.Fatalf("cache-hit verdict not re-stamped: %+v", v2)
	}
}

func TestHandle_NoStages_Approves(t *testing.T) {
	pub := &fakePublisher{}
	uc := newUC(newFakeCache(), nil, fakeStage{name: "fb"}, pub, app.Options{})

	v, err := uc.Handle(context.Background(), sub(t))
	if err != nil {
		t.Fatal(err)
	}
	if v.Decision != domain.DecisionApproved || v.Source != domain.SourcePipeline {
		t.Fatalf("verdict = %+v", v)
	}
	if !v.EvaluatedAt.Equal(now) {
		t.Fatalf("EvaluatedAt = %v, want %v", v.EvaluatedAt, now)
	}
}

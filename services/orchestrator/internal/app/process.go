// Package app holds the orchestrator use-cases. ProcessSubmission implements
// the analysis Saga: cache-first, staged analysis under a hard deadline, and a
// deterministic fallback so the developer is never blocked (ARCHITECTURE §3).
package app

import (
	"context"
	"time"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/domain"
	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/ports"
)

const opProcess = "orchestrator.app.ProcessSubmission"

// Options tune the Saga. Zero values fall back to safe defaults.
type Options struct {
	// StageDeadline bounds the whole staged-analysis phase (default 1200 ms,
	// the IDE-mode hard deadline).
	StageDeadline time.Duration
	// FallbackDeadline bounds the deterministic fallback (default 200 ms).
	FallbackDeadline time.Duration
	// RulesetVersion is folded into the cache key so verdicts invalidate when
	// rules change (default "v1").
	RulesetVersion string
}

func (o Options) withDefaults() Options {
	if o.StageDeadline <= 0 {
		o.StageDeadline = 1200 * time.Millisecond
	}
	if o.FallbackDeadline <= 0 {
		o.FallbackDeadline = 200 * time.Millisecond
	}
	if o.RulesetVersion == "" {
		o.RulesetVersion = "v1"
	}
	return o
}

// ProcessSubmission is the Saga use-case for one consumed submission.
type ProcessSubmission struct {
	cache     ports.VerdictCache
	stages    []ports.AnalysisStage // ordered Strategy chain
	fallback  ports.AnalysisStage   // deterministic fallback on deadline/stage failure
	publisher ports.VerdictPublisher
	clock     ports.Clock
	opts      Options
}

// NewProcessSubmission wires the Saga. All dependencies are required; stages
// may be empty (the verdict is then approved unless the fallback says otherwise).
func NewProcessSubmission(
	cache ports.VerdictCache,
	stages []ports.AnalysisStage,
	fallback ports.AnalysisStage,
	publisher ports.VerdictPublisher,
	clock ports.Clock,
	opts Options,
) *ProcessSubmission {
	return &ProcessSubmission{
		cache:     cache,
		stages:    stages,
		fallback:  fallback,
		publisher: publisher,
		clock:     clock,
		opts:      opts.withDefaults(),
	}
}

// Handle runs the Saga for sub and returns the published verdict.
//
//	cache hit  → republish cached verdict (Source=cache)
//	cache miss → run stages under StageDeadline → consolidate (Source=pipeline)
//	stage error/timeout → deterministic fallback (Source=fallback)
func (uc *ProcessSubmission) Handle(ctx context.Context, sub domain.Submission) (domain.Verdict, error) {
	key := domain.CacheKey(sub.Diff, uc.opts.RulesetVersion)

	if cached, hit, err := uc.cache.Get(ctx, key); err == nil && hit {
		verdict := cached
		verdict.SubmissionID = sub.SubmissionID
		verdict.OrganizationID = sub.OrganizationID
		verdict.Source = domain.SourceCache
		if err := uc.publisher.Publish(ctx, verdict); err != nil {
			return domain.Verdict{}, errs.Unavailable(opProcess, "publish cached verdict", err)
		}
		return verdict, nil
	}
	// A cache read error is deliberately non-fatal: the pipeline below is the
	// source of truth and a degraded cache must never block analysis.

	verdict := uc.analyze(ctx, sub)

	// Cache writes are best-effort: a miss next time costs latency, not
	// correctness, and a degraded cache must never fail the Saga.
	if verdict.Source == domain.SourcePipeline {
		cacheable := verdict
		cacheable.SubmissionID = ""
		cacheable.OrganizationID = ""
		_ = uc.cache.Set(ctx, key, cacheable)
	}

	if err := uc.publisher.Publish(ctx, verdict); err != nil {
		return domain.Verdict{}, errs.Unavailable(opProcess, "publish verdict", err)
	}
	return verdict, nil
}

// analyze runs the staged pipeline; on any stage failure or deadline breach it
// degrades to the deterministic fallback (never returns an error: the Saga
// always yields a verdict). A stage that returns a masked diff rewrites the
// artifact for every later stage (D-024).
func (uc *ProcessSubmission) analyze(ctx context.Context, sub domain.Submission) domain.Verdict {
	stageCtx, cancel := context.WithTimeout(ctx, uc.opts.StageDeadline)
	defer cancel()

	var findings []domain.Finding
	current := sub // local copy: stages may redact the diff for later stages
	for _, stage := range uc.stages {
		out, err := stage.Analyze(stageCtx, current)
		if err != nil {
			return uc.runFallback(ctx, sub)
		}
		findings = append(findings, out.Findings...)
		if out.MaskedDiff != "" {
			current.Diff = out.MaskedDiff
		}
	}

	return domain.Verdict{
		SubmissionID:   sub.SubmissionID,
		OrganizationID: sub.OrganizationID,
		Decision:       domain.Consolidate(findings),
		Source:         domain.SourcePipeline,
		RulesetVersion: uc.opts.RulesetVersion,
		Findings:       findings,
		EvaluatedAt:    uc.clock.Now(),
	}
}

// runFallback executes the deterministic fallback with its own fresh deadline
// (the stage context may already be expired).
func (uc *ProcessSubmission) runFallback(ctx context.Context, sub domain.Submission) domain.Verdict {
	fbCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), uc.opts.FallbackDeadline)
	defer cancel()

	out, err := uc.fallback.Analyze(fbCtx, sub)
	findings := out.Findings
	decision := domain.Consolidate(findings)
	if err != nil {
		// Even the fallback failed: fail safe — demand human review.
		decision = domain.DecisionNeedsReview
		findings = append(findings, domain.Finding{
			Stage:    uc.fallback.Name(),
			Rule:     "fallback-error",
			Severity: domain.SeverityWarning,
			Message:  "deterministic fallback failed; manual review required",
		})
	}
	return domain.Verdict{
		SubmissionID:   sub.SubmissionID,
		OrganizationID: sub.OrganizationID,
		Decision:       decision,
		Source:         domain.SourceFallback,
		RulesetVersion: uc.opts.RulesetVersion,
		Findings:       findings,
		EvaluatedAt:    uc.clock.Now(),
	}
}

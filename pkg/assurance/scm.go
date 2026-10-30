package assurance

import (
	"context"
	"strings"
	"sync"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
)

// SCMMode is the enforcement posture for publishing the combined release gate,
// mirroring the Membrane reporter's shadow→enforce rollout (D-035). In shadow
// mode a BLOCK is surfaced but does NOT block the merge; enforcement is opt-in.
type SCMMode string

// SCM modes.
const (
	ModeShadow  SCMMode = "shadow"
	ModeEnforce SCMMode = "enforce"
)

// SCMOutcome is the report-level state, 1:1 with commit-status semantics.
type SCMOutcome string

// SCM outcomes.
const (
	OutcomeSuccess SCMOutcome = "success" // PASS
	OutcomeFailure SCMOutcome = "failure" // BLOCK in enforce mode
	OutcomeNeutral SCMOutcome = "neutral" // WARN / INCOMPLETE / ERROR / shadowed BLOCK (non-blocking, visible)
)

// SCMStatus is the publishable representation of a release decision.
type SCMStatus struct {
	Repository string
	CommitSHA  string
	Context    string
	Outcome    SCMOutcome
	// State is the commit-status state for the outcome (success/failure/pending).
	State string
	Title string
	Body  string
}

// StatusContext is the commit-status context this integration publishes under.
const StatusContext = "membrane-ai/release-assurance"

// MapSCMStatus maps a release evidence record to a publishable SCM status under
// the given mode. It is deterministic and fail-safe: only a PASS gate is a
// success; BLOCK is a failure in enforce mode and a non-blocking neutral
// ("would block") in shadow mode; WARN/INCOMPLETE/ERROR are visible neutrals and
// never a success (work order §4, §5 Phase 4).
func MapSCMStatus(ev ReleaseEvidenceV1, mode SCMMode) SCMStatus {
	rules := joinOr(ev.TriggeredRules, "no rules")
	var outcome SCMOutcome
	var title string
	switch ev.GateDecision {
	case DecisionPass:
		outcome, title = OutcomeSuccess, "MEMBRANE.AI ✓ release checks passed"
	case DecisionWarn:
		outcome, title = OutcomeNeutral, "MEMBRANE.AI ⚠ release warnings — review ("+rules+")"
	case DecisionBlock:
		outcome, title = OutcomeFailure, "MEMBRANE.AI ✗ release blocked ("+rules+")"
	case DecisionIncomplete:
		outcome, title = OutcomeNeutral, "MEMBRANE.AI ⚠ release evidence incomplete — not a pass"
	case DecisionError:
		outcome, title = OutcomeNeutral, "MEMBRANE.AI ⚠ release evaluation errored — not a pass"
	default:
		outcome, title = OutcomeNeutral, "MEMBRANE.AI ⚠ release status unknown — not a pass"
	}

	// Shadow mode: a BLOCK is downgraded to a non-blocking neutral and labelled
	// (observe before enforcing). Enforcement is opt-in.
	if mode == ModeShadow && outcome == OutcomeFailure {
		outcome = OutcomeNeutral
		title = "MEMBRANE.AI ⚠ shadow mode — would BLOCK in enforce mode (" + rules + ")"
	}

	return SCMStatus{
		Repository: ev.Repository,
		CommitSHA:  ev.CommitSHA,
		Context:    StatusContext,
		Outcome:    outcome,
		State:      stateFor(outcome),
		Title:      title,
		Body:       scmBody(ev),
	}
}

// stateFor renders a commit-status state (no "neutral" in the statuses API, so
// a neutral outcome maps to the non-blocking "pending", mirroring the reporter).
func stateFor(o SCMOutcome) string {
	switch o {
	case OutcomeSuccess:
		return "success"
	case OutcomeFailure:
		return "failure"
	default:
		return "pending"
	}
}

func scmBody(ev ReleaseEvidenceV1) string {
	var b strings.Builder
	b.WriteString("Combined release gate: " + string(ev.GateDecision) + "\n")
	b.WriteString("Code verdict: " + string(ev.CodeVerdict) + " · agent status: " + string(ev.OverallStatus) + "\n")
	b.WriteString("Triggered rules: " + joinOr(ev.TriggeredRules, "none") + "\n")
	b.WriteString("Agent " + ev.BaselineVersion + " → " + ev.CandidateVersion + " · eval " + ev.EvaluationRunID + " · seed " + ev.RunSeed + "\n")
	return b.String()
}

// Publisher publishes an SCM status. Implementations: FakePublisher (tests/demo)
// and GitHubStatusPublisher (live).
type Publisher interface {
	Publish(ctx context.Context, s SCMStatus) error
}

// FakePublisher records published statuses without any I/O.
type FakePublisher struct {
	mu        sync.Mutex
	Published []SCMStatus
}

// Publish records the status.
func (f *FakePublisher) Publish(_ context.Context, s SCMStatus) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Published = append(f.Published, s)
	return nil
}

// IdempotentPublisher wraps a Publisher with an in-memory delivery log so a
// repeated identical status (same repo+commit+context+state) is delivered only
// once — retries and duplicate deliveries do not re-post (work order §3.5, §5
// Phase 4). A changed state re-publishes.
type IdempotentPublisher struct {
	inner Publisher
	mu    sync.Mutex
	seen  map[string]string // key -> last state
}

// NewIdempotentPublisher wraps inner with a fresh delivery log.
func NewIdempotentPublisher(inner Publisher) *IdempotentPublisher {
	return &IdempotentPublisher{inner: inner, seen: make(map[string]string)}
}

// Publish delivers s unless an identical status was already delivered; it
// returns (delivered, error).
func (p *IdempotentPublisher) Publish(ctx context.Context, s SCMStatus) (bool, error) {
	key := s.Repository + "@" + s.CommitSHA + "#" + s.Context
	p.mu.Lock()
	if prev, ok := p.seen[key]; ok && prev == s.State {
		p.mu.Unlock()
		return false, nil // already delivered this state; idempotent no-op
	}
	p.mu.Unlock()

	if err := p.inner.Publish(ctx, s); err != nil {
		return false, err
	}
	p.mu.Lock()
	p.seen[key] = s.State
	p.mu.Unlock()
	return true, nil
}

// validate rejects an unpublishable status early (missing coordinates).
func (s SCMStatus) validate() error {
	const op = Op + ".SCMStatus.validate"
	if strings.TrimSpace(s.Repository) == "" || strings.TrimSpace(s.CommitSHA) == "" {
		return errs.Validation(op, "repository and commit SHA are required to publish", nil)
	}
	if _, _, ok := strings.Cut(s.Repository, "/"); !ok {
		return errs.Validation(op, "repository must be owner/repo, got "+s.Repository, nil)
	}
	return nil
}

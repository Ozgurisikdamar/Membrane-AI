// Package domain holds the pure reporter domain: turning a consumed verdict
// into a human-facing report. No I/O (ENGINEERING-STANDARDS §1).
package domain

import (
	"fmt"
	"strings"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
)

// Outcome is the report-level state derived from a verdict decision.
type Outcome string

// Outcomes map 1:1 onto CI/commit-status semantics.
const (
	OutcomeSuccess Outcome = "success" // approved
	OutcomeFailure Outcome = "failure" // rejected
	OutcomeNeutral Outcome = "neutral" // needs_review — a human must look
)

// Mode is the enforcement posture (the shadow-mode → enforcement rollout path):
// in shadow mode a rejection is surfaced but does NOT block the merge.
type Mode string

const (
	// ModeEnforce blocks merges on rejected verdicts (rejected → failure).
	ModeEnforce Mode = "enforce"
	// ModeShadow reports rejections as non-blocking (rejected → neutral), so a
	// team can observe the gate's verdicts before turning it into a hard gate.
	ModeShadow Mode = "shadow"
)

// Finding mirrors one verdict finding for rendering.
type Finding struct {
	Stage    string
	Rule     string
	Severity string
	Message  string
}

// Verdict is the consumed shape the reporter works from (decoded off the bus).
type Verdict struct {
	SubmissionID   string
	OrganizationID string
	Decision       string
	Source         string
	RulesetVersion string
	Findings       []Finding
	// Repository ("owner/repo"), CommitSHA and PRNumber are optional source
	// coordinates; commit-status/PR notifiers need them.
	Repository string
	CommitSHA  string
	PRNumber   int
}

// Report is the rendered notification.
type Report struct {
	SubmissionID   string
	OrganizationID string
	Outcome        Outcome
	Title          string
	Body           string
	Repository     string
	CommitSHA      string
	PRNumber       int
}

const op = "reporter.domain.NewReport"

// NewReport validates and renders a verdict into a notification-ready report.
// In shadow mode a rejection is downgraded to a non-blocking neutral outcome so
// it never fails the merge gate (the findings are still surfaced).
func NewReport(v Verdict, mode Mode) (Report, error) {
	if strings.TrimSpace(v.SubmissionID) == "" {
		return Report{}, errs.Validation(op, "submission_id is required", nil)
	}
	outcome, err := outcomeFor(v.Decision)
	if err != nil {
		return Report{}, err
	}
	t := title(outcome, v)
	if mode == ModeShadow && outcome == OutcomeFailure {
		outcome = OutcomeNeutral // non-blocking: observe before enforcing
		t = fmt.Sprintf("MEMBRANE.AI ⚠ shadow mode — %d finding(s) would block in enforce mode", len(v.Findings))
	}
	return Report{
		SubmissionID:   v.SubmissionID,
		OrganizationID: v.OrganizationID,
		Outcome:        outcome,
		Title:          t,
		Body:           body(v),
		Repository:     v.Repository,
		CommitSHA:      v.CommitSHA,
		PRNumber:       v.PRNumber,
	}, nil
}

func outcomeFor(decision string) (Outcome, error) {
	switch decision {
	case "approved":
		return OutcomeSuccess, nil
	case "rejected":
		return OutcomeFailure, nil
	case "needs_review":
		return OutcomeNeutral, nil
	default:
		return "", errs.Validation(op, "unknown decision "+decision, nil)
	}
}

func title(outcome Outcome, v Verdict) string {
	switch outcome {
	case OutcomeSuccess:
		return "MEMBRANE.AI ✓ approved — architectural & security checks passed"
	case OutcomeFailure:
		return fmt.Sprintf("MEMBRANE.AI ✗ rejected — %d finding(s) must be fixed", len(v.Findings))
	default:
		return "MEMBRANE.AI ⚠ needs review — human sign-off required"
	}
}

// maxBodyFindings caps the rendered list so chat messages stay readable.
const maxBodyFindings = 10

func body(v Verdict) string {
	var b strings.Builder
	fmt.Fprintf(&b, "submission %s · source %s · ruleset %s\n", v.SubmissionID, v.Source, v.RulesetVersion)
	n := len(v.Findings)
	shown := v.Findings
	if n > maxBodyFindings {
		shown = v.Findings[:maxBodyFindings]
	}
	for _, f := range shown {
		fmt.Fprintf(&b, "• [%s] %s (%s): %s\n", f.Stage, f.Rule, f.Severity, f.Message)
	}
	if n > maxBodyFindings {
		fmt.Fprintf(&b, "… and %d more finding(s)\n", n-maxBodyFindings)
	}
	if n == 0 {
		b.WriteString("no findings\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

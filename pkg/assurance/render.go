package assurance

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
)

// RenderJSON serializes a release evidence record to canonical, stable JSON
// (two-space indent, Go struct field order). It is deterministic: the same
// record always renders byte-identically, so a persisted record regenerates its
// report exactly (work order §3: "report regeneration is deterministic from
// persisted evidence").
func RenderJSON(ev ReleaseEvidenceV1) ([]byte, error) {
	b, err := json.MarshalIndent(ev, "", "  ")
	if err != nil {
		return nil, errs.Internal(Op+".RenderJSON", "marshal evidence", err)
	}
	return b, nil
}

// RenderMarkdown renders a human-auditable report from the evidence alone. Like
// RenderJSON it depends only on the record, so regeneration is deterministic.
func RenderMarkdown(ev ReleaseEvidenceV1) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Release evidence — %s %s → %s\n\n", ev.AgentID, ev.BaselineVersion, ev.CandidateVersion)
	fmt.Fprintf(&b, "- **Gate decision:** %s\n", ev.GateDecision)
	fmt.Fprintf(&b, "- Triggered rules: %s\n", joinOr(ev.TriggeredRules, "none"))
	fmt.Fprintf(&b, "- Code verdict: %s (%d blocking)\n", ev.CodeVerdict, ev.FindingsSummary["blocking"])
	fmt.Fprintf(&b, "- Agent eval status: %s\n", ev.OverallStatus)
	fmt.Fprintf(&b, "- Evaluation run: %s · seed %s\n", ev.EvaluationRunID, ev.RunSeed)
	fmt.Fprintf(&b, "- Repository: %s @ %s\n", ev.Repository, ev.CommitSHA)
	fmt.Fprintf(&b, "- Release: %s · correlation %s\n", ev.ReleaseID, ev.CorrelationID)
	fmt.Fprintf(&b, "- Snapshot: %s\n\n", ev.SnapshotHash)

	if len(ev.CriticalFailures) > 0 {
		b.WriteString("## Critical failures\n\n")
		for _, c := range ev.CriticalFailures {
			fmt.Fprintf(&b, "- **%s** (%s)\n", c.Scenario, c.Severity)
		}
		b.WriteString("\n")
	}
	if len(ev.PerScenario) > 0 {
		b.WriteString("## Scenarios\n\n| Scenario | Severity | Classification |\n| --- | --- | --- |\n")
		for _, s := range ev.PerScenario {
			fmt.Fprintf(&b, "| %s | %s | %s |\n", s.Scenario, s.Severity, s.Status)
		}
	}
	return b.String()
}

func joinOr(xs []string, empty string) string {
	if len(xs) == 0 {
		return empty
	}
	return strings.Join(xs, ", ")
}

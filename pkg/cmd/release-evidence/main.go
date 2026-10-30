// Command release-evidence produces a combined Membrane release-evidence record
// from (1) a Membrane code-governance scan result and (2) an AgentTwin paired
// evaluation, applying the deterministic fail-safe gate (pkg/assurance). It is
// the Phase-2 "one documented action" that links a code change to agent
// behavior into one auditable, reproducible record (JSON + Markdown).
//
// Usage:
//
//	membrane scan <path> --json > scan.json
//	release-evidence --scan-json scan.json \
//	  --repo owner/repo --commit <sha> --agent support-refund-agent \
//	  --agenttwin-url http://localhost:8080 --eval-run-id <id> --allow-loopback
//	# or, offline/demo with the deterministic fake (no backend, no credentials):
//	release-evidence --scan-json scan.json --fake --repo owner/repo --commit <sha>
//
// Credentials come from --agenttwin-key or $AGENTTWIN_API_KEY (never logged).
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/assurance"
)

// scanResult mirrors the `membrane scan --json` payload (clients/cli runner.Result).
type scanResult struct {
	FilesScanned int `json:"files_scanned"`
	Blocking     int `json:"blocking"`
	Warnings     int `json:"warnings"`
	Infos        int `json:"infos"`
	Findings     []struct {
		Rule     string `json:"rule"`
		Severity string `json:"severity"`
	} `json:"findings"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "release-evidence: "+err.Error())
		os.Exit(1)
	}
}

func run() error {
	var (
		scanJSON     = flag.String("scan-json", "", "path to `membrane scan --json` output (required)")
		repo         = flag.String("repo", "", "repository, e.g. owner/repo (required)")
		commit       = flag.String("commit", "", "commit SHA (required)")
		agent        = flag.String("agent", "support-refund-agent", "agent id")
		org          = flag.String("org", "membrane-demo", "Membrane organization id")
		project      = flag.String("project", "demo", "Membrane project id")
		releaseID    = flag.String("release-id", "", "release id (default: rel-<commit>)")
		corrID       = flag.String("correlation-id", "", "correlation id (default: corr-<commit>)")
		atURL        = flag.String("agenttwin-url", os.Getenv("AGENTTWIN_BASE_URL"), "AgentTwin base URL")
		atKey        = flag.String("agenttwin-key", os.Getenv("AGENTTWIN_API_KEY"), "AgentTwin API key (or $AGENTTWIN_API_KEY)")
		evalRunID    = flag.String("eval-run-id", "", "AgentTwin evaluation run id (live mode)")
		allowLoop    = flag.Bool("allow-loopback", false, "allow a loopback AgentTwin URL (local/dev)")
		allowedHosts = flag.String("allowed-hosts", "", "comma-separated AgentTwin host allowlist")
		useFake      = flag.Bool("fake", false, "use the deterministic fake (no backend); result is labelled deterministic-fake")
		out          = flag.String("out", "", "output file prefix; writes <prefix>.json and <prefix>.md (default: JSON to stdout)")
	)
	flag.Parse()

	if *scanJSON == "" || *repo == "" || *commit == "" {
		return fmt.Errorf("--scan-json, --repo and --commit are required")
	}

	scan, err := readScan(*scanJSON)
	if err != nil {
		return err
	}
	codeVerdict, findings := codeVerdictOf(scan)

	rid := orDefault(*releaseID, "rel-"+*commit)
	cid := orDefault(*corrID, "corr-"+*commit)
	corr := assurance.CorrelationContext{
		SchemaVersion:    assurance.CorrelationSchemaVersion,
		OrganizationID:   *org,
		ProjectID:        *project,
		CodeSubmissionID: "sub-" + *commit,
		Repository:       *repo,
		CommitSHA:        *commit,
		ReleaseID:        rid,
		AgentID:          *agent,
		AgentVersion:     "candidate",
		CorrelationID:    cid,
	}

	ref, label, err := fetchEval(*useFake, *atURL, *atKey, *evalRunID, *allowLoop, *allowedHosts)
	if err != nil {
		return err
	}

	ev, err := assurance.BuildReleaseEvidence(assurance.BuildInput{
		Correlation:  corr,
		CodeVerdict:  codeVerdict,
		CodeFindings: findings,
		CodeRuleset:  "scan-v1",
		Agent:        ref,
		Now:          time.Now().UTC(),
		SnapshotHash: "sha256:" + *commit,
	})
	if err != nil {
		return err
	}

	jsonBytes, err := json.MarshalIndent(ev, "", "  ")
	if err != nil {
		return err
	}
	md := markdown(ev, label)

	if *out == "" {
		fmt.Println(string(jsonBytes))
		fmt.Fprintln(os.Stderr, "gate="+string(ev.GateDecision)+" ("+label+")")
		return nil
	}
	if err := os.WriteFile(*out+".json", jsonBytes, 0o600); err != nil { //nolint:gosec // G304: path is an operator-supplied report file
		return err
	}
	if err := os.WriteFile(*out+".md", []byte(md), 0o600); err != nil { //nolint:gosec // G304: path is an operator-supplied report file
		return err
	}
	fmt.Fprintln(os.Stderr, "wrote "+*out+".json and "+*out+".md · gate="+string(ev.GateDecision)+" ("+label+")")
	return nil
}

func readScan(path string) (scanResult, error) {
	b, err := os.ReadFile(path) //nolint:gosec // path is an operator-supplied report file
	if err != nil {
		return scanResult{}, err
	}
	var s scanResult
	if err := json.Unmarshal(b, &s); err != nil {
		return scanResult{}, fmt.Errorf("parse scan json: %w", err)
	}
	return s, nil
}

func codeVerdictOf(s scanResult) (assurance.GateDecision, map[string]int) {
	summary := map[string]int{"blocking": s.Blocking, "warning": s.Warnings, "info": s.Infos}
	switch {
	case s.Blocking > 0:
		return assurance.DecisionBlock, summary
	case s.Warnings > 0:
		return assurance.DecisionWarn, summary
	default:
		return assurance.DecisionPass, summary
	}
}

func fetchEval(useFake bool, url, key, evalRunID string, allowLoop bool, hosts string) (assurance.EvalResultRef, string, error) {
	if useFake || url == "" {
		return fakeRef(), "deterministic-fake (no backend)", nil
	}
	if evalRunID == "" {
		return assurance.EvalResultRef{}, "", fmt.Errorf("--eval-run-id is required in live mode (omit --agenttwin-url or pass --fake for the deterministic fake)")
	}
	hostList := splitNonEmpty(hosts)
	client, err := assurance.NewHTTPClient(assurance.HTTPConfig{
		BaseURL:   url,
		APIKey:    key,
		Allowlist: assurance.NewAllowlist(allowLoop, hostList...),
	})
	if err != nil {
		return assurance.EvalResultRef{}, "", err
	}
	ref, err := client.GetEvalResult(context.Background(), evalRunID)
	if err != nil {
		return assurance.EvalResultRef{}, "", err
	}
	return ref, "live AgentTwin", nil
}

// fakeRef is the deterministic 1.3.0 result (real reproduced counts), labelled.
func fakeRef() assurance.EvalResultRef {
	return assurance.EvalResultRef{
		EvaluationRunID:   "fake-1.3.0",
		AgentTwinProject:  "demo",
		BaselineVersion:   "1.2.4",
		CandidateVersion:  "1.3.0",
		Seed:              "42",
		RawStatus:         "COMPLETED",
		Counts:            assurance.EvalResult{NewCriticalFailures: 2, Regressed: 1},
		ScenarioSelection: []string{"refund-tool-success-lie", "refund-timeout-after-mutation"},
		PerScenario: []assurance.ScenarioOutcome{
			{Scenario: "refund-tool-success-lie", Status: "NEW_CRITICAL_FAILURE", Severity: "critical"},
			{Scenario: "refund-timeout-after-mutation", Status: "NEW_CRITICAL_FAILURE", Severity: "critical"},
		},
	}
}

func markdown(ev assurance.ReleaseEvidenceV1, label string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Release evidence — %s %s → %s\n\n", ev.AgentID, ev.BaselineVersion, ev.CandidateVersion)
	fmt.Fprintf(&b, "- **Gate decision:** %s\n", ev.GateDecision)
	fmt.Fprintf(&b, "- Triggered rules: %s\n", strings.Join(ev.TriggeredRules, ", "))
	fmt.Fprintf(&b, "- Code verdict: %s (%d blocking)\n", ev.CodeVerdict, ev.FindingsSummary["blocking"])
	fmt.Fprintf(&b, "- Agent eval status: %s · run %s · seed %s\n", ev.OverallStatus, ev.EvaluationRunID, ev.RunSeed)
	fmt.Fprintf(&b, "- Repository: %s @ %s\n", ev.Repository, ev.CommitSHA)
	fmt.Fprintf(&b, "- Evidence source: %s\n\n", label)
	if len(ev.PerScenario) > 0 {
		b.WriteString("## Scenarios\n\n| Scenario | Severity | Classification |\n| --- | --- | --- |\n")
		for _, s := range ev.PerScenario {
			fmt.Fprintf(&b, "| %s | %s | %s |\n", s.Scenario, s.Severity, s.Status)
		}
	}
	return b.String()
}

func orDefault(v, d string) string {
	if strings.TrimSpace(v) == "" {
		return d
	}
	return v
}

func splitNonEmpty(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

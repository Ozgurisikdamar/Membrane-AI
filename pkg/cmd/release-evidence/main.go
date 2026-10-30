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
		store        = flag.String("store", "", "persist the immutable evidence record to this directory")
		regenerate   = flag.String("regenerate", "", "read a stored record by release id and re-emit it (requires --store); skips scan/eval")
		publishSCM   = flag.Bool("publish-shadow", false, "publish the combined gate as an SCM status (shadow mode = non-blocking)")
		enforce      = flag.Bool("enforce", false, "with --publish-shadow: enforce mode (BLOCK → failure) instead of shadow")
		ghToken      = flag.String("github-token", os.Getenv("GITHUB_TOKEN"), "GitHub token for commit-status publish (or $GITHUB_TOKEN); empty = fake publisher")
		ghBase       = flag.String("github-base-url", "https://api.github.com", "GitHub API base URL (for GHE/tests)")
	)
	flag.Parse()

	// Deterministic regeneration from the persisted record (no scan/eval needed).
	if *regenerate != "" {
		if *store == "" {
			return fmt.Errorf("--regenerate requires --store")
		}
		st, err := assurance.NewFileStore(*store)
		if err != nil {
			return err
		}
		ev, err := st.Get(*regenerate)
		if err != nil {
			return err
		}
		return emit(ev, "regenerated from store", *out)
	}

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

	if *store != "" {
		st, err := assurance.NewFileStore(*store)
		if err != nil {
			return err
		}
		if err := st.Put(ev); err != nil {
			return err // immutability conflict or validation error, surfaced honestly
		}
		fmt.Fprintln(os.Stderr, "persisted evidence for release "+ev.ReleaseID+" to "+*store)
	}

	if *publishSCM {
		if err := publishShadow(ev, *enforce, *ghToken, *ghBase); err != nil {
			return err
		}
	}
	return emit(ev, label, *out)
}

// publishShadow maps the evidence to an SCM status and publishes it idempotently.
// Shadow mode (default) is non-blocking; enforcement is opt-in. With no token a
// deterministic fake publisher prints the status instead of calling GitHub.
func publishShadow(ev assurance.ReleaseEvidenceV1, enforce bool, token, base string) error {
	mode := assurance.ModeShadow
	if enforce {
		mode = assurance.ModeEnforce
	}
	st := assurance.MapSCMStatus(ev, mode)
	var pub assurance.Publisher
	target := "fake"
	if strings.TrimSpace(token) != "" {
		p, err := assurance.NewGitHubStatusPublisher(base, token)
		if err != nil {
			return err
		}
		pub, target = p, "github"
	} else {
		pub = &assurance.FakePublisher{}
	}
	delivered, err := assurance.NewIdempotentPublisher(pub).Publish(context.Background(), st)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "scm publish (%s, %s): outcome=%s state=%s delivered=%v · %s\n",
		mode, target, st.Outcome, st.State, delivered, st.Title)
	return nil
}

// emit renders the canonical JSON/Markdown (deterministic from the record) to
// stdout or a file prefix.
func emit(ev assurance.ReleaseEvidenceV1, label, out string) error {
	jsonBytes, err := assurance.RenderJSON(ev)
	if err != nil {
		return err
	}
	md := assurance.RenderMarkdown(ev)
	if out == "" {
		fmt.Println(string(jsonBytes))
		fmt.Fprintln(os.Stderr, "gate="+string(ev.GateDecision)+" ("+label+")")
		return nil
	}
	if err := os.WriteFile(out+".json", jsonBytes, 0o600); err != nil { //nolint:gosec // G304: operator-supplied report prefix
		return err
	}
	if err := os.WriteFile(out+".md", []byte(md), 0o600); err != nil { //nolint:gosec // G304: operator-supplied report prefix
		return err
	}
	fmt.Fprintln(os.Stderr, "wrote "+out+".json and "+out+".md · gate="+string(ev.GateDecision)+" ("+label+")")
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

package assurance

import (
	"time"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
)

// EvalResultRef is the minimal, already-validated result an AgentTwin paired
// evaluation yields, as returned by the AgentTwin client port. It carries only
// references and counts — no raw spans/prompts/PII (work order §3.3).
type EvalResultRef struct {
	EvaluationRunID   string
	AgentTwinProject  string
	BaselineVersion   string
	CandidateVersion  string
	Seed              string
	RawStatus         string
	Counts            EvalResult // reuses the status-mapper input (NewCritical/Regressed/...)
	ScenarioSelection []string
	PerScenario       []ScenarioOutcome
	SimulationPairIDs []string
	EngineVersion     string
	EvaluatorVersion  string
}

// BuildInput is everything the aggregator needs to assemble a release evidence
// record: the Membrane side (correlation + code verdict), the AgentTwin result,
// and the policy knobs.
type BuildInput struct {
	Correlation     CorrelationContext
	CodeVerdict     GateDecision
	CodeFindings    map[string]int
	CodeRuleset     string
	CodeEvidenceURL string
	Agent           EvalResultRef
	Policy          Policy
	Now             time.Time
	SnapshotHash    string
}

// BuildReleaseEvidence is the deterministic combined-gate aggregator: it maps
// the AgentTwin result to an integration status, computes the fail-safe gate
// decision over the code verdict + agent status, assembles a ReleaseEvidenceV1,
// and validates it. The result is reproducible from the same input (work order
// §3.6, §4). A critical failure, ERROR or INCOMPLETE can never become PASS.
func BuildReleaseEvidence(in BuildInput) (ReleaseEvidenceV1, error) {
	const op = Op + ".BuildReleaseEvidence"
	if err := in.Correlation.Validate(); err != nil {
		return ReleaseEvidenceV1{}, errs.Validation(op, "invalid correlation", err)
	}
	if err := in.CodeVerdict.Validate(); err != nil {
		return ReleaseEvidenceV1{}, errs.Validation(op, "invalid code verdict", err)
	}

	status := MapEvalStatus(in.Agent.Counts.withStatus(in.Agent.RawStatus))
	gate := combinedGate(status, in.CodeVerdict, in.Policy)

	now := in.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}

	ev := ReleaseEvidenceV1{
		SchemaVersion:      EvidenceSchemaVersion,
		ReleaseID:          in.Correlation.ReleaseID,
		CorrelationID:      in.Correlation.CorrelationID,
		CreatedAt:          now,
		OrganizationID:     in.Correlation.OrganizationID,
		ProjectID:          in.Correlation.ProjectID,
		Repository:         in.Correlation.Repository,
		CodeSubmissionID:   in.Correlation.CodeSubmissionID,
		CommitSHA:          in.Correlation.CommitSHA,
		PullRequestNumber:  in.Correlation.PullRequestNumber,
		CodeVerdict:        in.CodeVerdict,
		RulesetVersion:     in.CodeRuleset,
		FindingsSummary:    in.CodeFindings,
		CodeEvidenceURL:    in.CodeEvidenceURL,
		AgentID:            in.Correlation.AgentID,
		BaselineVersion:    in.Agent.BaselineVersion,
		CandidateVersion:   in.Agent.CandidateVersion,
		AgentTwinProjectID: in.Agent.AgentTwinProject,
		EvaluationRunID:    in.Agent.EvaluationRunID,
		SimulationPairIDs:  in.Agent.SimulationPairIDs,
		ScenarioSelection:  in.Agent.ScenarioSelection,
		RunSeed:            in.Agent.Seed,
		EngineVersion:      in.Agent.EngineVersion,
		EvaluatorVersion:   in.Agent.EvaluatorVersion,
		OverallStatus:      status,
		PerScenario:        in.Agent.PerScenario,
		CriticalFailures:   criticalOnly(in.Agent.PerScenario),
		GateDecision:       gate.decision,
		TriggeredRules:     gate.rules,
		SnapshotHash:       in.SnapshotHash,
	}
	if err := ev.Validate(); err != nil {
		return ReleaseEvidenceV1{}, errs.Validation(op, "assembled evidence failed validation", err)
	}
	return ev, nil
}

type gateResult struct {
	decision GateDecision
	rules    []string
}

// combinedGate layers the code-governance verdict onto the agent-assurance
// decision. It is fail-safe and explainable: a hard BLOCK on either side blocks;
// a non-PASS agent status never becomes PASS; every BLOCK/WARN cites a rule.
func combinedGate(status IntegrationStatus, code GateDecision, p Policy) gateResult {
	rules := make([]string, 0, 2)
	if code == DecisionBlock {
		rules = append(rules, "CODE_CRITICAL")
	}
	agent := Decide(status, p)
	switch status {
	case StatusCompletedWithCriticalFailure:
		rules = append(rules, "AGENT_CRITICAL_SCENARIO")
	case StatusCompletedWithRegression:
		rules = append(rules, "NONCRITICAL_REGRESSION")
	case StatusError, StatusIncomplete, StatusPending, StatusBlocked:
		rules = append(rules, "AGENT_EVAL_INCOMPLETE")
	case StatusNotConfigured:
		rules = append(rules, "OPTIONAL_ASSURANCE_UNAVAILABLE")
	case StatusCompleted:
	}

	// Decision precedence: BLOCK > ERROR > INCOMPLETE > WARN > PASS.
	if code == DecisionBlock || agent == DecisionBlock {
		return gateResult{DecisionBlock, rules}
	}
	if agent == DecisionError {
		return gateResult{DecisionError, rules}
	}
	if agent == DecisionIncomplete {
		return gateResult{DecisionIncomplete, rules}
	}
	if agent == DecisionWarn || code == DecisionWarn {
		if code == DecisionWarn {
			rules = append(rules, "COVERAGE_GAP")
		}
		return gateResult{DecisionWarn, rules}
	}
	return gateResult{DecisionPass, rules}
}

func criticalOnly(s []ScenarioOutcome) []ScenarioOutcome {
	var out []ScenarioOutcome
	for _, o := range s {
		if o.Severity == "critical" && o.Status == "NEW_CRITICAL_FAILURE" {
			out = append(out, o)
		}
	}
	return out
}

// withStatus returns a copy of the counts as the EvalResult the status mapper
// expects, stamped with the raw upstream status.
func (c EvalResult) withStatus(raw string) EvalResult {
	c.RawStatus = raw
	return c
}

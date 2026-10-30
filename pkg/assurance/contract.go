package assurance

import (
	"strings"
	"time"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
)

// CorrelationContext is the versioned cross-product correlation contract carried
// from Membrane into AgentTwin on every assurance call (work order §3.4). It
// identifies the release and change, not the tenant: tenant identity derives
// from the authenticated mapping, so none of these fields may be used as an
// authorization decision. A commit SHA or agent version appearing here is
// metadata only.
type CorrelationContext struct {
	SchemaVersion    string `json:"schema_version"`
	OrganizationID   string `json:"membrane_organization_id"`
	ProjectID        string `json:"membrane_project_id"`
	CodeSubmissionID string `json:"membrane_code_submission_id"`
	Repository       string `json:"membrane_repository"`
	CommitSHA        string `json:"membrane_commit_sha"`
	// PullRequestNumber is optional (0 means "not from a pull request").
	PullRequestNumber int    `json:"membrane_pull_request_number,omitempty"`
	ReleaseID         string `json:"membrane_release_id"`
	AgentID           string `json:"membrane_agent_id"`
	AgentVersion      string `json:"membrane_agent_version"`
	ChangeSetID       string `json:"membrane_change_set_id,omitempty"`
	CorrelationID     string `json:"membrane_correlation_id"`
	// Traceparent is the W3C trace-context header value, propagated where the
	// transport supports it. It is distinct from the business release ID.
	Traceparent string `json:"traceparent,omitempty"`
	Tracestate  string `json:"tracestate,omitempty"`
}

// Validate checks the required correlation fields and the version. It does not
// require the optional PR number, change set, or trace headers.
func (c CorrelationContext) Validate() error {
	const op = Op + ".CorrelationContext.Validate"
	if c.SchemaVersion != CorrelationSchemaVersion {
		return errs.Validation(op, "unexpected correlation schema_version: "+c.SchemaVersion, nil)
	}
	for field, v := range map[string]string{
		"membrane_organization_id":    c.OrganizationID,
		"membrane_project_id":         c.ProjectID,
		"membrane_code_submission_id": c.CodeSubmissionID,
		"membrane_repository":         c.Repository,
		"membrane_commit_sha":         c.CommitSHA,
		"membrane_release_id":         c.ReleaseID,
		"membrane_agent_id":           c.AgentID,
		"membrane_agent_version":      c.AgentVersion,
		"membrane_correlation_id":     c.CorrelationID,
	} {
		if strings.TrimSpace(v) == "" {
			return errs.Validation(op, "missing required correlation field: "+field, nil)
		}
	}
	if !looksLikeCommitSHA(c.CommitSHA) {
		return errs.Validation(op, "membrane_commit_sha is not a hex commit SHA", nil)
	}
	return nil
}

// MappingRecordV1 is the minimal tenant/project mapping between a Membrane
// project and an AgentTwin project (work order §3.3, IMPLEMENTATION_PLAN §4.1).
// The AgentTwin base URL is server-side allowlisted configuration and is never a
// user-controlled arbitrary URL; the credential is referenced by secret-manager
// key name only and is never stored in plaintext here.
type MappingRecordV1 struct {
	SchemaVersion      string    `json:"schema_version"`
	MappingID          string    `json:"mapping_id"`
	MembraneOrgID      string    `json:"membrane_org_id"`
	MembraneProjectID  string    `json:"membrane_project_id"`
	AgentTwinOrgID     string    `json:"agenttwin_org_id"`
	AgentTwinProjectID string    `json:"agenttwin_project_id"`
	AgentTwinBaseURL   string    `json:"agenttwin_base_url"`
	CredentialRef      string    `json:"credential_ref"`
	Status             string    `json:"status"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
	Actor              string    `json:"actor"`
	AuditRef           string    `json:"audit_ref"`
	// DefaultAgentID is the optional agent used when a submission does not name
	// one (e.g. the demo agent).
	DefaultAgentID string `json:"default_agent_id,omitempty"`
}

// Validate enforces the required mapping fields, the version, and the two
// security invariants: the base URL must pass the destination allowlist, and the
// credential reference must look like a key name, not an inline secret.
func (m MappingRecordV1) Validate(allow *Allowlist) error {
	const op = Op + ".MappingRecordV1.Validate"
	if m.SchemaVersion != MappingSchemaVersion {
		return errs.Validation(op, "unexpected mapping schema_version: "+m.SchemaVersion, nil)
	}
	for field, v := range map[string]string{
		"mapping_id":           m.MappingID,
		"membrane_org_id":      m.MembraneOrgID,
		"membrane_project_id":  m.MembraneProjectID,
		"agenttwin_org_id":     m.AgentTwinOrgID,
		"agenttwin_project_id": m.AgentTwinProjectID,
		"agenttwin_base_url":   m.AgentTwinBaseURL,
		"credential_ref":       m.CredentialRef,
		"status":               m.Status,
		"actor":                m.Actor,
		"audit_ref":            m.AuditRef,
	} {
		if strings.TrimSpace(v) == "" {
			return errs.Validation(op, "missing required mapping field: "+field, nil)
		}
	}
	if allow == nil {
		return errs.Validation(op, "a destination allowlist is required to validate a mapping", nil)
	}
	if err := allow.CheckURL(m.AgentTwinBaseURL); err != nil {
		return errs.Validation(op, "agenttwin_base_url rejected by allowlist", err)
	}
	if looksLikeInlineSecret(m.CredentialRef) {
		return errs.Validation(op, "credential_ref must be a secret-manager key name, not an inline secret", nil)
	}
	return nil
}

// ScenarioOutcome is a per-scenario line in the evidence record: enough to audit
// which scenario produced which severity, with no raw trajectory content.
type ScenarioOutcome struct {
	Scenario string `json:"scenario"`
	Status   string `json:"status"`
	Severity string `json:"severity"`
	// EvidenceURLs point at canonical AgentTwin evidence (trajectory, state diff,
	// verified outcome, first divergence). Keys are stable names; values are URLs.
	EvidenceURLs map[string]string `json:"evidence_urls,omitempty"`
}

// JudgeMetadata records semantic-judge provenance when a semantic judge was
// involved, so a reviewer can tell calibrated from advisory findings (§4.3).
type JudgeMetadata struct {
	Provider          string `json:"provider,omitempty"`
	Model             string `json:"model,omitempty"`
	PromptHash        string `json:"prompt_hash,omitempty"`
	CalibrationStatus string `json:"calibration_status,omitempty"`
}

// ReleaseEvidenceV1 is the minimal, immutable record Membrane persists to make a
// combined release decision auditable without trusting an agent's final answer
// (work order §3.3, §4.2, IMPLEMENTATION_PLAN §4.2). It stores references and
// summaries only — never raw spans, prompts, tool payloads or PII.
type ReleaseEvidenceV1 struct {
	SchemaVersion string    `json:"schema_version"`
	ReleaseID     string    `json:"release_id"`
	CorrelationID string    `json:"correlation_id"`
	CreatedAt     time.Time `json:"created_at"`

	// Membrane side.
	OrganizationID    string `json:"membrane_organization_id"`
	ProjectID         string `json:"membrane_project_id"`
	Repository        string `json:"membrane_repository"`
	CodeSubmissionID  string `json:"membrane_code_submission_id"`
	CommitSHA         string `json:"membrane_commit_sha"`
	PullRequestNumber int    `json:"membrane_pull_request_number,omitempty"`

	// Code governance verdict (summary + pointer, not the raw findings).
	CodeVerdict     GateDecision   `json:"code_verdict"`
	RulesetVersion  string         `json:"ruleset_version"`
	FindingsSummary map[string]int `json:"findings_summary"`
	CodeEvidenceURL string         `json:"code_evidence_url,omitempty"`

	// Agent identity.
	AgentID          string `json:"agent_id"`
	BaselineVersion  string `json:"baseline_version"`
	CandidateVersion string `json:"candidate_version"`
	ManifestHash     string `json:"candidate_manifest_hash,omitempty"`
	PromptHash       string `json:"candidate_prompt_hash,omitempty"`

	// AgentTwin side (external references, pinned metadata).
	AgentTwinProjectID string   `json:"agenttwin_project_id"`
	EvaluationRunID    string   `json:"agenttwin_evaluation_run_id"`
	SimulationPairIDs  []string `json:"agenttwin_simulation_pair_ids,omitempty"`
	ScenarioSelection  []string `json:"scenario_selection"`
	RunSeed            string   `json:"run_seed"`
	EngineVersion      string   `json:"engine_version,omitempty"`
	EvaluatorVersion   string   `json:"evaluator_version,omitempty"`

	// Outcome.
	OverallStatus    IntegrationStatus `json:"overall_status"`
	PerScenario      []ScenarioOutcome `json:"per_scenario"`
	CriticalFailures []ScenarioOutcome `json:"critical_failures,omitempty"`
	Judge            *JudgeMetadata    `json:"judge,omitempty"`

	// Combined gate.
	GateDecision   GateDecision `json:"gate_decision"`
	TriggeredRules []string     `json:"triggered_rules"`

	// Audit.
	SnapshotHash string `json:"snapshot_hash"`
	AuditRef     string `json:"audit_ref,omitempty"`
}

// Validate enforces the required evidence fields, the version, and the two
// fail-safe invariants that make the record trustworthy:
//
//  1. A non-clean agent outcome (critical failure, error, incomplete, blocked,
//     pending) can never carry a PASS gate decision.
//  2. A BLOCK/WARN gate decision must cite at least one triggered rule, so no
//     decision is unexplained.
func (e ReleaseEvidenceV1) Validate() error {
	const op = Op + ".ReleaseEvidenceV1.Validate"
	if e.SchemaVersion != EvidenceSchemaVersion {
		return errs.Validation(op, "unexpected evidence schema_version: "+e.SchemaVersion, nil)
	}
	for field, v := range map[string]string{
		"release_id":                  e.ReleaseID,
		"correlation_id":              e.CorrelationID,
		"membrane_organization_id":    e.OrganizationID,
		"membrane_project_id":         e.ProjectID,
		"membrane_repository":         e.Repository,
		"membrane_code_submission_id": e.CodeSubmissionID,
		"membrane_commit_sha":         e.CommitSHA,
		"agent_id":                    e.AgentID,
		"baseline_version":            e.BaselineVersion,
		"candidate_version":           e.CandidateVersion,
		"agenttwin_project_id":        e.AgentTwinProjectID,
		"agenttwin_evaluation_run_id": e.EvaluationRunID,
		"run_seed":                    e.RunSeed,
		"snapshot_hash":               e.SnapshotHash,
	} {
		if strings.TrimSpace(v) == "" {
			return errs.Validation(op, "missing required evidence field: "+field, nil)
		}
	}
	if e.CreatedAt.IsZero() {
		return errs.Validation(op, "missing required evidence field: created_at", nil)
	}
	if !looksLikeCommitSHA(e.CommitSHA) {
		return errs.Validation(op, "membrane_commit_sha is not a hex commit SHA", nil)
	}
	if len(e.ScenarioSelection) == 0 {
		return errs.Validation(op, "scenario_selection must not be empty", nil)
	}
	if err := e.OverallStatus.Validate(); err != nil {
		return err
	}
	if err := e.GateDecision.Validate(); err != nil {
		return err
	}
	if err := e.CodeVerdict.Validate(); err != nil {
		return errs.Validation(op, "invalid code_verdict", err)
	}
	// Invariant 1: a non-clean agent outcome can never be PASS.
	if e.GateDecision == DecisionPass && e.OverallStatus != StatusCompleted {
		return errs.Validation(op, "gate_decision PASS contradicts overall_status "+string(e.OverallStatus), nil)
	}
	// Invariant 2: a blocking/warning decision must be explained.
	if (e.GateDecision == DecisionBlock || e.GateDecision == DecisionWarn) && len(e.TriggeredRules) == 0 {
		return errs.Validation(op, "gate_decision "+string(e.GateDecision)+" must cite at least one triggered rule", nil)
	}
	return nil
}

// looksLikeCommitSHA reports whether s is a 7–64 char lowercase/upper hex string.
func looksLikeCommitSHA(s string) bool {
	if len(s) < 7 || len(s) > 64 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= '0' && c <= '9':
		case c >= 'a' && c <= 'f':
		case c >= 'A' && c <= 'F':
		default:
			return false
		}
	}
	return true
}

// looksLikeInlineSecret rejects obvious plaintext secrets in a field that must
// hold only a secret-manager key name. It is a guard, not a secret scanner: a
// key name like "agenttwin/demo-key" passes; a token like "atk_demo0000_..."
// or a long opaque blob is rejected.
func looksLikeInlineSecret(ref string) bool {
	r := strings.TrimSpace(ref)
	if r == "" {
		return false // emptiness is caught by the required-field check
	}
	lower := strings.ToLower(r)
	for _, p := range []string{"atk_", "sk-", "sk_", "ghp_", "akia", "bearer ", "-----begin"} {
		if strings.HasPrefix(lower, p) {
			return true
		}
	}
	// An over-long, space-free, high-entropy-looking blob is treated as a secret.
	if len(r) > 64 && !strings.ContainsAny(r, " /.:") {
		return true
	}
	return false
}

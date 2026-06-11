// Package envelope defines the shared wire schemas (JSON) of the MEMBRANE.AI
// event topics. Producers and consumers across services import these types so
// the contract lives in exactly one place. Field names are part of the public
// contract: never rename a tag — add fields instead (events are versioned by
// topic suffix, D-006).
package envelope

import "time"

// Default topic names (overridable per deployment via service config).
const (
	TopicSubmissionV1 = "code.submission.v1"
	TopicVerdictV1    = "code.verdict.v1"
)

// SubmissionV1 is the payload of code.submission.v1. The Kafka message key is
// OrganizationID (strict per-tenant ordering).
type SubmissionV1 struct {
	SubmissionID   string    `json:"submission_id"`
	OrganizationID string    `json:"organization_id"`
	Repository     string    `json:"repository"`
	FilePath       string    `json:"file_path"`
	Language       string    `json:"language"`
	Origin         string    `json:"origin"`
	Diff           string    `json:"diff"`
	OccurredAt     time.Time `json:"occurred_at"`
	// CommitSHA/PRNumber are optional (omitted when unknown); they enable
	// commit-status and PR-comment reporting downstream.
	CommitSHA string `json:"commit_sha,omitempty"`
	PRNumber  int    `json:"pr_number,omitempty"`
}

// FindingV1 is one analysis finding inside a verdict.
type FindingV1 struct {
	Stage    string `json:"stage"`
	Rule     string `json:"rule"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

// VerdictV1 is the payload of code.verdict.v1. The Kafka message key is
// OrganizationID.
type VerdictV1 struct {
	SubmissionID   string      `json:"submission_id"`
	OrganizationID string      `json:"organization_id"`
	Decision       string      `json:"decision"`
	Source         string      `json:"source"`
	RulesetVersion string      `json:"ruleset_version"`
	Findings       []FindingV1 `json:"findings"`
	EvaluatedAt    time.Time   `json:"evaluated_at"`
	// Repository/CommitSHA/PRNumber echo the submission's optional source
	// coordinates so reporters can post commit statuses / PR comments.
	Repository string `json:"repository,omitempty"`
	CommitSHA  string `json:"commit_sha,omitempty"`
	PRNumber   int    `json:"pr_number,omitempty"`
}

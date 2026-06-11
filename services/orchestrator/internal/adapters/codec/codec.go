// Package codec maps the shared wire envelopes (pkg/envelope) to and from the
// orchestrator domain. Every adapter that touches the bus or the outbox uses
// this single mapping, so the wire contract is encoded exactly once.
package codec

import (
	"encoding/json"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/envelope"
	"github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/domain"
)

const op = "orchestrator.adapters.codec"

// DecodeSubmission parses a code.submission.v1 payload into the domain.
func DecodeSubmission(payload []byte) (domain.Submission, error) {
	var env envelope.SubmissionV1
	if err := json.Unmarshal(payload, &env); err != nil {
		return domain.Submission{}, errs.Validation(op, "malformed submission payload", err)
	}
	sub, err := domain.NewSubmission(
		env.SubmissionID, env.OrganizationID, env.Repository,
		env.FilePath, env.Language, env.Diff, env.Origin, env.OccurredAt,
	)
	if err != nil {
		return domain.Submission{}, err
	}
	return sub.WithSource(env.CommitSHA, env.PRNumber), nil
}

// EncodeVerdict renders a domain verdict as a code.verdict.v1 payload.
func EncodeVerdict(v domain.Verdict) ([]byte, error) {
	findings := make([]envelope.FindingV1, 0, len(v.Findings))
	for _, f := range v.Findings {
		findings = append(findings, envelope.FindingV1{
			Stage: f.Stage, Rule: f.Rule, Severity: string(f.Severity), Message: f.Message,
		})
	}
	payload, err := json.Marshal(envelope.VerdictV1{
		SubmissionID:   v.SubmissionID,
		OrganizationID: v.OrganizationID,
		Decision:       string(v.Decision),
		Source:         string(v.Source),
		RulesetVersion: v.RulesetVersion,
		Findings:       findings,
		EvaluatedAt:    v.EvaluatedAt,
		Repository:     v.Repository,
		CommitSHA:      v.CommitSHA,
		PRNumber:       v.PRNumber,
	})
	if err != nil {
		return nil, errs.Internal(op, "marshal verdict", err)
	}
	return payload, nil
}

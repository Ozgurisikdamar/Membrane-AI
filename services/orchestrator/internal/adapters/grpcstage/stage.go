// Package grpcstage implements the AnalysisStage port against the remote
// analyzer service (membrane.analyzer.v1). The Saga's stage deadline flows in
// through ctx; on transport errors the Saga degrades to its deterministic
// fallback, so this adapter stays deliberately thin.
package grpcstage

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
	analyzerv1 "github.com/Ozgurisikdamar/Membrane-AI/proto/gen/membrane/analyzer/v1"
	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/domain"
)

const op = "orchestrator.adapters.grpcstage"

// Stage calls the analyzer service as one step of the Saga.
type Stage struct {
	conn   *grpc.ClientConn
	client analyzerv1.AnalyzerServiceClient
}

// Option customizes the underlying gRPC client (tests inject a bufconn dialer).
type Option func(*[]grpc.DialOption)

// WithDialOption appends a raw grpc.DialOption.
func WithDialOption(o grpc.DialOption) Option {
	return func(opts *[]grpc.DialOption) { *opts = append(*opts, o) }
}

// New dials the analyzer at addr (non-blocking; the connection is established
// lazily on first use).
func New(addr string, options ...Option) (*Stage, error) {
	dialOpts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	for _, o := range options {
		o(&dialOpts)
	}
	conn, err := grpc.NewClient(addr, dialOpts...)
	if err != nil {
		return nil, errs.Unavailable(op, "dial analyzer", err)
	}
	return &Stage{conn: conn, client: analyzerv1.NewAnalyzerServiceClient(conn)}, nil
}

// Name implements ports.AnalysisStage.
func (*Stage) Name() string { return "analyzer" }

// Analyze forwards the submission to the analyzer service and maps its
// findings into the domain.
func (s *Stage) Analyze(ctx context.Context, sub domain.Submission) ([]domain.Finding, error) {
	resp, err := s.client.Analyze(ctx, &analyzerv1.AnalyzeRequest{
		SubmissionId:   sub.SubmissionID,
		OrganizationId: sub.OrganizationID,
		Repository:     sub.Repository,
		FilePath:       sub.FilePath,
		Language:       sub.Language,
		Diff:           sub.Diff,
	})
	if err != nil {
		return nil, errs.Unavailable(op, "analyzer call failed", err)
	}
	findings := make([]domain.Finding, 0, len(resp.GetFindings()))
	for _, f := range resp.GetFindings() {
		findings = append(findings, domain.Finding{
			Stage:    s.Name(),
			Rule:     f.GetRule(),
			Severity: severityFromProto(f.GetSeverity()),
			Message:  f.GetMessage(),
		})
	}
	return findings, nil
}

func severityFromProto(s analyzerv1.Severity) domain.Severity {
	switch s {
	case analyzerv1.Severity_SEVERITY_BLOCKING:
		return domain.SeverityBlocking
	case analyzerv1.Severity_SEVERITY_WARNING:
		return domain.SeverityWarning
	case analyzerv1.Severity_SEVERITY_INFO:
		return domain.SeverityInfo
	default:
		// Unknown severities fail safe as warnings: surfaced, never silently dropped.
		return domain.SeverityWarning
	}
}

// Close releases the client connection.
func (s *Stage) Close() error { return s.conn.Close() }

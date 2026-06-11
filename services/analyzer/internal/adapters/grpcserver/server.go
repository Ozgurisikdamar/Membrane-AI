// Package grpcserver adapts the AnalyzerService gRPC contract to the analyzer
// use-case, mapping typed domain errors to gRPC codes at the boundary.
package grpcserver

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pkgerrs "github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
	analyzerv1 "github.com/Ozgurisikdamar/Membrane-AI/proto/gen/membrane/analyzer/v1"
	"github.com/Ozgurisikdamar/Membrane-AI/services/analyzer/internal/app"
	"github.com/Ozgurisikdamar/Membrane-AI/services/analyzer/internal/domain"
)

// Analyzer is the use-case this adapter drives.
type Analyzer interface {
	Handle(ctx context.Context, in domain.Input) (domain.Result, error)
}

// Server implements analyzerv1.AnalyzerServiceServer.
type Server struct {
	analyzerv1.UnimplementedAnalyzerServiceServer
	analyze Analyzer
}

// New returns a gRPC server backed by the given use-case.
func New(analyze Analyzer) *Server { return &Server{analyze: analyze} }

// Analyze handles one analysis request.
func (s *Server) Analyze(ctx context.Context, req *analyzerv1.AnalyzeRequest) (*analyzerv1.AnalyzeResponse, error) {
	in, err := domain.NewInput(
		req.GetSubmissionId(), req.GetOrganizationId(), req.GetRepository(),
		req.GetFilePath(), req.GetLanguage(), req.GetDiff(),
	)
	if err != nil {
		return nil, statusFromErr(err)
	}
	res, err := s.analyze.Handle(ctx, in)
	if err != nil {
		return nil, statusFromErr(err)
	}
	findings := make([]*analyzerv1.Finding, 0, len(res.Findings))
	for _, f := range res.Findings {
		findings = append(findings, &analyzerv1.Finding{
			Rule:     f.Rule,
			Severity: severityToProto(f.Severity),
			Message:  f.Message,
			Line:     int32(f.Line), //nolint:gosec // diff line counts are far below int32 limits
		})
	}
	return &analyzerv1.AnalyzeResponse{Findings: findings, MaskedDiff: res.MaskedDiff}, nil
}

func severityToProto(s domain.Severity) analyzerv1.Severity {
	switch s {
	case domain.SeverityInfo:
		return analyzerv1.Severity_SEVERITY_INFO
	case domain.SeverityWarning:
		return analyzerv1.Severity_SEVERITY_WARNING
	case domain.SeverityBlocking:
		return analyzerv1.Severity_SEVERITY_BLOCKING
	default:
		return analyzerv1.Severity_SEVERITY_UNSPECIFIED
	}
}

func statusFromErr(err error) error {
	switch pkgerrs.KindOf(err) {
	case pkgerrs.KindValidation:
		return status.Error(codes.InvalidArgument, err.Error())
	case pkgerrs.KindUnavailable:
		return status.Error(codes.Unavailable, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}

// Ensure the concrete use-case satisfies the local interface.
var _ Analyzer = (*app.AnalyzeDiff)(nil)

package grpcstage_test

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"

	pkgerrs "github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
	analyzerv1 "github.com/Ozgurisikdamar/Membrane-AI/proto/gen/membrane/analyzer/v1"
	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/adapters/grpcstage"
	"github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator/internal/domain"
)

// fakeAnalyzer is an in-test implementation of the analyzer contract.
type fakeAnalyzer struct {
	analyzerv1.UnimplementedAnalyzerServiceServer
	resp *analyzerv1.AnalyzeResponse
	err  error
}

func (f *fakeAnalyzer) Analyze(context.Context, *analyzerv1.AnalyzeRequest) (*analyzerv1.AnalyzeResponse, error) {
	return f.resp, f.err
}

// newStage spins up a bufconn analyzer and returns a Stage dialed against it.
func newStage(t *testing.T, fa *fakeAnalyzer) *grpcstage.Stage {
	t.Helper()
	lis := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer()
	analyzerv1.RegisterAnalyzerServiceServer(srv, fa)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	stage, err := grpcstage.New("passthrough:///bufnet",
		grpcstage.WithDialOption(grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		})),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = stage.Close() })
	return stage
}

func sub(t *testing.T) domain.Submission {
	t.Helper()
	s, err := domain.NewSubmission("s-1", "o-1", "r", "f.go", "go", "+diff", "ide", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestAnalyze_MapsFindings(t *testing.T) {
	stage := newStage(t, &fakeAnalyzer{resp: &analyzerv1.AnalyzeResponse{
		Findings: []*analyzerv1.Finding{
			{Rule: "aws-access-key-id", Severity: analyzerv1.Severity_SEVERITY_BLOCKING, Message: "m", Line: 2},
			{Rule: "weird", Severity: analyzerv1.Severity_SEVERITY_UNSPECIFIED, Message: "u"},
		},
	}})
	findings, err := stage.Analyze(context.Background(), sub(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 2 {
		t.Fatalf("findings = %+v", findings)
	}
	if findings[0].Severity != domain.SeverityBlocking || findings[0].Stage != "analyzer" {
		t.Fatalf("first = %+v", findings[0])
	}
	if findings[1].Severity != domain.SeverityWarning {
		t.Fatalf("unknown severity must fail safe as warning: %+v", findings[1])
	}
}

func TestAnalyze_TransportErrorIsUnavailable(t *testing.T) {
	stage := newStage(t, &fakeAnalyzer{err: errors.New("boom")})
	_, err := stage.Analyze(context.Background(), sub(t))
	if pkgerrs.KindOf(err) != pkgerrs.KindUnavailable {
		t.Fatalf("kind = %v, want unavailable", pkgerrs.KindOf(err))
	}
}

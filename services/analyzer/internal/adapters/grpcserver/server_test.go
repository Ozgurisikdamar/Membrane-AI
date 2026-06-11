package grpcserver_test

import (
	"context"
	"net"
	"strings"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	analyzerv1 "github.com/Ozgurisikdamar/Membrane-AI/proto/gen/membrane/analyzer/v1"
	"github.com/Ozgurisikdamar/Membrane-AI/services/analyzer/internal/adapters/grpcserver"
	"github.com/Ozgurisikdamar/Membrane-AI/services/analyzer/internal/app"
	"github.com/Ozgurisikdamar/Membrane-AI/services/analyzer/internal/domain"
)

func dialBuf(t *testing.T) analyzerv1.AnalyzerServiceClient {
	t.Helper()
	lis := bufconn.Listen(1024 * 1024)
	uc := app.NewAnalyzeDiff(domain.NewSecretDetector(), domain.NewRiskyPatternDetector())
	srv := grpc.NewServer()
	analyzerv1.RegisterAnalyzerServiceServer(srv, grpcserver.New(uc))
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return analyzerv1.NewAnalyzerServiceClient(conn)
}

func TestAnalyze_FindsAndMasks(t *testing.T) {
	client := dialBuf(t)
	resp, err := client.Analyze(context.Background(), &analyzerv1.AnalyzeRequest{
		SubmissionId:   "s-1",
		OrganizationId: "o-1",
		Language:       "go",
		Diff:           "+key := \"AKIAIOSFODNN7EXAMPLE\"",
	})
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if len(resp.GetFindings()) != 1 {
		t.Fatalf("findings = %+v", resp.GetFindings())
	}
	f := resp.GetFindings()[0]
	if f.GetRule() != "aws-access-key-id" || f.GetSeverity() != analyzerv1.Severity_SEVERITY_BLOCKING || f.GetLine() != 1 {
		t.Fatalf("finding = %+v", f)
	}
	if strings.Contains(resp.GetMaskedDiff(), "AKIA") {
		t.Fatalf("masked diff leaked the secret: %s", resp.GetMaskedDiff())
	}
}

func TestAnalyze_EmptyDiffIsInvalid(t *testing.T) {
	client := dialBuf(t)
	_, err := client.Analyze(context.Background(), &analyzerv1.AnalyzeRequest{Language: "go"})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("code = %v, want InvalidArgument", status.Code(err))
	}
}

func TestAnalyze_CleanDiff(t *testing.T) {
	client := dialBuf(t)
	resp, err := client.Analyze(context.Background(), &analyzerv1.AnalyzeRequest{
		Language: "go", Diff: "+x := 1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.GetFindings()) != 0 || resp.GetMaskedDiff() != "+x := 1" {
		t.Fatalf("resp = %+v", resp)
	}
}

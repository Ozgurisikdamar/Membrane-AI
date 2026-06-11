package grpcserver_test

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	ingestionv1 "github.com/Ozgurisikdamar/Membrane-AI/proto/gen/membrane/ingestion/v1"
	"github.com/Ozgurisikdamar/Membrane-AI/services/ingestion/internal/adapters/grpcserver"
	"github.com/Ozgurisikdamar/Membrane-AI/services/ingestion/internal/adapters/idgen"
	"github.com/Ozgurisikdamar/Membrane-AI/services/ingestion/internal/adapters/memory"
	"github.com/Ozgurisikdamar/Membrane-AI/services/ingestion/internal/app"
)

func dialBuf(t *testing.T) (ingestionv1.IngestionServiceClient, *memory.Publisher) {
	t.Helper()
	lis := bufconn.Listen(1024 * 1024)
	pub := memory.NewPublisher()
	uc := app.NewEnqueueSubmission(pub, idgen.UUID{}, idgen.SystemClock{})

	srv := grpc.NewServer()
	ingestionv1.RegisterIngestionServiceServer(srv, grpcserver.New(uc))
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
	return ingestionv1.NewIngestionServiceClient(conn), pub
}

func validSubmission() *ingestionv1.CodeSubmission {
	return &ingestionv1.CodeSubmission{
		OrganizationId: "org-1",
		Repository:     "repo",
		FilePath:       "a.go",
		Language:       "go",
		Diff:           "diff",
		Origin:         ingestionv1.Origin_ORIGIN_IDE,
	}
}

func TestSubmitDiff_OK(t *testing.T) {
	client, pub := dialBuf(t)
	resp, err := client.SubmitDiff(context.Background(), &ingestionv1.SubmitDiffRequest{Submission: validSubmission()})
	if err != nil {
		t.Fatalf("SubmitDiff: %v", err)
	}
	if resp.GetSubmissionId() == "" || resp.GetStatus() != app.StatusQueued {
		t.Fatalf("resp = %+v", resp)
	}
	if len(pub.Events()) != 1 {
		t.Fatalf("expected 1 event, got %d", len(pub.Events()))
	}
}

func TestSubmitDiff_InvalidArgument(t *testing.T) {
	client, _ := dialBuf(t)
	_, err := client.SubmitDiff(context.Background(), &ingestionv1.SubmitDiffRequest{
		Submission: &ingestionv1.CodeSubmission{OrganizationId: ""}, // missing required fields
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("code = %v, want InvalidArgument", status.Code(err))
	}
}

func TestStreamCodeDiff_OK(t *testing.T) {
	client, pub := dialBuf(t)
	stream, err := client.StreamCodeDiff(context.Background())
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}
	const n = 3
	for i := 0; i < n; i++ {
		if err := stream.Send(&ingestionv1.StreamCodeDiffRequest{Submission: validSubmission()}); err != nil {
			t.Fatalf("send: %v", err)
		}
		resp, err := stream.Recv()
		if err != nil {
			t.Fatalf("recv: %v", err)
		}
		if resp.GetSubmissionId() == "" {
			t.Fatal("empty submission id in ack")
		}
	}
	if err := stream.CloseSend(); err != nil {
		t.Fatalf("close send: %v", err)
	}
	if len(pub.Events()) != n {
		t.Fatalf("expected %d events, got %d", n, len(pub.Events()))
	}
}

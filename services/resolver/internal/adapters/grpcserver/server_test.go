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

	resolverv1 "github.com/Ozgurisikdamar/Membrane-AI/proto/gen/membrane/resolver/v1"
	"github.com/Ozgurisikdamar/Membrane-AI/services/resolver/internal/adapters/grpcserver"
	"github.com/Ozgurisikdamar/Membrane-AI/services/resolver/internal/domain"
)

const orgUUID = "0b7e3f6a-1f2d-4c5b-9e8d-2a1b3c4d5e6f"

type fakeResolver struct {
	got     domain.Query
	matches []domain.GoldMatch
	err     error
}

func (f *fakeResolver) Handle(_ context.Context, q domain.Query) ([]domain.GoldMatch, error) {
	f.got = q
	return f.matches, f.err
}

func dialBuf(t *testing.T, r grpcserver.Resolver) resolverv1.ResolverServiceClient {
	t.Helper()
	lis := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer()
	resolverv1.RegisterResolverServiceServer(srv, grpcserver.New(r))
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
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return resolverv1.NewResolverServiceClient(conn)
}

func TestResolveContext_MapsRequestAndResponse(t *testing.T) {
	fake := &fakeResolver{matches: []domain.GoldMatch{
		{FilePath: "a.go", RawCodeContent: "code", ArchitecturalContext: "ctx", CosineDistance: 0.12},
	}}
	client := dialBuf(t, fake)

	resp, err := client.ResolveContext(context.Background(), &resolverv1.ResolveContextRequest{
		OrganizationId: orgUUID, Language: "GO", Diff: "+x", Limit: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	if fake.got.Language != "go" || fake.got.Limit != domain.DefaultLimit {
		t.Fatalf("query = %+v", fake.got)
	}
	if len(resp.GetMatches()) != 1 || resp.GetMatches()[0].GetFilePath() != "a.go" {
		t.Fatalf("matches = %+v", resp.GetMatches())
	}
	if resp.GetMatches()[0].GetCosineDistance() != 0.12 {
		t.Fatalf("distance = %v", resp.GetMatches()[0].GetCosineDistance())
	}
}

func TestResolveContext_InvalidOrgIsInvalidArgument(t *testing.T) {
	client := dialBuf(t, &fakeResolver{})
	_, err := client.ResolveContext(context.Background(), &resolverv1.ResolveContextRequest{
		OrganizationId: "not-a-uuid", Language: "go", Diff: "+x",
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("code = %v, want InvalidArgument", status.Code(err))
	}
}

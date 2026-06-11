package semanticstage

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
	resolverv1 "github.com/Ozgurisikdamar/Membrane-AI/proto/gen/membrane/resolver/v1"
)

const opFetcher = "orchestrator.adapters.semanticstage.ResolverFetcher"

// ResolverFetcher implements ContextFetcher against the resolver service
// (membrane.resolver.v1).
type ResolverFetcher struct {
	conn   *grpc.ClientConn
	client resolverv1.ResolverServiceClient
}

// NewResolverFetcher dials the resolver at addr (lazy connection).
func NewResolverFetcher(addr string, options ...grpc.DialOption) (*ResolverFetcher, error) {
	dialOpts := append(
		[]grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())},
		options...,
	)
	conn, err := grpc.NewClient(addr, dialOpts...)
	if err != nil {
		return nil, errs.Unavailable(opFetcher, "dial resolver", err)
	}
	return &ResolverFetcher{conn: conn, client: resolverv1.NewResolverServiceClient(conn)}, nil
}

// Fetch implements ContextFetcher (top-3 matches per resolver defaults).
func (f *ResolverFetcher) Fetch(ctx context.Context, orgID, language, diff string) ([]GoldContext, error) {
	resp, err := f.client.ResolveContext(ctx, &resolverv1.ResolveContextRequest{
		OrganizationId: orgID,
		Language:       language,
		Diff:           diff,
	})
	if err != nil {
		return nil, errs.Unavailable(opFetcher, "resolve context", err)
	}
	gold := make([]GoldContext, 0, len(resp.GetMatches()))
	for _, m := range resp.GetMatches() {
		gold = append(gold, GoldContext{
			FilePath:             m.GetFilePath(),
			Code:                 m.GetRawCodeContent(),
			ArchitecturalContext: m.GetArchitecturalContext(),
		})
	}
	return gold, nil
}

// Close releases the client connection.
func (f *ResolverFetcher) Close() error { return f.conn.Close() }

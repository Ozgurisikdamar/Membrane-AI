// Package grpcserver adapts the ResolverService gRPC contract to the resolver
// use-case, mapping typed domain errors to gRPC codes at the boundary.
package grpcserver

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pkgerrs "github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
	resolverv1 "github.com/Ozgurisikdamar/Membrane-AI/proto/gen/membrane/resolver/v1"
	"github.com/Ozgurisikdamar/Membrane-AI/services/resolver/internal/app"
	"github.com/Ozgurisikdamar/Membrane-AI/services/resolver/internal/domain"
)

// Resolver is the use-case this adapter drives.
type Resolver interface {
	Handle(ctx context.Context, q domain.Query) ([]domain.GoldMatch, error)
}

// Server implements resolverv1.ResolverServiceServer.
type Server struct {
	resolverv1.UnimplementedResolverServiceServer
	resolve Resolver
}

// New returns a gRPC server backed by the given use-case.
func New(resolve Resolver) *Server { return &Server{resolve: resolve} }

// ResolveContext handles one retrieval request.
func (s *Server) ResolveContext(ctx context.Context, req *resolverv1.ResolveContextRequest) (*resolverv1.ResolveContextResponse, error) {
	q, err := domain.NewQuery(req.GetOrganizationId(), req.GetLanguage(), req.GetDiff(), int(req.GetLimit()))
	if err != nil {
		return nil, statusFromErr(err)
	}
	matches, err := s.resolve.Handle(ctx, q)
	if err != nil {
		return nil, statusFromErr(err)
	}
	out := make([]*resolverv1.GoldMatch, 0, len(matches))
	for _, m := range matches {
		out = append(out, &resolverv1.GoldMatch{
			FilePath:             m.FilePath,
			RawCodeContent:       m.RawCodeContent,
			ArchitecturalContext: m.ArchitecturalContext,
			CosineDistance:       m.CosineDistance,
		})
	}
	return &resolverv1.ResolveContextResponse{Matches: out}, nil
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
var _ Resolver = (*app.ResolveContext)(nil)

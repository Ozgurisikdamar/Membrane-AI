// Package grpcserver adapts the IngestionService gRPC contract to the
// application use-cases. It translates protobuf messages to/from the domain and
// maps typed domain errors to gRPC status codes at the boundary.
package grpcserver

import (
	"context"
	"errors"
	"io"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pkgerrs "github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
	ingestionv1 "github.com/Ozgurisikdamar/Membrane-AI/proto/gen/membrane/ingestion/v1"
	"github.com/Ozgurisikdamar/Membrane-AI/services/ingestion/internal/app"
	"github.com/Ozgurisikdamar/Membrane-AI/services/ingestion/internal/domain"
)

// Enqueuer is the use-case this adapter drives (declared here so the adapter
// depends on a narrow interface, not the concrete struct).
type Enqueuer interface {
	Handle(ctx context.Context, sub domain.Submission) (app.EnqueueResult, error)
}

// Server implements ingestionv1.IngestionServiceServer.
type Server struct {
	ingestionv1.UnimplementedIngestionServiceServer
	enqueue Enqueuer
}

// New returns a gRPC server backed by the given use-case.
func New(enqueue Enqueuer) *Server { return &Server{enqueue: enqueue} }

// SubmitDiff handles a one-shot submission.
func (s *Server) SubmitDiff(ctx context.Context, req *ingestionv1.SubmitDiffRequest) (*ingestionv1.SubmitDiffResponse, error) {
	sub, err := toDomain(req.GetSubmission())
	if err != nil {
		return nil, statusFromErr(err)
	}
	res, err := s.enqueue.Handle(ctx, sub)
	if err != nil {
		return nil, statusFromErr(err)
	}
	return &ingestionv1.SubmitDiffResponse{SubmissionId: res.SubmissionID, Status: res.Status}, nil
}

// StreamCodeDiff handles a duplex stream of submissions, acking each one.
func (s *Server) StreamCodeDiff(stream ingestionv1.IngestionService_StreamCodeDiffServer) error {
	ctx := stream.Context()
	for {
		req, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		sub, derr := toDomain(req.GetSubmission())
		if derr != nil {
			return statusFromErr(derr)
		}
		res, herr := s.enqueue.Handle(ctx, sub)
		if herr != nil {
			return statusFromErr(herr)
		}
		if err := stream.Send(&ingestionv1.StreamCodeDiffResponse{
			SubmissionId: res.SubmissionID,
			Status:       res.Status,
		}); err != nil {
			return err
		}
	}
}

func toDomain(cs *ingestionv1.CodeSubmission) (domain.Submission, error) {
	if cs == nil {
		return domain.Submission{}, pkgerrs.Validation("ingestion.grpc", "submission is required", nil)
	}
	sub, err := domain.NewSubmission(
		cs.GetOrganizationId(),
		cs.GetRepository(),
		cs.GetFilePath(),
		cs.GetLanguage(),
		cs.GetDiff(),
		originFromProto(cs.GetOrigin()),
	)
	if err != nil {
		return domain.Submission{}, err
	}
	return sub.WithSource(cs.GetCommitSha(), int(cs.GetPrNumber())), nil
}

func originFromProto(o ingestionv1.Origin) domain.Origin {
	switch o {
	case ingestionv1.Origin_ORIGIN_IDE:
		return domain.OriginIDE
	case ingestionv1.Origin_ORIGIN_WEBHOOK:
		return domain.OriginWebhook
	default:
		return domain.OriginUnspecified
	}
}

func statusFromErr(err error) error {
	return status.Error(grpcCode(pkgerrs.KindOf(err)), err.Error())
}

func grpcCode(k pkgerrs.Kind) codes.Code {
	switch k {
	case pkgerrs.KindValidation:
		return codes.InvalidArgument
	case pkgerrs.KindNotFound:
		return codes.NotFound
	case pkgerrs.KindConflict:
		return codes.AlreadyExists
	case pkgerrs.KindUnauthorized:
		return codes.Unauthenticated
	case pkgerrs.KindUnavailable:
		return codes.Unavailable
	default:
		return codes.Internal
	}
}

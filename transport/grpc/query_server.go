package grpc

import (
	"context"

	errorfamily "github.com/larsartmann/go-error-family"
	"google.golang.org/grpc"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	cqrsproto "github.com/larsartmann/go-cqrs-lite/transport/grpc/v4/proto"
)

// QueryDispatcher is the server-side interface that the gRPC query service
// delegates to. [query.Dispatcher] satisfies this interface.
type QueryDispatcher interface {
	Dispatch(ctx context.Context, q query.Query) (any, error)
}

// RegisterQueryService registers a CQRS query dispatch service on the
// given gRPC server. Remote clients can dispatch queries via gRPC.
//
// Query payloads and results are encoded on the wire using the configured
// codec (JSON by default; override with [WithCodec]).
func RegisterQueryService(srv *grpc.Server, dispatcher QueryDispatcher, opts ...Option) {
	cfg := resolveConfig(opts)
	cqrsproto.RegisterQueryServiceServer(
		srv,
		&queryServer{ //nolint:exhaustruct // grpc server pattern: embedded Unimplemented is zero-valued
			dispatcher: dispatcher,
			codec:      cfg.codec,
		},
	)
}

type queryServer struct {
	cqrsproto.UnimplementedQueryServiceServer

	dispatcher QueryDispatcher
	codec      codec.Codec
}

func (s *queryServer) Ask(
	ctx context.Context,
	envelope *cqrsproto.QueryEnvelope,
) (*cqrsproto.QueryResult, error) {
	ctx, span := cqrsotel.StartSpan(
		ctx, tracer(), "grpc.query.ask",
		cqrsotel.SpanKindServer,
		cqrsotel.WithAttributes(
			cqrsotel.AttrString(cqrsotel.AttrQueryType, envelope.GetType()),
		),
	)
	defer span.End()

	q, err := query.New(query.Type(envelope.GetType()))
	if err != nil {
		cqrsotel.RecordError(span, err)

		return queryErrorResult(err), nil
	}

	result, err := s.dispatcher.Dispatch(ctx, q)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return queryErrorResult(err), nil
	}

	payload, err := s.codec.Encode(result)
	if err != nil {
		return queryErrorResult(errorfamily.WrapCorruption(err, "grpc.query.marshal_result",
			"marshal result")), nil
	}

	return &cqrsproto.QueryResult{Payload: payload}, nil //nolint:exhaustruct // proto
}

func queryErrorResult(err error) *cqrsproto.QueryResult {
	code, family := classifyError(err)

	return &cqrsproto.QueryResult{ //nolint:exhaustruct // proto
		Error:       err.Error(),
		ErrorCode:   code,
		ErrorFamily: family,
	}
}

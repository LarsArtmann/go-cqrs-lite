package grpc

import (
	"context"
	"encoding/json"
	"fmt"

	"google.golang.org/grpc"

	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v3"
	"github.com/larsartmann/go-cqrs-lite/query/v3"
	cqrsproto "github.com/larsartmann/go-cqrs-lite/transport/grpc/v3/proto"
)

// QueryDispatcher is the server-side interface that the gRPC query service
// delegates to. [query.Dispatcher] satisfies this interface.
type QueryDispatcher interface {
	Dispatch(ctx context.Context, q query.Query) (any, error)
}

// RegisterQueryService registers a CQRS query dispatch service on the
// given gRPC server. Remote clients can dispatch queries via gRPC.
//
// Query payloads and results are JSON-encoded on the wire.
func RegisterQueryService(srv *grpc.Server, dispatcher QueryDispatcher) {
	cqrsproto.RegisterQueryServiceServer(
		srv,
		&queryServer{dispatcher: dispatcher}, //nolint:exhaustruct // grpc server pattern
	)
}

type queryServer struct {
	cqrsproto.UnimplementedQueryServiceServer

	dispatcher QueryDispatcher
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

	payload, err := json.Marshal(result)
	if err != nil {
		return queryErrorResult(fmt.Errorf("marshal result: %w", err)), nil
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

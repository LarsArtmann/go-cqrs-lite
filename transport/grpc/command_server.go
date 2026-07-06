package grpc

import (
	"context"

	errorfamily "github.com/larsartmann/go-error-family"
	"google.golang.org/grpc"

	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v3"
	cqrsproto "github.com/larsartmann/go-cqrs-lite/transport/grpc/v3/proto"
)

// CommandDispatcher is the server-side interface that the gRPC command
// service delegates to. [command.Dispatcher] satisfies this interface.
type CommandDispatcher interface {
	Dispatch(ctx context.Context, cmd command.Command) error
}

// RegisterCommandService registers a CQRS command dispatch service on the
// given gRPC server. Remote clients can dispatch commands via gRPC.
//
// Unlike [RegisterQueryService], the command service does NOT accept a
// [WithCodec] option. Command payloads travel as metadata strings (not
// codec-encoded wire bytes), so there is no codec to configure. If you need
// structured command payloads over gRPC, encode them client-side and pass
// via command metadata custom fields — the server-side handler extracts them
// from cmd.Metadata().Custom.
func RegisterCommandService(srv *grpc.Server, dispatcher CommandDispatcher) {
	cqrsproto.RegisterCommandServiceServer(
		srv,
		&commandServer{dispatcher: dispatcher}, //nolint:exhaustruct // grpc server pattern
	)
}

type commandServer struct {
	cqrsproto.UnimplementedCommandServiceServer

	dispatcher CommandDispatcher
}

func (s *commandServer) Dispatch(
	ctx context.Context,
	envelope *cqrsproto.CommandEnvelope,
) (*cqrsproto.CommandResult, error) {
	ctx, span := cqrsotel.StartSpan(
		ctx, tracer(), "grpc.command.dispatch",
		cqrsotel.SpanKindServer,
		cqrsotel.WithAttributes(
			cqrsotel.AttrString(cqrsotel.AttrCommandType, envelope.GetType()),
			cqrsotel.AttrString(cqrsotel.AttrAggregateID, envelope.GetAggregateId()),
		),
	)
	defer span.End()

	aggID, err := id.ParseAggregateID(envelope.GetAggregateId())
	if err != nil {
		cqrsotel.RecordError(span, err)

		return errorResult(errorfamily.WrapRejection(err, "grpc.command.parse_aggregate_id",
			"parse aggregate ID")), nil
	}

	var opts []command.Option

	if cmdIDStr := envelope.GetMetadata()["command_id"]; cmdIDStr != "" {
		if cmdID, err := id.ParseCommandID(cmdIDStr); err == nil {
			opts = append(opts, command.WithCommandID(cmdID))
		}
	}

	for k, v := range envelope.GetMetadata() {
		if k == "command_id" {
			continue
		}

		opts = append(opts, command.WithCustomMetadata(k, v))
	}

	if len(envelope.GetPayload()) > 0 {
		opts = append(opts, command.WithCustomMetadata("payload", string(envelope.GetPayload())))
	}

	cmd, err := command.New(command.Type(envelope.GetType()), aggID, opts...)
	if err != nil {
		return errorResult(errorfamily.WrapRejection(err, "grpc.command.create",
			"create command")), nil
	}

	err = s.dispatcher.Dispatch(ctx, cmd)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return errorResult(err), nil
	}

	return &cqrsproto.CommandResult{Success: true, Error: "", ErrorCode: "", ErrorFamily: ""}, nil
}

func errorResult(err error) *cqrsproto.CommandResult {
	code, family := classifyError(err)

	return &cqrsproto.CommandResult{
		Success:     false,
		Error:       err.Error(),
		ErrorCode:   code,
		ErrorFamily: family,
	}
}

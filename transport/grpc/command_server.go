package grpc

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
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
// Command payloads are JSON-encoded on the wire. When non-empty, the payload
// is stored in the command metadata under the key "payload" as a JSON string.
// Server-side handlers extract it via cmd.Metadata().Custom["payload"].
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
	aggID, err := id.ParseAggregateID(envelope.GetAggregateId())
	if err != nil {
		return errorResult(fmt.Errorf("parse aggregate ID: %w", err)), nil
	}

	var opts []command.Option

	for k, v := range envelope.GetMetadata() {
		opts = append(opts, command.WithCustomMetadata(k, v))
	}

	if len(envelope.GetPayload()) > 0 {
		opts = append(opts, command.WithCustomMetadata("payload", string(envelope.GetPayload())))
	}

	cmd, err := command.New(command.Type(envelope.GetType()), aggID, opts...)
	if err != nil {
		return errorResult(fmt.Errorf("create command: %w", err)), nil
	}

	err = s.dispatcher.Dispatch(ctx, cmd)
	if err != nil {
		return errorResult(err), nil
	}

	return &cqrsproto.CommandResult{Success: true, Error: ""}, nil
}

func errorResult(err error) *cqrsproto.CommandResult {
	return &cqrsproto.CommandResult{
		Success: false,
		Error:   err.Error(),
	}
}

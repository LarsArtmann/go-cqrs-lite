package grpc

import (
	"context"
	"fmt"

	errorfamily "github.com/larsartmann/go-error-family"
	"google.golang.org/grpc"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	cqrsproto "github.com/larsartmann/go-cqrs-lite/transport/grpc/v4/proto"
)

var (
	errDispatchFailed = errorfamily.NewInfrastructure(
		"grpc.dispatch_failed",
		"grpc: server returned failure",
	)
	errQueryFailed      = errorfamily.NewInfrastructure("grpc.query_failed", "grpc: query failed")
	errMissingCommandID = errorfamily.NewRejection(
		"grpc.missing_command_id",
		"grpc: command has no ID",
	)
)

// CommandClient dispatches commands to a remote gRPC server.
// It implements [CommandDispatcher], so it can be used anywhere a local
// dispatcher would be — the dispatch is transparently remote.
type CommandClient struct {
	client cqrsproto.CommandServiceClient
}

// NewCommandClient creates a remote command dispatcher backed by conn.
// The caller owns the connection lifecycle.
func NewCommandClient(conn *grpc.ClientConn) *CommandClient {
	return &CommandClient{client: cqrsproto.NewCommandServiceClient(conn)}
}

// Dispatch sends cmd to the remote gRPC server for processing.
// Returns an error if the server returned a failure or the RPC failed.
func (c *CommandClient) Dispatch(ctx context.Context, cmd command.Command) error {
	envelope := &cqrsproto.CommandEnvelope{ //nolint:exhaustruct // proto
		Type:        string(cmd.Type()),
		AggregateId: cmd.AggregateID().String(),
	}

	if cmd.ID().IsZero() {
		return errorfamily.WrapRejection(errMissingCommandID,
			"grpc.dispatch_missing_id",
			fmt.Sprintf("dispatch %s", cmd.Type()))
	}

	envelope.Metadata = make(map[string]string)
	envelope.Metadata["command_id"] = cmd.ID().String()

	// Carry metadata if the command is a *BasicCommand.
	if bc, ok := cmd.(*command.BasicCommand); ok {
		md := bc.Metadata()
		for k, v := range md.Custom {
			envelope.Metadata[string(k)] = v
		}
	}

	result, err := c.client.Dispatch(ctx, envelope)
	if err != nil {
		return errorfamily.WrapInfrastructure(err, "grpc.dispatch",
			fmt.Sprintf("dispatch %s", cmd.Type()))
	}

	if !result.GetSuccess() {
		return reconstructError(errDispatchFailed,
			result.GetError(), result.GetErrorCode(), result.GetErrorFamily())
	}

	return nil
}

// QueryClient dispatches queries to a remote gRPC server.
type QueryClient struct {
	client cqrsproto.QueryServiceClient
	codec  codec.Codec
}

// NewQueryClient creates a remote query dispatcher backed by conn.
// Pass [WithCodec] to override the default JSON wire encoding.
func NewQueryClient(conn *grpc.ClientConn, opts ...Option) *QueryClient {
	cfg := resolveConfig(opts)

	return &QueryClient{
		client: cqrsproto.NewQueryServiceClient(conn),
		codec:  cfg.codec,
	}
}

// Ask sends a query to the remote gRPC server and decodes the result into out.
// out must be a pointer. The wire encoding is determined by the client's codec
// (JSON by default; override via [WithCodec] at construction).
func (c *QueryClient) Ask(ctx context.Context, queryType string, out any) error {
	result, err := c.client.Ask(
		ctx,
		&cqrsproto.QueryEnvelope{Type: queryType}, //nolint:exhaustruct // proto
	)
	if err != nil {
		return errorfamily.WrapInfrastructure(err, "grpc.ask",
			"ask "+queryType)
	}

	if result.GetError() != "" {
		return reconstructError(errQueryFailed,
			result.GetError(), result.GetErrorCode(), result.GetErrorFamily())
	}

	err = c.codec.Decode(result.GetPayload(), out)
	if err != nil {
		return errorfamily.WrapCorruption(err, "grpc.unmarshal_result",
			"unmarshal query result")
	}

	return nil
}

// Compile-time assertion.
var _ CommandDispatcher = (*CommandClient)(nil)

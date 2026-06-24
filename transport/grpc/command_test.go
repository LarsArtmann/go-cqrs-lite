package grpc_test

import (
	"context"
	"errors"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	cqrsgrpc "github.com/larsartmann/go-cqrs-lite/transport/grpc/v3"
)

func listen(t *testing.T) net.Listener {
	t.Helper()

	lc := &net.ListenConfig{}

	lis, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}

	return lis
}

func TestCommandDispatch_RoundTrip(t *testing.T) {
	t.Parallel()

	lis := listen(t)

	srv := grpc.NewServer()

	var receivedCmdType command.Type
	var receivedAggID id.AggregateID

	cmdDispatcher := command.NewDispatcher()
	_ = cmdDispatcher.Register("test.cmd", func(_ context.Context, cmd command.Command) error {
		receivedCmdType = cmd.Type()
		receivedAggID = cmd.AggregateID()

		return nil
	})

	cqrsgrpc.RegisterCommandService(srv, cmdDispatcher)

	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient(
		lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	t.Cleanup(func() { _ = conn.Close() })

	client := cqrsgrpc.NewCommandClient(conn)

	aggID := id.NewAggregateID()
	cmd, err := command.New("test.cmd", aggID)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	err = client.Dispatch(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}

	if receivedCmdType != "test.cmd" {
		t.Fatalf("received type: got %s, want test.cmd", receivedCmdType)
	}

	if receivedAggID != aggID {
		t.Fatalf("received aggID: got %s, want %s", receivedAggID, aggID)
	}
}

func TestCommandDispatch_HandlerError(t *testing.T) {
	t.Parallel()

	lis := listen(t)

	srv := grpc.NewServer()

	handlerError := errors.New("handler failed")
	cmdDispatcher := command.NewDispatcher()
	_ = cmdDispatcher.Register("fail.cmd", func(_ context.Context, _ command.Command) error {
		return handlerError
	})

	cqrsgrpc.RegisterCommandService(srv, cmdDispatcher)

	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient(
		lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	t.Cleanup(func() { _ = conn.Close() })

	client := cqrsgrpc.NewCommandClient(conn)

	aggID := id.NewAggregateID()
	cmd, _ := command.New("fail.cmd", aggID)

	err = client.Dispatch(context.Background(), cmd)
	if err == nil {
		t.Fatal("Dispatch: expected error, got nil")
	}
}

func TestCommandDispatch_PayloadInMetadata(t *testing.T) {
	t.Parallel()

	lis := listen(t)

	srv := grpc.NewServer()

	var receivedPayload string

	cmdDispatcher := command.NewDispatcher()
	_ = cmdDispatcher.Register("data.cmd", func(_ context.Context, cmd command.Command) error {
		if bc, ok := cmd.(*command.BasicCommand); ok {
			md := bc.Metadata()
			receivedPayload = md.Custom["payload"]
		}

		return nil
	})

	cqrsgrpc.RegisterCommandService(srv, cmdDispatcher)

	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient(
		lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	t.Cleanup(func() { _ = conn.Close() })

	client := cqrsgrpc.NewCommandClient(conn)

	aggID := id.NewAggregateID()
	cmd, _ := command.New(
		"data.cmd", aggID,
		command.WithCustomMetadata("payload", `{"name":"Alice"}`),
	)

	err = client.Dispatch(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}

	if receivedPayload != `{"name":"Alice"}` {
		t.Fatalf("received payload: got %s, want Alice", receivedPayload)
	}
}

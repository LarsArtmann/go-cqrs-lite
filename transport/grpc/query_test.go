package grpc_test

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/larsartmann/go-cqrs-lite/codec/v3"
	"github.com/larsartmann/go-cqrs-lite/query/v3"
	cqrsgrpc "github.com/larsartmann/go-cqrs-lite/transport/grpc/v3"
)

type userResult struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func queryDispatcher(t *testing.T) (*query.Dispatcher, *userResult) {
	t.Helper()

	result := &userResult{Name: "Alice", Email: "alice@test.com"}
	disp := query.NewDispatcher()

	err := disp.Register("user.get", func(_ context.Context, _ query.Query) (any, error) {
		return result, nil
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	return disp, result
}

func TestQueryRoundTrip_JSON(t *testing.T) {
	t.Parallel()

	lis := listen(t)
	srv := grpc.NewServer()

	disp, want := queryDispatcher(t)
	cqrsgrpc.RegisterQueryService(srv, disp)

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

	client := cqrsgrpc.NewQueryClient(conn)

	var got userResult
	if err := client.Ask(context.Background(), "user.get", &got); err != nil {
		t.Fatalf("Ask: %v", err)
	}

	if got != *want {
		t.Errorf("Ask result = %+v, want %+v", got, *want)
	}
}

func TestQueryRoundTrip_CBOR(t *testing.T) {
	t.Parallel()

	lis := listen(t)
	srv := grpc.NewServer()

	disp, want := queryDispatcher(t)
	cqrsgrpc.RegisterQueryService(srv, disp, cqrsgrpc.WithCodec(codec.CBORCodec{}))

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

	client := cqrsgrpc.NewQueryClient(conn, cqrsgrpc.WithCodec(codec.CBORCodec{}))

	var got userResult
	if err := client.Ask(context.Background(), "user.get", &got); err != nil {
		t.Fatalf("Ask: %v", err)
	}

	if got != *want {
		t.Errorf("Ask result = %+v, want %+v", got, *want)
	}
}

func TestQueryRoundTrip_HandlerError(t *testing.T) {
	t.Parallel()

	lis := listen(t)
	srv := grpc.NewServer()

	disp := query.NewDispatcher()
	_ = disp.Register("fail.query", func(_ context.Context, _ query.Query) (any, error) {
		return nil, errors.New("simulated failure")
	})

	cqrsgrpc.RegisterQueryService(srv, disp)

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

	client := cqrsgrpc.NewQueryClient(conn)

	var got any
	err = client.Ask(context.Background(), "fail.query", &got)
	if err == nil {
		t.Fatal("Ask: expected error, got nil")
	}
}

func TestQueryRoundTrip_CodecMismatch(t *testing.T) {
	t.Parallel()

	lis := listen(t)
	srv := grpc.NewServer()

	disp, _ := queryDispatcher(t)
	// Server encodes results as CBOR.
	cqrsgrpc.RegisterQueryService(srv, disp, cqrsgrpc.WithCodec(codec.CBORCodec{}))

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

	// Client expects JSON — should fail to decode CBOR bytes.
	client := cqrsgrpc.NewQueryClient(conn) // default JSON

	var got userResult
	err = client.Ask(context.Background(), "user.get", &got)
	if err == nil {
		t.Fatal("Ask: expected decode error from codec mismatch, got nil")
	}
}

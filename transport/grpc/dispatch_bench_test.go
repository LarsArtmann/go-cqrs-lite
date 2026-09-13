package grpc_test

import (
	"context"
	"net"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	cqrsgrpc "github.com/larsartmann/go-cqrs-lite/transport/grpc/v4"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// dispatch_bench_test.go — measures gRPC transport overhead for remote
// command and query dispatch.

// noopCommandDispatcher is a minimal CommandDispatcher for benchmarking.
type noopCommandDispatcher struct{}

func (noopCommandDispatcher) Dispatch(_ context.Context, _ command.Command) error {
	return nil
}

// BenchmarkGRPC_CommandDispatch measures the latency of dispatching a command
// through a local gRPC server+client. This is the overhead of the gRPC
// transport layer on top of the command path.
func BenchmarkGRPC_CommandDispatch(b *testing.B) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		b.Fatal(err)
	}

	srv := grpc.NewServer()
	cqrsgrpc.RegisterCommandService(srv, noopCommandDispatcher{})

	go func() { _ = srv.Serve(lis) }()
	defer srv.Stop()

	conn, err := grpc.NewClient(
		lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = conn.Close() }()

	client := cqrsgrpc.NewCommandClient(conn)
	ctx := context.Background()

	// Create a simple command.
	cmd, err := command.New("bench.command", id.NewStreamID())
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	for b.Loop() {
		if err := client.Dispatch(ctx, cmd); err != nil {
			b.Fatal(err)
		}
	}

	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "commands/sec")
}

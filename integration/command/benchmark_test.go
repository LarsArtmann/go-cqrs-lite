package command_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/testutil/v4"
)

func benchmarkDispatch(b *testing.B, dispatcher *command.Dispatcher) {
	cmd := testutil.NewCmd(b, "bench.cmd", id.NewStreamID())
	ctx := context.Background()

	for b.Loop() {
		err := dispatcher.Dispatch(ctx, cmd)
		if err != nil {
			b.Fatalf("dispatch: %v", err)
		}
	}
}

func passThroughMiddleware() command.Middleware {
	return func(next command.Handler) command.Handler {
		return func(ctx context.Context, cmd command.Command) error {
			return next(ctx, cmd)
		}
	}
}

func BenchmarkDispatcher_Dispatch(b *testing.B) {
	b.ReportAllocs()
	dispatcher := command.NewDispatcher()

	registerBenchCmd(b, dispatcher)

	benchmarkDispatch(b, dispatcher)
}

func BenchmarkDispatcher_Dispatch_WithMiddleware(b *testing.B) {
	b.ReportAllocs()
	dispatcher := command.NewDispatcher()

	dispatcher.Use(passThroughMiddleware())
	dispatcher.Use(passThroughMiddleware())

	registerBenchCmd(b, dispatcher)

	benchmarkDispatch(b, dispatcher)
}

func registerBenchCmd(b *testing.B, dispatcher *command.Dispatcher) {
	err := dispatcher.Register("bench.cmd", noopCommandHandler())
	if err != nil {
		b.Fatalf("register: %v", err)
	}
}

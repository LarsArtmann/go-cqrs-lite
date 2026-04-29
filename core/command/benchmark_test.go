package command_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/testhelpers"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

func benchmarkDispatch(b *testing.B, dispatcher *command.Dispatcher) {
	cmd := command.MustNew("bench.cmd", id.NewAggregateID())
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
	dispatcher := command.NewDispatcher()

	registerBenchCmd(b, dispatcher)

	benchmarkDispatch(b, dispatcher)
}

func BenchmarkDispatcher_Dispatch_WithMiddleware(b *testing.B) {
	dispatcher := command.NewDispatcher()

	dispatcher.Use(passThroughMiddleware())
	dispatcher.Use(passThroughMiddleware())

	registerBenchCmd(b, dispatcher)

	benchmarkDispatch(b, dispatcher)
}

func registerBenchCmd(b *testing.B, dispatcher *command.Dispatcher) {
	err := dispatcher.Register("bench.cmd", testhelpers.NoopCommandHandler())
	if err != nil {
		b.Fatalf("register: %v", err)
	}
}

package command_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command"
	"github.com/larsartmann/go-cqrs-lite/pkg/id"
)

func BenchmarkDispatcher_Dispatch(b *testing.B) {
	dispatcher := command.NewDispatcher()

	err := dispatcher.Register("bench.cmd", func(_ context.Context, _ command.Command) error {
		return nil
	})
	if err != nil {
		b.Fatalf("register: %v", err)
	}

	cmd := command.New("bench.cmd", id.NewAggregateID())
	ctx := context.Background()

	for b.Loop() {
		err := dispatcher.Dispatch(ctx, cmd)
		if err != nil {
			b.Fatalf("dispatch: %v", err)
		}
	}
}

func BenchmarkDispatcher_Dispatch_WithMiddleware(b *testing.B) {
	dispatcher := command.NewDispatcher()

	dispatcher.Use(func(next command.Handler) command.Handler {
		return func(ctx context.Context, cmd command.Command) error {
			return next(ctx, cmd)
		}
	})

	dispatcher.Use(func(next command.Handler) command.Handler {
		return func(ctx context.Context, cmd command.Command) error {
			return next(ctx, cmd)
		}
	})

	err := dispatcher.Register("bench.cmd", func(_ context.Context, _ command.Command) error {
		return nil
	})
	if err != nil {
		b.Fatalf("register: %v", err)
	}

	cmd := command.New("bench.cmd", id.NewAggregateID())
	ctx := context.Background()

	for b.Loop() {
		err := dispatcher.Dispatch(ctx, cmd)
		if err != nil {
			b.Fatalf("dispatch: %v", err)
		}
	}
}

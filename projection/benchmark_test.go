package projection_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/memory"
	"github.com/larsartmann/go-cqrs-lite/projection"
)

func BenchmarkRunner_Register(b *testing.B) {
	bus := memory.NewMemoryBus()
	b.Cleanup(func() { _ = bus.Close() })

	checkpoint := memory.NewCheckpointStore()
	b.Cleanup(func() { _ = checkpoint.Close() })

	runner, err := projection.NewRunner(nil, bus, checkpoint)
	if err != nil {
		b.Fatalf("NewRunner: %v", err)
	}

	for b.Loop() {
		err = runner.Register(event.NewProjection("noop",
			func(_ context.Context, _ event.Event) error { return nil },
			[]event.Type{"UserCreated"},
		))
		if err != nil {
			b.Fatalf("Register: %v", err)
		}
	}
}

func BenchmarkRunner_NewRunner(b *testing.B) {
	for b.Loop() {
		bus := memory.NewMemoryBus()
		checkpoint := memory.NewCheckpointStore()

		_, err := projection.NewRunner(nil, bus, checkpoint)
		if err != nil {
			b.Fatalf("NewRunner: %v", err)
		}

		_ = bus.Close()
		_ = checkpoint.Close()
	}
}

func BenchmarkRunner_CurrentCheckpoint(b *testing.B) {
	bus := memory.NewMemoryBus()
	b.Cleanup(func() { _ = bus.Close() })

	checkpoint := memory.NewCheckpointStore()
	b.Cleanup(func() { _ = checkpoint.Close() })

	ctx := context.Background()

	evtID := id.NewEventID()

	err := checkpoint.Save(ctx, "bench-proj", evtID)
	if err != nil {
		b.Fatalf("Save checkpoint: %v", err)
	}

	runner, err := projection.NewRunner(nil, bus, checkpoint)
	if err != nil {
		b.Fatalf("NewRunner: %v", err)
	}

	for b.Loop() {
		_, err = runner.CurrentCheckpoint(ctx, "bench-proj")
		if err != nil {
			b.Fatalf("CurrentCheckpoint: %v", err)
		}
	}
}

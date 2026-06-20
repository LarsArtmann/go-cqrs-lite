package projection_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/memory/v2"
	"github.com/larsartmann/go-cqrs-lite/projection/v2"
)

func BenchmarkRunner_Register(b *testing.B) {
	b.ReportAllocs()
	bus := eventtest.NewFakeBus()
	b.Cleanup(func() { _ = bus.Close() })

	checkpoint := memory.NewMemoryCheckpointStore()
	b.Cleanup(func() { _ = checkpoint.Close() })

	runner, err := projection.NewRunner(nil, bus, checkpoint)
	if err != nil {
		b.Fatalf("NewRunner: %v", err)
	}

	var i int

	for b.Loop() {
		err = runner.Register(event.NewProjection(
			fmt.Sprintf("noop-%d", i),
			eventtest.NoopEventHandler(),
			[]event.Type{"UserCreated"},
		))
		if err != nil {
			b.Fatalf("Register: %v", err)
		}

		i++
	}
}

func BenchmarkRunner_NewRunner(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		bus := eventtest.NewFakeBus()
		checkpoint := memory.NewMemoryCheckpointStore()

		_, err := projection.NewRunner(nil, bus, checkpoint)
		if err != nil {
			b.Fatalf("NewRunner: %v", err)
		}

		_ = bus.Close()
		_ = checkpoint.Close()
	}
}

func BenchmarkRunner_CurrentCheckpoint(b *testing.B) {
	b.ReportAllocs()
	bus := eventtest.NewFakeBus()
	b.Cleanup(func() { _ = bus.Close() })

	checkpoint := memory.NewMemoryCheckpointStore()
	b.Cleanup(func() { _ = checkpoint.Close() })

	ctx := context.Background()

	evtID := id.NewEventID()

	err := checkpoint.Save(
		ctx,
		"bench-proj",
		event.Checkpoint{EventID: evtID, ProcessedAt: time.Now()},
	)
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

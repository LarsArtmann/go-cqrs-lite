package projection_test

import (
	"log/slog"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/memory/v2"
	"github.com/larsartmann/go-cqrs-lite/projection/v2"
)

func TestRunner_Close(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	bus := memory.NewMemoryBus()
	checkpoint := memory.NewMemoryCheckpointStore()

	runner, err := projection.NewRunner(store, bus, checkpoint)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	err = runner.Close()
	if err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestWithLogger(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	bus := memory.NewMemoryBus()
	checkpoint := memory.NewMemoryCheckpointStore()
	logger := slog.Default()

	_, err := projection.NewRunner(
		store, bus, checkpoint,
		projection.WithLogger(logger),
	)
	if err != nil {
		t.Fatalf("NewRunner with logger: %v", err)
	}
}

package projection_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/memory/v2"
	"github.com/larsartmann/go-cqrs-lite/projection/v2"
)

func TestNewRunner_NilBus(t *testing.T) {
	t.Parallel()

	_, err := projection.NewRunner(nil, nil, memory.NewMemoryCheckpointStore())
	if err == nil {
		t.Fatal("expected error for nil bus")
	}
}

func TestNewRunner_NilCheckpoint(t *testing.T) {
	t.Parallel()

	_, err := projection.NewRunner(nil, memory.NewMemoryBus(), nil)
	if err == nil {
		t.Fatal("expected error for nil checkpoint")
	}
}

func TestNewRunner_NilLoaderIsOK(t *testing.T) {
	t.Parallel()

	_, err := projection.NewRunner(nil, memory.NewMemoryBus(), memory.NewMemoryCheckpointStore())
	if err != nil {
		t.Fatalf("nil loader should be ok: %v", err)
	}
}

func TestRunner_Register_NilProjection(t *testing.T) {
	t.Parallel()

	runner := newTestRunner(t)

	err := runner.Register(nil)
	if err == nil {
		t.Fatal("expected error for nil projection")
	}
}

func testProjection() event.Projection {
	return event.NewProjection("test", func(_ context.Context, _ event.Event) error {
		return nil
	}, []event.Type{"UserCreated"})
}

func TestRunner_Register_ValidProjection(t *testing.T) {
	t.Parallel()

	runner := newTestRunner(t)

	err := runner.Register(testProjection())
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
}

func TestRunner_Register_Duplicate(t *testing.T) {
	t.Parallel()

	runner := newTestRunner(t)

	proj := testProjection()

	if err := runner.Register(proj); err != nil {
		t.Fatalf("first Register: %v", err)
	}

	err := runner.Register(proj)
	if err == nil {
		t.Fatal("expected error for duplicate projection registration")
	}
}

func TestRunner_NoProjections(t *testing.T) {
	t.Parallel()

	runner := newTestRunner(t)

	err := runner.Run(context.Background())
	if err == nil {
		t.Fatal("expected error when no projections registered")
	}
}

func TestHandlerRegistry_OnAll_NilHandler(t *testing.T) {
	t.Parallel()

	registry := projection.NewHandlerRegistry()

	err := registry.OnAll(nil)
	if err == nil {
		t.Fatal("expected error for nil handler")
	}
}

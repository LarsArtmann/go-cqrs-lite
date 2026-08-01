package stack_test

import (
	"testing"

	flightrecorder "github.com/larsartmann/go-cqrs-lite/flightrecorder/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	memory "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

func TestWithFlightRecorder(t *testing.T) {
	t.Parallel()

	recorder, err := flightrecorder.New()
	if err != nil {
		t.Fatalf("flightrecorder.New: %v", err)
	}

	memStore := memory.NewMemoryStore()

	bundle, err := stack.New(
		stack.WithEventStore(memStore),
		stack.WithFlightRecorder(recorder),
	)
	if err != nil {
		t.Fatalf("stack.New: %v", err)
	}

	if bundle.FlightRecorder() == nil {
		t.Fatal("FlightRecorder() returned nil, want non-nil")
	}

	if bundle.FlightRecorder() != recorder {
		t.Fatal("FlightRecorder() returned a different pointer than what was passed to WithFlightRecorder")
	}

	if err := bundle.Close(); err != nil {
		t.Fatalf("bundle.Close: %v", err)
	}

	// After Close, the recorder should be stopped (not enabled).
	if recorder.Enabled() {
		t.Fatal("recorder should be stopped after bundle.Close")
	}
}

func TestFlightRecorder_NilWhenNotSet(t *testing.T) {
	t.Parallel()

	memStore := memory.NewMemoryStore()

	bundle, err := stack.New(
		stack.WithEventStore(memStore),
	)
	if err != nil {
		t.Fatalf("stack.New: %v", err)
	}
	defer bundle.Close()

	if bundle.FlightRecorder() != nil {
		t.Fatal("FlightRecorder() should return nil when not set")
	}
}

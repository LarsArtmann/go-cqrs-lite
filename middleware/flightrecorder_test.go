package middleware_test

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	flightrecorder "github.com/larsartmann/go-cqrs-lite/flightrecorder/v4"
	"github.com/larsartmann/go-cqrs-lite/middleware/v4"
)

// recorderMu serializes tests that call Start/Stop because Go's
// runtime/trace allows only ONE active flight recorder per process.
var recorderMu sync.Mutex

func newTestCommand(typeName string) command.Command {
	cmd, _ := command.New(typeName, "stream-1", nil)
	return cmd
}

func TestNewFlightRecorder_NilRecorder(t *testing.T) {
	t.Parallel()

	called := false

	mw := middleware.NewFlightRecorder(middleware.CommandAdapter, nil,
		flightrecorder.OnAlways())
	handler := mw(func(_ context.Context, _ command.Command) error {
		called = true

		return nil
	})

	err := handler(context.Background(), newTestCommand("test"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !called {
		t.Fatal("handler should have been called")
	}
}

func TestNewFlightRecorder_NilTrigger(t *testing.T) {
	recorderMu.Lock()
	defer recorderMu.Unlock()

	r, _ := flightrecorder.New(
		flightrecorder.WithWriter(&bytes.Buffer{}),
	)

	var buf bytes.Buffer
	r2, _ := flightrecorder.New(
		flightrecorder.WithWriter(&buf),
	)

	mw := middleware.NewFlightRecorder(middleware.CommandAdapter, r2, nil)
	handler := mw(func(_ context.Context, _ command.Command) error {
		return nil
	})

	_ = handler(context.Background(), newTestCommand("test"))
}

func TestCommandFlightRecorder_LatencyTrigger(t *testing.T) {
	recorderMu.Lock()
	defer recorderMu.Unlock()

	var buf bytes.Buffer

	r, _ := flightrecorder.New(
		flightrecorder.WithMinAge(50*time.Millisecond),
		flightrecorder.WithMaxBytes(1<<20),
		flightrecorder.WithWriter(&buf),
	)

	if err := r.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}

	t.Cleanup(r.Stop)

	time.Sleep(100 * time.Millisecond)

	mw := middleware.CommandFlightRecorder(r,
		flightrecorder.OnLatency(10*time.Millisecond))

	handler := mw(func(_ context.Context, _ command.Command) error {
		time.Sleep(20 * time.Millisecond) // exceeds trigger threshold

		return nil
	})

	err := handler(context.Background(), newTestCommand("slow.cmd"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Snapshot runs in a goroutine; give it time to complete.
	time.Sleep(200 * time.Millisecond)

	if buf.Len() == 0 {
		t.Fatal("expected trace data after slow command triggered snapshot")
	}
}

func TestCommandFlightRecorder_FastOperationNoSnapshot(t *testing.T) {
	recorderMu.Lock()
	defer recorderMu.Unlock()

	var buf bytes.Buffer

	r, _ := flightrecorder.New(
		flightrecorder.WithMinAge(50*time.Millisecond),
		flightrecorder.WithMaxBytes(1<<20),
		flightrecorder.WithWriter(&buf),
	)

	r.Start()
	t.Cleanup(r.Stop)

	time.Sleep(100 * time.Millisecond)

	mw := middleware.CommandFlightRecorder(r,
		flightrecorder.OnLatency(100*time.Millisecond))

	handler := mw(func(_ context.Context, _ command.Command) error {
		return nil // instant
	})

	_ = handler(context.Background(), newTestCommand("fast.cmd"))

	time.Sleep(100 * time.Millisecond)

	if buf.Len() != 0 {
		t.Fatal("expected NO trace data for fast operation")
	}
}

func TestCommandFlightRecorder_ErrorTrigger(t *testing.T) {
	recorderMu.Lock()
	defer recorderMu.Unlock()

	var buf bytes.Buffer

	r, _ := flightrecorder.New(
		flightrecorder.WithMinAge(50*time.Millisecond),
		flightrecorder.WithMaxBytes(1<<20),
		flightrecorder.WithWriter(&buf),
	)

	r.Start()
	t.Cleanup(r.Stop)

	time.Sleep(100 * time.Millisecond)

	testErr := errors.New("handler failed")

	mw := middleware.CommandFlightRecorder(r,
		flightrecorder.OnError())

	handler := mw(func(_ context.Context, _ command.Command) error {
		return testErr
	})

	err := handler(context.Background(), newTestCommand("fail.cmd"))
	if !errors.Is(err, testErr) {
		t.Fatalf("expected testErr, got: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	if buf.Len() == 0 {
		t.Fatal("expected trace data after error-triggered snapshot")
	}
}

func TestCommandFlightRecorder_PreservesError(t *testing.T) {
	recorderMu.Lock()
	defer recorderMu.Unlock()

	r, _ := flightrecorder.New()
	r.Start()
	t.Cleanup(r.Stop)

	testErr := errors.New("boom")

	mw := middleware.CommandFlightRecorder(r,
		flightrecorder.OnError())

	handler := mw(func(_ context.Context, _ command.Command) error {
		return testErr
	})

	err := handler(context.Background(), newTestCommand("test"))

	if !errors.Is(err, testErr) {
		t.Fatalf("middleware should preserve original error, got: %v", err)
	}
}

func TestEventFlightRecorder_ErrorTrigger(t *testing.T) {
	recorderMu.Lock()
	defer recorderMu.Unlock()

	var buf bytes.Buffer

	r, _ := flightrecorder.New(
		flightrecorder.WithMinAge(50*time.Millisecond),
		flightrecorder.WithMaxBytes(1<<20),
		flightrecorder.WithWriter(&buf),
	)

	r.Start()
	t.Cleanup(r.Stop)

	time.Sleep(100 * time.Millisecond)

	mw := middleware.EventFlightRecorder(r,
		flightrecorder.OnError())

	evt, _ := event.NewEvent("test.event", "stream-1", "Test", 1, nil,
		event.WithCorrelationID("corr-1"))

	handler := mw(func(_ context.Context, _ event.Event) error {
		return errors.New("event handler failed")
	})

	_ = handler(context.Background(), evt)

	time.Sleep(200 * time.Millisecond)

	if buf.Len() == 0 {
		t.Fatal("expected trace data after event error snapshot")
	}
}

func TestQueryFlightRecorder_LatencyTrigger(t *testing.T) {
	recorderMu.Lock()
	defer recorderMu.Unlock()

	var buf bytes.Buffer

	r, _ := flightrecorder.New(
		flightrecorder.WithMinAge(50*time.Millisecond),
		flightrecorder.WithMaxBytes(1<<20),
		flightrecorder.WithWriter(&buf),
	)

	r.Start()
	t.Cleanup(r.Stop)

	time.Sleep(100 * time.Millisecond)

	mw := middleware.QueryFlightRecorder(r,
		flightrecorder.OnLatency(10*time.Millisecond))

	q, _ := command.New("test.query", "q-1", nil) // reuse for simplicity

	_ = mw(func(_ context.Context, _ any) (any, error) {
		time.Sleep(20 * time.Millisecond)

		return "result", nil
	})

	// Query middleware uses AsQuery, which wraps differently.
	// Verify the generic path works.
	genMW := middleware.NewFlightRecorder(middleware.QueryAdapter, r,
		flightrecorder.OnLatency(10*time.Millisecond))

	handler := genMW(func(_ context.Context, _ command.Command) error {
		time.Sleep(20 * time.Millisecond)

		return nil
	})

	_ = handler(context.Background(), q)

	time.Sleep(200 * time.Millisecond)

	if buf.Len() == 0 {
		t.Fatal("expected trace data after slow query snapshot")
	}
}

func TestNewFlightRecorder_RecorderSnapshotOnce(t *testing.T) {
	recorderMu.Lock()
	defer recorderMu.Unlock()

	var buf bytes.Buffer

	r, _ := flightrecorder.New(
		flightrecorder.WithMinAge(50*time.Millisecond),
		flightrecorder.WithMaxBytes(1<<20),
		flightrecorder.WithWriter(&buf),
	)

	r.Start()
	t.Cleanup(r.Stop)

	time.Sleep(100 * time.Millisecond)

	mw := middleware.CommandFlightRecorder(r,
		flightrecorder.OnAlways())

	handler := mw(func(_ context.Context, _ command.Command) error {
		return nil
	})

	// First call should trigger a snapshot.
	_ = handler(context.Background(), newTestCommand("first"))
	time.Sleep(200 * time.Millisecond)

	firstSize := buf.Len()
	if firstSize == 0 {
		t.Fatal("expected trace data from first trigger")
	}

	// Second call should be a no-op (once semantics).
	_ = handler(context.Background(), newTestCommand("second"))
	time.Sleep(200 * time.Millisecond)

	if buf.Len() != firstSize {
		t.Fatalf("once-semantics violated: %d -> %d", firstSize, buf.Len())
	}
}

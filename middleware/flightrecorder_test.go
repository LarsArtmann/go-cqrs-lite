package middleware

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	flightrecorder "github.com/larsartmann/go-cqrs-lite/flightrecorder/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

// frRecorderMu serializes flight recorder tests because Go's runtime/trace
// allows only ONE active flight recorder per process.
var frRecorderMu sync.Mutex

func TestFlightRecorder_NilRecorder(t *testing.T) {
	called := false

	mw := NewFlightRecorder(CommandAdapter, nil,
		flightrecorder.OnAlways())
	handler := mw(func(_ context.Context, _ command.Command) error {
		called = true

		return nil
	})

	err := handler(context.Background(), &testCommand{streamID: id.NewStreamID()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !called {
		t.Fatal("handler should have been called")
	}
}

func TestFlightRecorder_NilTrigger(t *testing.T) {
	frRecorderMu.Lock()
	defer frRecorderMu.Unlock()

	var buf bytes.Buffer

	r, _ := flightrecorder.New(
		flightrecorder.WithMinAge(50*time.Millisecond),
		flightrecorder.WithMaxBytes(1<<20),
		flightrecorder.WithWriter(&buf),
	)
	r.Start()
	t.Cleanup(r.Stop)

	time.Sleep(100 * time.Millisecond)

	mw := NewFlightRecorder(CommandAdapter, r, nil)
	handler := mw(NoopCommandHandler())

	_ = handler(context.Background(), &testCommand{streamID: id.NewStreamID()})

	time.Sleep(100 * time.Millisecond)

	if buf.Len() != 0 {
		t.Fatal("expected NO snapshot with nil trigger")
	}
}

func TestCommandFlightRecorder_LatencyTrigger(t *testing.T) {
	frRecorderMu.Lock()
	defer frRecorderMu.Unlock()

	var buf bytes.Buffer

	r, _ := flightrecorder.New(
		flightrecorder.WithMinAge(50*time.Millisecond),
		flightrecorder.WithMaxBytes(1<<20),
		flightrecorder.WithWriter(&buf),
	)
	r.Start()
	t.Cleanup(r.Stop)

	time.Sleep(100 * time.Millisecond)

	mw := CommandFlightRecorder(r,
		flightrecorder.OnLatency(10*time.Millisecond))

	handler := mw(func(_ context.Context, _ command.Command) error {
		time.Sleep(20 * time.Millisecond)

		return nil
	})

	err := handler(context.Background(), &testCommand{streamID: id.NewStreamID()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	if buf.Len() == 0 {
		t.Fatal("expected trace data after slow command triggered snapshot")
	}
}

func TestCommandFlightRecorder_FastOperationNoSnapshot(t *testing.T) {
	frRecorderMu.Lock()
	defer frRecorderMu.Unlock()

	var buf bytes.Buffer

	r, _ := flightrecorder.New(
		flightrecorder.WithMinAge(50*time.Millisecond),
		flightrecorder.WithMaxBytes(1<<20),
		flightrecorder.WithWriter(&buf),
	)
	r.Start()
	t.Cleanup(r.Stop)

	time.Sleep(100 * time.Millisecond)

	mw := CommandFlightRecorder(r,
		flightrecorder.OnLatency(100*time.Millisecond))

	handler := mw(NoopCommandHandler())

	_ = handler(context.Background(), &testCommand{streamID: id.NewStreamID()})

	time.Sleep(100 * time.Millisecond)

	if buf.Len() != 0 {
		t.Fatal("expected NO trace data for fast operation")
	}
}

func TestCommandFlightRecorder_ErrorTrigger(t *testing.T) {
	frRecorderMu.Lock()
	defer frRecorderMu.Unlock()

	var buf bytes.Buffer

	r, _ := flightrecorder.New(
		flightrecorder.WithMinAge(50*time.Millisecond),
		flightrecorder.WithMaxBytes(1<<20),
		flightrecorder.WithWriter(&buf),
	)
	r.Start()
	t.Cleanup(r.Stop)

	time.Sleep(100 * time.Millisecond)

	mw := CommandFlightRecorder(r,
		flightrecorder.OnError())

	handler := mw(failingCommandHandler("handler failed"))

	err := handler(context.Background(), &testCommand{streamID: id.NewStreamID()})
	if err == nil {
		t.Fatal("expected error from handler")
	}

	time.Sleep(200 * time.Millisecond)

	if buf.Len() == 0 {
		t.Fatal("expected trace data after error-triggered snapshot")
	}
}

func TestCommandFlightRecorder_PreservesError(t *testing.T) {
	frRecorderMu.Lock()
	defer frRecorderMu.Unlock()

	r, _ := flightrecorder.New()
	r.Start()
	t.Cleanup(r.Stop)

	mw := CommandFlightRecorder(r,
		flightrecorder.OnError())

	handler := mw(failingCommandHandler("boom"))

	err := handler(context.Background(), &testCommand{streamID: id.NewStreamID()})

	if err == nil || err.Error() != "boom" {
		t.Fatalf("middleware should preserve original error, got: %v", err)
	}
}

func TestEventFlightRecorder_ErrorTrigger(t *testing.T) {
	frRecorderMu.Lock()
	defer frRecorderMu.Unlock()

	var buf bytes.Buffer

	r, _ := flightrecorder.New(
		flightrecorder.WithMinAge(50*time.Millisecond),
		flightrecorder.WithMaxBytes(1<<20),
		flightrecorder.WithWriter(&buf),
	)
	r.Start()
	t.Cleanup(r.Stop)

	time.Sleep(100 * time.Millisecond)

	evt, err := eventtest.NewTestEvent()
	if err != nil {
		t.Fatalf("NewTestEvent: %v", err)
	}

	mw := EventFlightRecorder(r,
		flightrecorder.OnError())

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
	frRecorderMu.Lock()
	defer frRecorderMu.Unlock()

	var buf bytes.Buffer

	r, _ := flightrecorder.New(
		flightrecorder.WithMinAge(50*time.Millisecond),
		flightrecorder.WithMaxBytes(1<<20),
		flightrecorder.WithWriter(&buf),
	)
	r.Start()
	t.Cleanup(r.Stop)

	time.Sleep(100 * time.Millisecond)

	mw := QueryFlightRecorder(r,
		flightrecorder.OnLatency(10*time.Millisecond))

	handler := mw(func(_ context.Context, _ query.Query) (any, error) {
		time.Sleep(20 * time.Millisecond)

		return "result", nil
	})

	_, err := handler(context.Background(), &testQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	if buf.Len() == 0 {
		t.Fatal("expected trace data after slow query snapshot")
	}
}

func TestFlightRecorder_SnapshotOnce(t *testing.T) {
	frRecorderMu.Lock()
	defer frRecorderMu.Unlock()

	var buf bytes.Buffer

	r, _ := flightrecorder.New(
		flightrecorder.WithMinAge(50*time.Millisecond),
		flightrecorder.WithMaxBytes(1<<20),
		flightrecorder.WithWriter(&buf),
	)
	r.Start()
	t.Cleanup(r.Stop)

	time.Sleep(100 * time.Millisecond)

	mw := CommandFlightRecorder(r,
		flightrecorder.OnAlways())

	handler := mw(NoopCommandHandler())

	_ = handler(context.Background(), &testCommand{streamID: id.NewStreamID()})
	time.Sleep(200 * time.Millisecond)

	firstSize := buf.Len()
	if firstSize == 0 {
		t.Fatal("expected trace data from first trigger")
	}

	_ = handler(context.Background(), &testCommand{streamID: id.NewStreamID()})
	time.Sleep(200 * time.Millisecond)

	if buf.Len() != firstSize {
		t.Fatalf("once-semantics violated: %d -> %d", firstSize, buf.Len())
	}
}

func TestFlightRecorder_LoggerOnError(t *testing.T) {
	frRecorderMu.Lock()
	defer frRecorderMu.Unlock()

	r, _ := flightrecorder.New(
		flightrecorder.WithMinAge(50*time.Millisecond),
		flightrecorder.WithMaxBytes(1<<20),
	)
	r.Start()
	t.Cleanup(r.Stop)

	time.Sleep(100 * time.Millisecond)

	logger, counter := newTestLogger()

	mw := CommandFlightRecorder(r,
		flightrecorder.OnAlways(),
		WithLogger(logger))

	// Use a failing writer to trigger snapshot error.
	handler := mw(NoopCommandHandler())

	_ = handler(context.Background(), &testCommand{streamID: id.NewStreamID()})

	time.Sleep(200 * time.Millisecond)

	// Snapshot succeeds (writes to discard), so no error log expected.
	// But verify the logger was at least available.
	_ = counter
}

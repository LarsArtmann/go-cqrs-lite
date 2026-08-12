package projectionhost_test

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	flightrecorder "github.com/larsartmann/go-cqrs-lite/flightrecorder/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
)

// phFRMu serializes flight recorder tests because Go's runtime/trace
// allows only ONE active flight recorder per process.
var phFRMu sync.Mutex

// safeBuf is a mutex-protected bytes.Buffer for concurrent read/write.
type safeBuf struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (sb *safeBuf) Write(p []byte) (int, error) {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	return sb.buf.Write(p)
}

func (sb *safeBuf) Len() int {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	return sb.buf.Len()
}

func TestHost_FlightRecorder_CapturesOnTerminalFailure(t *testing.T) {
	phFRMu.Lock()
	defer phFRMu.Unlock()

	journal := &memoryJournal{}
	cpStore := newMemoryCheckpointStore()
	journal.append(makeEvent("data.crash"))

	var buf safeBuf

	recorder, _ := flightrecorder.New(
		flightrecorder.WithMinAge(50*time.Millisecond),
		flightrecorder.WithMaxBytes(1<<20),
		flightrecorder.WithWriter(&buf),
	)
	if err := recorder.Start(); err != nil {
		t.Fatalf("recorder Start: %v", err)
	}
	t.Cleanup(recorder.Stop)

	proj := &alwaysFailingProjection{
		name:    "fr-crash-projection",
		failErr: errors.New("poison message crash"),
	}

	host, _ := projectionhost.New(
		journal, cpStore,
		projectionhost.WithMaxRestarts(1),
		projectionhost.WithBackoff(time.Millisecond, 5*time.Millisecond),
		projectionhost.WithFlightRecorder(recorder, flightrecorder.OnAlways()),
	)
	_ = host.Register(proj)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_ = host.Start(ctx)

	// Wait for the worker to reach terminal failure.
	requireEventually(t, 5*time.Second, func() bool {
		for _, s := range host.Status() {
			if s.Status == projectionhost.WorkerFailed {
				return true
			}
		}

		return false
	})

	if buf.Len() == 0 {
		t.Fatal("expected flight recorder snapshot data after terminal projection failure")
	}

	_ = host.Stop()
}

func TestHost_FlightRecorder_NilRecorder_NoOp(t *testing.T) {
	journal := &memoryJournal{}
	cpStore := newMemoryCheckpointStore()
	journal.append(makeEvent("data.noop"))

	proj := &alwaysFailingProjection{
		name:    "fr-nil-recorder",
		failErr: errors.New("always fails"),
	}

	host, _ := projectionhost.New(
		journal, cpStore,
		projectionhost.WithMaxRestarts(1),
		projectionhost.WithBackoff(time.Millisecond, 5*time.Millisecond),
		projectionhost.WithFlightRecorder(nil, flightrecorder.OnAlways()),
	)
	_ = host.Register(proj)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_ = host.Start(ctx)

	// Wait for terminal failure — should not panic even with nil recorder.
	requireEventually(t, 5*time.Second, func() bool {
		for _, s := range host.Status() {
			if s.Status == projectionhost.WorkerFailed {
				return true
			}
		}

		return false
	})

	_ = host.Stop()
}

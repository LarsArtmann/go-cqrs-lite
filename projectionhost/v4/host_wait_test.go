package projectionhost_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
)

// stalledJournal never hands out an event, like a live journal that has
// caught up: the worker keeps polling it until its context ends.
type stalledJournal struct {
	*memoryJournal
	reads atomic.Int64
}

func (j *stalledJournal) ReadFrom(
	ctx context.Context,
	after id.EventID,
	limit int,
) ([]event.Event, error) {
	j.reads.Add(1)
	<-ctx.Done()

	return nil, ctx.Err() //nolint:wrapcheck // test double forwarding the worker's own cancellation
}

// TestRegisterAndWait_BlocksUntilContextCancelled pins the documented
// contract of RegisterAndWait: it does not return while the projections are
// running, and cancelling the context stops the host and releases it. It
// used to return as soon as the workers were launched.
func TestRegisterAndWait_BlocksUntilContextCancelled(t *testing.T) {
	fresh, err := projectionhost.New(&stalledJournal{memoryJournal: &memoryJournal{}}, newMemoryCheckpointStore())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	logs := &bytes.Buffer{}
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(logs, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	done := make(chan error, 1)

	go func() {
		done <- projectionhost.RegisterAndWait(ctx, fresh, &countingProjection{name: "test"})
	}()

	select {
	case err := <-done:
		t.Fatalf("RegisterAndWait returned %v while the host was still running", err)
	case <-time.After(150 * time.Millisecond):
	}

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("RegisterAndWait after cancel: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("RegisterAndWait did not return after the context was cancelled")
	}

	if strings.Contains(logs.String(), "Start context was cancelled") {
		t.Errorf("cancelling the RegisterAndWait context must read as a shutdown, got log: %s", logs.String())
	}

	for _, state := range fresh.Status() {
		if state.Status == projectionhost.WorkerRunning || state.Status == projectionhost.WorkerLive {
			t.Fatalf("worker %q still %q after RegisterAndWait returned", state.Name, state.Status)
		}
	}
}

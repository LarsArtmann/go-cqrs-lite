package watermill

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	memory "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

// logCapture records every record a slog.Logger emits so tests can pin log
// levels without touching the package-level default logger.
type logCapture struct {
	mu      sync.Mutex
	records []slog.Record
}

func (c *logCapture) Handle(_ context.Context, r slog.Record) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.records = append(c.records, r.Clone())

	return nil
}

func (c *logCapture) Enabled(context.Context, slog.Level) bool { return true }

func (c *logCapture) WithAttrs([]slog.Attr) slog.Handler { return c }

func (c *logCapture) WithGroup(string) slog.Handler { return c }

// errorsContaining returns the ERROR+ records whose message contains substr.
func (c *logCapture) errorsContaining(substr string) []slog.Record {
	c.mu.Lock()
	defer c.mu.Unlock()

	var hits []slog.Record
	for _, r := range c.records {
		if r.Level >= slog.LevelError && strings.Contains(r.Message, substr) {
			hits = append(hits, r)
		}
	}

	return hits
}

// TestCatchUpSubscriber_CloseDoesNotLogReplayFailure pins the shutdown-noise
// contract: a deliberate Close() interrupting a replay parked in awaitAck must
// NOT log ERROR "catch-up replay failed". Context cancellation during shutdown
// is a Debug event, not a failure — every routine Close used to emit the
// truthful-but-noisy ERROR line.
func TestCatchUpSubscriber_CloseDoesNotLogReplayFailure(t *testing.T) {
	t.Parallel()

	store := eventtest.NewFakeStore()
	bus := eventtest.NewFakeBus()
	cpStore := memory.NewMemoryCheckpointStore()

	streamID := id.NewStreamID()

	evt, err := event.NewEvent(
		"test.closenoise", streamID, "TestStream", event.Version(1),
		[]byte(`{}`),
	)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	if err := store.AppendBatch(context.Background(),
		id.NewStreamRef("TestStream", streamID), []event.Event{evt}); err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	capture := &logCapture{}

	catchUp, err := NewCatchUpSubscriber(
		store, NewSubscriberAdapter(bus), cpStore, slog.New(capture))
	if err != nil {
		t.Fatalf("NewCatchUpSubscriber: %v", err)
	}

	ch, err := catchUp.Subscribe(context.Background(), "test.closenoise")
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	select {
	case msg := <-ch:
		// Deliberately do NOT ack: the subscriber parks in awaitAck.
		_ = msg
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for replayed event")
	}

	closePromptly(t, "Close while subscription blocked in awaitAck", catchUp.Close)

	// drainUntilClosed returns only after runCatchUp returned — by then any
	// replay-failure log line has already been emitted, so the assertion
	// below is race-free.
	_ = drainUntilClosed(t, ch)

	if hits := capture.errorsContaining("catch-up replay failed"); len(hits) > 0 {
		t.Errorf(
			"Close() logged %d ERROR \"catch-up replay failed\" record(s); "+
				"shutdown must stay at Debug level",
			len(hits),
		)
	}
}

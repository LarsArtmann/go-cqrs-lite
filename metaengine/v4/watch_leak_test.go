package metaengine

import (
	"context"
	"testing"
	"time"
)

// TestWatch_ChannelClosedOnContextCancel verifies that the channel returned
// by Watch is closed when the context is cancelled. Before the fix, the
// adapter goroutine exited without closing ch, leaving consumers blocked
// forever on a channel that would never deliver another value.
func TestWatch_ChannelClosedOnContextCancel(t *testing.T) {
	t.Parallel()

	store := newMemoryTestStore(t)

	ctx, cancel := context.WithCancel(context.Background())
	w := NewWatcher[testTask](store, "tasks")
	defer w.Close()

	ch := w.Watch(ctx, nil)

	cancel()

	// The channel must be closed within a reasonable time.
	select {
	case _, ok := <-ch:
		if ok {
			t.Error("expected channel to be closed after context cancel, but received a value")
		}
	case <-time.After(2 * time.Second):
		t.Error("channel was not closed within 2s of context cancel — goroutine leak")
	}
}

// TestWatchWithSeq_ChannelClosedOnContextCancel verifies the same for WatchWithSeq.
func TestWatchWithSeq_ChannelClosedOnContextCancel(t *testing.T) {
	t.Parallel()

	store := newMemoryTestStore(t)

	ctx, cancel := context.WithCancel(context.Background())
	w := NewWatcher[testTask](store, "tasks-seq")
	defer w.Close()

	ch := w.WatchWithSeq(ctx, nil)

	cancel()

	select {
	case _, ok := <-ch:
		if ok {
			t.Error("expected channel to be closed after context cancel, but received a value")
		}
	case <-time.After(2 * time.Second):
		t.Error("channel was not closed within 2s of context cancel — goroutine leak")
	}
}

// TestWatch_ChannelClosedOnWatcherClose verifies that the channel is closed
// when Watcher.Close() is called (which closes entry.ch, causing the adapter
// goroutine to exit).
func TestWatch_ChannelClosedOnWatcherClose(t *testing.T) {
	t.Parallel()

	store := newMemoryTestStore(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	w := NewWatcher[testTask](store, "tasks-close")

	ch := w.Watch(ctx, nil)

	w.Close()

	select {
	case _, ok := <-ch:
		if ok {
			t.Error("expected channel to be closed after Watcher.Close, but received a value")
		}
	case <-time.After(2 * time.Second):
		t.Error("channel was not closed within 2s of Watcher.Close — goroutine leak")
	}
}

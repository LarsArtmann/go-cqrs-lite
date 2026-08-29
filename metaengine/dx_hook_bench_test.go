package metaengine

import (
	"context"
	"testing"
	"time"
)

// TestWatcher_ReificationFailureHook fires when a notification cannot be
// converted to V (type drift between the fold's payload type and the
// watcher's type parameter). The fold stores testTask values; watching the
// same collection as a bare string must hit the hook, not the channel.
func TestWatcher_ReificationFailureHook(t *testing.T) {
	t.Parallel()

	store := newMemoryTestStore(t)
	defer store.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	hookCalled := make(chan any, 4)

	ch, w := WatchTyped[string](store, ctx, "tasks", nil)
	w = w.WithReificationFailureHook(func(val any) {
		hookCalled <- val
	})
	defer w.Close()

	if err := store.Apply(ctx, "task_created", testTask{ID: "t1", Title: "ok"}); err != nil {
		t.Fatalf("Apply task: %v", err)
	}

	select {
	case val := <-hookCalled:
		if _, ok := val.(testTask); !ok {
			t.Errorf("hook payload = %T, want testTask", val)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the reification failure hook")
	}

	select {
	case v := <-ch:
		t.Fatalf("incompatible value must not reach the typed channel, got %v", v)
	default:
	}
}

// BenchmarkWatcherNotificationLatency measures Apply → typed-notification
// end-to-end latency through the watcher pipeline (fold + hub + adapter).
func BenchmarkWatcherNotificationLatency(b *testing.B) {
	store := newMemoryTestStore(&testing.T{})
	defer store.Close()

	ctx := context.Background()

	ch, w := WatchTyped[testTask](store, ctx, "tasks", nil)
	defer w.Close()

	if err := store.Apply(ctx, "task_created", testTask{ID: "warm", Title: "x"}); err != nil {
		b.Fatalf("Apply warmup: %v", err)
	}

	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		b.Fatal("timed out waiting for warmup notification")
	}

	b.ResetTimer()

	for range b.N {
		if err := store.Apply(ctx, "task_created", testTask{ID: "b", Title: "x"}); err != nil {
			b.Fatalf("Apply: %v", err)
		}

		select {
		case <-ch:
		case <-time.After(5 * time.Second):
			b.Fatal("timed out waiting for notification")
		}
	}
}

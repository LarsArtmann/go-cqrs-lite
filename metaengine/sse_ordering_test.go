package metaengine

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestSubscriberHub_NotifyLatestWins pins the hub's latest-wins delivery:
// when an entry's buffer is full, notify evicts the OLDEST buffered
// notification so the newest state always gets through. Dropping the newest
// instead would leave a slow consumer stuck on stale projection state forever
// (and is inconsistent with forwardWithDropOld downstream).
func TestSubscriberHub_NotifyLatestWins(t *testing.T) {
	t.Parallel()

	h := newSubscriberHub()
	entry := &watcherEntry{ch: make(chan any, 1)}
	h.registerWatcher("tasks", entry)

	h.notify("tasks", "k1", "stale")
	h.notify("tasks", "k1", "fresh")

	select {
	case got := <-entry.ch:
		if got != "fresh" {
			t.Fatalf("expected latest value %q, got %v", "fresh", got)
		}
	default:
		t.Fatal("expected a buffered notification, got none")
	}
}

// headerProbingRecorder fires onStatus the moment the response status is
// committed — the earliest a client can observe the stream as live. Write
// is intercepted into a mutex-guarded snapshot: the SSE handler goroutine
// keeps writing the embedded recorder's Body after the test starts polling,
// and bytes.Buffer is not safe for concurrent read/write — polling
// rec.Body.String() directly is a -race failure (found by #verify run17,
// 2026-11-06), not a production defect.
type headerProbingRecorder struct {
	*httptest.ResponseRecorder

	onStatus func()

	mu  sync.Mutex
	got strings.Builder
}

func (r *headerProbingRecorder) WriteHeader(code int) {
	r.onStatus()
	r.ResponseRecorder.WriteHeader(code)
}

func (r *headerProbingRecorder) Write(p []byte) (int, error) {
	r.mu.Lock()
	r.got.Write(p)
	r.mu.Unlock()

	return r.ResponseRecorder.Write(p)
}

// snapshot returns everything streamed so far, safe to call while the
// handler is still writing.
func (r *headerProbingRecorder) snapshot() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.got.String()
}

// TestServeSSE_SubscribesBeforeHeaders pins ServeSSE's ordering guarantee:
// by the time the response status is committed, the watcher subscription is
// already registered — a store write racing the client's connect is buffered
// for the stream instead of being lost. Regression test for the systemscenario
// SSE recurrence (2026-11-05): notifications fired between the header flush
// and the handler-side registration never reached the subscriber, which
// surfaced as TestSSE_TaskViewStreamsOverHTTP receiving 0 values under load.
func TestServeSSE_SubscribesBeforeHeaders(t *testing.T) {
	t.Parallel()

	store := newMemoryTestStore(t)
	defer store.Close()

	watcher := NewWatcher[testTask](store, "tasks")
	defer watcher.Close()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	req := httptest.NewRequest(http.MethodGet, "/sse", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	probing := &headerProbingRecorder{ResponseRecorder: rec}
	probing.onStatus = func() {
		// The write happens exactly when the client first observes the
		// stream as live; if registration had not happened yet, the
		// notification would be lost and the stream would stay empty.
		if err := store.Apply(ctx, "task_created", testTask{ID: "t1", Title: "race"}); err != nil {
			t.Errorf("probe write: %v", err)
		}
	}

	done := make(chan error, 1)
	go func() { done <- ServeSSE(probing, req, watcher) }()

	deadline := time.After(5 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if strings.Contains(probing.snapshot(), "race") {
			cancel()

			return
			}
		case <-deadline:
			cancel()
			t.Fatalf("value written at header-commit time never streamed; body so far: %q", probing.snapshot())
		}
	}
}

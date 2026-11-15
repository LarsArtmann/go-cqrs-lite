package metaengine

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestForwardWithDropOld_EvictionsCounted is the throttled-consumer repro
// for the :328 recurrence class: a consumer slower than the producer must
// (a) never stall the loop, (b) always end up with the NEWEST value, and
// (c) surface every eviction in the SSEDroppedEvents counter. Before the
// counter, (c) was invisible — "the RENAME never streamed" could not be
// distinguished from other loss mechanisms.
func TestForwardWithDropOld_EvictionsCounted(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	src := make(chan string)
	buf := make(chan string, 1) // the ring: room for exactly one
	go func() {
		defer close(src)
		src <- "v1" // lands in the empty buffer
		src <- "v2" // buffer full: evicts v1
		src <- "v3" // buffer full: evicts v2
	}()

	before := SSEDroppedEvents()
	forwardWithDropOld(ctx, src, buf, nil) // runs inline: returns when src closes

	if got, want := SSEDroppedEvents()-before, int64(2); got != want {
		t.Fatalf("two evictions must be counted; counter delta = %d, want %d", got, want)
	}

	// Drop-old semantics: only the newest value survives.
	if got := <-buf; got != "v3" {
		t.Fatalf("slow consumer must end up with the newest value; got %q", got)
	}
	if _, open := <-buf; open {
		t.Fatal("buffer must hold exactly one value after three sends and no reads")
	}
}

// TestForwardWithDropOld_FilteredValuesNotCounted pins the accounting
// boundary: only EVICTIONS count, not filter drops — a filtered value is
// never buffered, so it is not backpressure.
func TestForwardWithDropOld_FilteredValuesNotCounted(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	src := make(chan int)
	buf := make(chan int, 2)
	go func() {
		defer close(src)
		src <- 1
		src <- 2
		src <- 3
	}()

	before := SSEDroppedEvents()
	forwardWithDropOld(ctx, src, buf, func(v int) bool { return v != 2 })

	if got := SSEDroppedEvents(); got != before {
		t.Fatalf("filter drops must not count as evictions; delta = %d", got-before)
	}
	if got := <-buf; got != 1 {
		t.Fatalf("first accepted value expected; got %d", got)
	}
	if got := <-buf; got != 3 {
		t.Fatalf("second accepted value expected; got %d", got)
	}
}

// TestServeSSE_FastConsumerNeverDrops asserts the counter stays flat over a
// fast-consumer SSE window — the diagnostic contract of SSEDroppedEvents:
// in the CI flake class, a NON-zero delta here (or in the systemscenario
// HTTP test) reclassifies "event never streamed" as ring backpressure.
func TestServeSSE_FastConsumerNeverDrops(t *testing.T) {
	t.Parallel()

	store := newMemoryTestStore(t)
	defer store.Close()

	watcher := NewWatcher[testTask](store, "tasks")
	defer watcher.Close()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	req := httptest.NewRequest(http.MethodGet, "/sse", nil).WithContext(ctx)
	rec := &headerProbingRecorder{ResponseRecorder: httptest.NewRecorder()}
	probing := rec
	probing.onStatus = func() {
		if err := store.Apply(ctx, "task_created", testTask{ID: "t1", Title: "drop-probe"}); err != nil {
			t.Errorf("probe write: %v", err)
		}
	}

	before := SSEDroppedEvents()

	done := make(chan error, 1)
	go func() { done <- ServeSSE(probing, req, watcher) }()

	deadline := time.After(5 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if strings.Contains(probing.snapshot(), "drop-probe") {
				cancel()

				if got := SSEDroppedEvents(); got != before {
					t.Errorf("fast consumer dropped %d event(s) — backpressure on a healthy window", got-before)
				}

				return
			}
		case <-deadline:
			cancel()
			t.Fatalf("value never streamed; body so far: %q (drops delta %d)",
				probing.snapshot(), SSEDroppedEvents()-before)
		}
	}
}

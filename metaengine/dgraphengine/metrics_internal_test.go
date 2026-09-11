package dgraphengine

import (
	"context"
	"sync"
	"testing"

	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
)

// fakeInt64Counter records Add calls for assertions.
type fakeInt64Counter struct {
	mu    sync.Mutex
	adds  []int64
}

func (f *fakeInt64Counter) Add(_ context.Context, incr int64, _ ...cqrsotel.AddOption) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.adds = append(f.adds, incr)
}



// TestContentionRetryCounter_CountsEachRetry pins the observability contract:
// every CONTENTION retry bumps cqrs.dgraph.contention_retry exactly once.
// Non-contention failures and successes never count.
func TestContentionRetryCounter_CountsEachRetry(t *testing.T) {
	eng := &dgraphEngine{}
	fake := &fakeInt64Counter{}
	eng.contentionRetry = fake

	calls := 0

	err := eng.retryOnContention(t.Context(), false, func() error {
		calls++
		if calls <= 2 {
			return errAborted //nolint:staticcheck // verbatim server message
		}

		return nil
	})
	if err != nil {
		t.Fatalf("retryOnContention: %v", err)
	}

	if len(fake.adds) != 2 {
		t.Fatalf("exactly one count per contention retry expected, got %v", fake.adds)
	}

	for _, add := range fake.adds {
		if add != 1 {
			t.Fatalf("each count must increment by 1, got %d", add)
		}
	}
}

// TestContentionRetryCounter_NonContentionNeverCounts: a failure outside the
// contention class fails fast without touching the counter.
func TestContentionRetryCounter_NonContentionNeverCounts(t *testing.T) {
	eng := &dgraphEngine{}
	fake := &fakeInt64Counter{}
	eng.contentionRetry = fake

	err := eng.retryOnContention(t.Context(), false, func() error {
		return errNonContention
	})
	if err == nil {
		t.Fatal("non-contention error must surface")
	}

	if len(fake.adds) != 0 {
		t.Fatalf("non-contention failure must not count, got %v", fake.adds)
	}
}

// TestContentionRetryCounter_NilCounterDisabled: the disabled (nil) sentinel
// must not panic — engines constructed before metrics wiring rely on it.
func TestContentionRetryCounter_NilCounterDisabled(t *testing.T) {
	eng := &dgraphEngine{} // contentionRetry nil

	if err := eng.retryOnContention(t.Context(), false, func() error {
		return errAborted //nolint:staticcheck // verbatim server message
	}); err == nil {
		t.Fatal("single-attempt contention must surface the error")
	}
}

// TestNewContentionRetryCounter_NonNilWithoutProvider: with no global meter
// provider configured (the default), construction still yields a usable
// no-op counter, never nil.
func TestNewContentionRetryCounter_NonNilWithoutProvider(t *testing.T) {
	if newContentionRetryCounter() == nil {
		t.Fatal("noop-provider construction must return the no-op counter, not nil")
	}
}

//nolint:staticcheck // verbatim server message below
var errNonContention = &staticError{"rpc error: code = PermissionDenied desc = not contention"}

type staticError struct{ msg string }

func (e *staticError) Error() string { return e.msg }

package dgraphengine

import (
	"sync"
	"testing"
)

// TestContentionObserver_CalledPerRetry pins the observability contract:
// every CONTENTION retry invokes the observer exactly once with the attempt
// number (1-based). Non-contention failures and successes never invoke it.
func TestContentionObserver_CalledPerRetry(t *testing.T) {
	var mu sync.Mutex

	var attempts []int

	eng := &dgraphEngine{contentionObserver: func(attempt int) {
		mu.Lock()
		defer mu.Unlock()

		attempts = append(attempts, attempt)
	}}

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

	if len(attempts) != 2 || attempts[0] != 1 || attempts[1] != 2 {
		t.Fatalf("observer must see attempts [1 2], got %v", attempts)
	}
}

// TestContentionObserver_NonContentionNeverFires: a failure outside the
// contention class fails fast without notifying the observer.
func TestContentionObserver_NonContentionNeverFires(t *testing.T) {
	fired := 0

	eng := &dgraphEngine{contentionObserver: func(int) { fired++ }}

	err := eng.retryOnContention(t.Context(), false, func() error {
		return errNonContention
	})
	if err == nil {
		t.Fatal("non-contention error must surface")
	}

	if fired != 0 {
		t.Fatalf("non-contention failure must not notify the observer, got %d calls", fired)
	}
}

// TestContentionObserver_NilObserverDisabled: no observer configured must
// never panic — the default construction path.
func TestContentionObserver_NilObserverDisabled(t *testing.T) {
	eng := &dgraphEngine{} // contentionObserver nil

	if err := eng.retryOnContention(t.Context(), false, func() error {
		return errAborted //nolint:staticcheck // verbatim server message
	}); err == nil {
		t.Fatal("single-attempt contention must surface the error")
	}
}

// TestWithContentionObserver_WiresEngine: the public Option must land on the
// engine struct.
func TestWithContentionObserver_WiresEngine(t *testing.T) {
	eng := &dgraphEngine{}

	WithContentionObserver(func(int) {})(eng)

	if eng.contentionObserver == nil {
		t.Fatal("WithContentionObserver must set the observer")
	}
}

//nolint:staticcheck // verbatim server message below
var errNonContention = &staticError{"rpc error: code = PermissionDenied desc = not contention"}

type staticError struct{ msg string }

func (e *staticError) Error() string { return e.msg }

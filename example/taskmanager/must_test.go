package main

import (
	"errors"
	"testing"
)

// TestMust_ReturnsValueOnErrorNil pins the happy path: Must forwards the
// wrapped value untouched when err is nil.
func TestMust_ReturnsValueOnErrorNil(t *testing.T) {
	t.Parallel()

	got := Must(42, nil)
	if got != 42 {
		t.Fatalf("Must(42, nil) = %d, want 42", got)
	}
}

// TestMust_PanicsOnError pins the panic path: a non-nil err must panic —
// Must exists for unrecoverable initialization errors only.
func TestMust_PanicsOnError(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("boom")

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("Must with non-nil error did not panic")
		}

		err, ok := r.(error)
		if !ok || !errors.Is(err, sentinel) {
			t.Fatalf("panic value = %v, want the wrapped error", r)
		}
	}()

	_ = Must("unused", sentinel)
}

// TestCheck_NoPanicOnErrorNil pins the fire-and-forget happy path.
func TestCheck_NoPanicOnErrorNil(t *testing.T) {
	t.Parallel()

	Check(nil)
}

// TestCheck_PanicsOnError pins that Check panics with the original error.
func TestCheck_PanicsOnError(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("boom")

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("Check with non-nil error did not panic")
		}

		err, ok := r.(error)
		if !ok || !errors.Is(err, sentinel) {
			t.Fatalf("panic value = %v, want the wrapped error", r)
		}
	}()

	Check(sentinel)
}

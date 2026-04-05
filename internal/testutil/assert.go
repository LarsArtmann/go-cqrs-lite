package testutil

import (
	"slices"
	"testing"
)

// AssertCallOrder verifies that the got slice matches the expected order.
func AssertCallOrder(t testing.TB, got, want []string) {
	t.Helper()

	if !slices.Equal(got, want) {
		t.Errorf("expected call order %v, got %v", want, got)
	}
}

// AssertNoError fails if err is not nil.
func AssertNoError(t testing.TB, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// AssertError fails if err is nil.
func AssertError(t testing.TB, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// AssertTrue fails if condition is false.
func AssertTrue(t testing.TB, condition bool, msg string) {
	t.Helper()

	if !condition {
		t.Error(msg)
	}
}

// AssertFalse fails if condition is true.
func AssertFalse(t testing.TB, condition bool, msg string) {
	t.Helper()

	if condition {
		t.Error(msg)
	}
}

// AssertEqual fails if got != want.
func AssertEqual[T comparable](t testing.TB, got, want T) {
	t.Helper()

	if got != want {
		t.Errorf("expected %v, got %v", want, got)
	}
}

// AssertNotNil fails if v is nil.
func AssertNotNil[T any](t testing.TB, v *T) {
	t.Helper()

	if v == nil {
		t.Fatal("expected non-nil, got nil")
	}
}

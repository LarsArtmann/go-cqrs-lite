package pebble

import (
	"errors"
	"fmt"
	"testing"

	"github.com/cockroachdb/pebble"
	errorfamily "github.com/larsartmann/go-error-family"
)

func TestPebbleErrNotFoundClassifiedAsRejection(t *testing.T) {
	t.Parallel()

	family := errorfamily.Classify(pebble.ErrNotFound)
	if family != errorfamily.Rejection {
		t.Fatalf("expected pebble.ErrNotFound to classify as Rejection, got %s", family)
	}

	wrapped := fmt.Errorf("wrapped: %w", pebble.ErrNotFound)
	family = errorfamily.Classify(wrapped)
	if family != errorfamily.Rejection {
		t.Fatalf("expected wrapped pebble.ErrNotFound to classify as Rejection, got %s", family)
	}

	if !errors.Is(wrapped, pebble.ErrNotFound) {
		t.Fatal("wrapped error should match pebble.ErrNotFound via errors.Is")
	}
}

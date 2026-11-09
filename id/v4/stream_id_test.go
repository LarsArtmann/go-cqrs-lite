package id_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// TestStreamIDFrom_NilIsZero pins the zero-ID contract's nil clause: the
// untyped nil literal returns the zero StreamID instead of panicking inside
// String() (audit finding 2026-11-09 — the documented contract predated the
// guard).
func TestStreamIDFrom_NilIsZero(t *testing.T) {
	got := id.StreamIDFrom(nil)
	if !got.IsZero() {
		t.Fatalf("StreamIDFrom(nil) must be the zero StreamID, got %q", got.Get())
	}
}

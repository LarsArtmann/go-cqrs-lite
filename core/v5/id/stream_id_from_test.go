package id_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/v5/id"
)

// TestStreamIDFrom_NilIsZero pins the zero-ID contract's nil clause (the
// v4 twin carries the same contract — keep both in lockstep).
func TestStreamIDFrom_NilIsZero(t *testing.T) {
	got := id.StreamIDFrom(nil)
	if !got.IsZero() {
		t.Fatalf("StreamIDFrom(nil) must be the zero StreamID, got %q", got.Get())
	}
}

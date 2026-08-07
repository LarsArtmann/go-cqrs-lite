package bench_test

import (
	"testing"

	pebbleengine "github.com/larsartmann/go-cqrs-lite/metaengine/pebbleengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// newPebbleEngine creates an in-memory Pebble engine for benchmarking.
// Pebble is pure Go (no CGo required). The caller is responsible for closing
// the store (which closes the engine) — Pebble panics on double-close.
func newPebbleEngine(tb testing.TB) metaengine.Engine {
	tb.Helper()

	eng, err := pebbleengine.NewPebbleEngine("")
	if err != nil {
		tb.Fatalf("PebbleEngine: %v", err)
	}

	return eng
}

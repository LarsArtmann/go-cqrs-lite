package tursoengine_test

import (
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestTurso_NoCounterBackend pins the capability boundary the tq row-683
// sweep discovered: the Turso engine (a sqlite delegate) exposes no
// CounterBackend, so the composite counter-key class is structurally out of
// reach — conform scenarios skip rather than exercise a phantom counter
// store.
func TestTurso_NoCounterBackend(t *testing.T) {
	t.Parallel()

	eng := mustNewTursoEngine(t)

	if _, ok := eng.(metaengine.CounterBackend); ok {
		t.Fatal("turso engine unexpectedly implements CounterBackend — re-run the conform counter-key scenario on it")
	}
}

package tursoengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/enginetest"
)

// TestTurso_CounterKeyComposite runs the conform counter-key scenario on the
// Turso engine: it delegates to [sqliteengine.NewSQLiteEngine], which
// implements [metaengine.CounterBackend] unconditionally, so the engine
// inherits the capability (and the class) from its delegate — the composite
// counter-key contract must hold here like on every other CounterBackend
// engine. (An earlier pin asserted the opposite capability boundary; the
// delegate type makes a no-CounterBackend state unrepresentable.)
func TestTurso_CounterKeyComposite(t *testing.T) {
	t.Parallel()

	eng := mustNewTursoEngine(t)

	enginetest.RunCounterKeyCompositeTest(t, eng)
}

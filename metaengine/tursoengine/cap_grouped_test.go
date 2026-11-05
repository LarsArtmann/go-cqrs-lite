package tursoengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
)

// TestTursoGroupedCap runs the per-group cap contract (ADR-0164, #114) over
// the turso engine: turso delegates to sqliteengine, so the window-function
// pushdown runs through the delegation and the observable contract is the
// sqlite one.
func TestTursoGroupedCap(t *testing.T) {
	t.Parallel()

	adttest.AssertGroupedCapContract(t, mustNewTursoEngine(t))
}

package dgraphengine_test

import (
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
)

// adt_matrix_test.go runs the full ADT test matrix across the Dgraph engine
// and the memory engine, asserting cross-engine parity. The harness
// auto-skips ADTs whose backend interface the engine does not implement.
//
// Dgraph's native strengths (Graph, Search) are not degraded — they run at
// full parity with the memory engine.

func TestDgraphADTMatrix(t *testing.T) {
	t.Parallel()

	adttest.RunMatrix(t, []adttest.Factory{
		{
			Name:   "memory",
			Create: func(t *testing.T) metaengine.Engine { return metaengine.NewMemoryEngine() },
		},
		{
			Name: "dgraph",
			Create: func(t *testing.T) metaengine.Engine {
				return newDgraphEngineOrSkip(t)
			},
		},
	})
}

package dgraphengine_test

import (
	"os"
	"testing"

	dgraphengine "github.com/larsartmann/go-cqrs-lite/metaengine/dgraphengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
)

// dgraphAddr returns the Dgraph gRPC address from DGRAPH_ADDR or defaults
// to localhost:9080.
func dgraphAddr() string {
	if addr := os.Getenv("DGRAPH_ADDR"); addr != "" {
		return addr
	}

	return "localhost:9080"
}

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
				t.Helper()

				eng, err := dgraphengine.New(dgraphAddr())
				if err != nil {
					t.Skipf("Dgraph not available: %v", err)
				}

				return eng
			},
		},
	})
}

package pgengine_test

import (
	"testing"

	pgengine "github.com/larsartmann/go-cqrs-lite/metaengine/pgengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
)

// adt_matrix_test.go runs the cross-engine ADT test matrix across the
// Postgres engine and the memory engine, asserting parity on every ADT
// that pgengine implements (Map, Counter, SortedMap). The harness
// auto-skips ADTs whose backend interface pgengine does not implement.

func TestPostgresADTMatrix(t *testing.T) {
	t.Parallel()

	adttest.RunMatrix(t, []adttest.Factory{
		{
			Name:   "memory",
			Create: func(t *testing.T) metaengine.Engine { return metaengine.NewMemoryEngine() },
		},
		{
			Name: "postgres",
			Create: func(t *testing.T) metaengine.Engine {
				t.Helper()

				eng, err := pgengine.New(pgDSN(t))
				if err != nil {
					t.Skipf("Postgres not available: %v", err)
				}

				return eng
			},
		},
	})
}

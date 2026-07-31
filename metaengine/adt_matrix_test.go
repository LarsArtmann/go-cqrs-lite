package metaengine_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
)

// adt_matrix_test.go delegates to the shared adttest harness so that
// external engine modules (e.g. pebbleengine) can run the same matrix
// with their engine included for cross-engine parity.
//
// To add a new engine, add a Factory here AND in the engine module's own
// adt_matrix_test.go (with all engines included for parity).

func TestADTMatrix(t *testing.T) {
	t.Parallel()

	adttest.RunMatrix(t, []adttest.Factory{
		{
			Name:   "memory",
			Create: func(t *testing.T) metaengine.Engine { return metaengine.NewMemoryEngine() },
		},
		{
			Name:   "sqlite",
			Create: func(t *testing.T) metaengine.Engine { return newIsolatedSQLiteEngine(t) },
		},
	})
}

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

// TestCapabilityConformance runs the declared-vs-implemented capability
// table (structural check) for the in-repo engines. Engine modules run the
// same check for their own engine in their adt_matrix_test.go.
func TestCapabilityConformance(t *testing.T) {
	t.Parallel()

	t.Run("memory", func(t *testing.T) {
		t.Parallel()

		adttest.RunCapabilityConformance(t, "memory", metaengine.NewMemoryEngine(), nil)
	})

	t.Run("sqlite", func(t *testing.T) {
		t.Parallel()

		adttest.RunCapabilityConformance(t, "sqlite", newIsolatedSQLiteEngine(t), nil)
	})
}

// TestPaginationConformance runs the compound-cursor MapScan pagination walk
// (M06/F22): tie-heavy pages must cover every row exactly once with monotone
// order and honest HasMore. Probes rebuild each engine's externally observable
// tiebreak key form — the raw map key for memory and sqlite (sqlite's MapScan
// selects the key column since the F22 conformance fix; it previously
// tiebreaked on stored value bytes, which json v2's non-canonical map key
// order made externally underivable). Engine modules run the same walk for
// their engines in their adt_matrix_test.go.
func TestPaginationConformance(t *testing.T) {
	t.Parallel()

	adttest.RunPaginationConformance(t, []adttest.PaginationProbe{
		{
			Name:      "memory",
			Create:    func(t *testing.T) metaengine.Engine { return metaengine.NewMemoryEngine() },
			CursorKey: adttest.CursorKeyRaw,
		},
		{
			Name:      "sqlite",
			Create:    func(t *testing.T) metaengine.Engine { return newIsolatedSQLiteEngine(t) },
			CursorKey: adttest.CursorKeyRaw,
		},
	})
}

func TestLayoutMatrix(t *testing.T) {
	t.Parallel()

	adttest.RunLayoutMatrix(t, []adttest.Factory{
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

func TestLayoutConflict(t *testing.T) {
	t.Parallel()

	adttest.RunLayoutConflictTest(t, []adttest.Factory{
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

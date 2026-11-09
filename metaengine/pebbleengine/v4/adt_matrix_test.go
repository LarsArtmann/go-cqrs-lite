package pebbleengine_test

import (
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
)

// adt_matrix_test.go runs the full 7-ADT test matrix across the Pebble
// engine and the memory engine, asserting cross-engine parity. The
// metaengine module's own adt_matrix_test.go covers memory↔sqlite parity;
// this test covers memory↔pebble parity. By transitivity, all three
// engines produce identical results.

func TestPebbleADTMatrix(t *testing.T) {
	t.Parallel()

	adttest.RunMatrix(t, []adttest.Factory{
		{
			Name:   "memory",
			Create: func(t *testing.T) metaengine.Engine { return metaengine.NewMemoryEngine() },
		},
		{
			Name: "pebble",
			Create: func(t *testing.T) metaengine.Engine {
				return newPebbleEngineOrSkip(t)
			},
		},
	})
}

// TestCapabilityConformance verifies this engine's Profile() declarations
// against its implemented backend interfaces (declared-vs-implemented table).
func TestCapabilityConformance(t *testing.T) {
	t.Parallel()

	adttest.RunCapabilityConformance(t, "pebble", newPebbleEngineOrSkip(t), nil)
}

// TestPaginationConformance runs the compound-cursor MapScan pagination walk
// (M06/F22). pebble's MapScan tiebreaks on the full prefixed stored key
// (keycodec.MapKey), so the probe rebuilds cursors in that form.
func TestPaginationConformance(t *testing.T) {
	t.Parallel()

	adttest.RunPaginationConformance(t, []adttest.PaginationProbe{
		{
			Name:      "memory",
			Create:    func(t *testing.T) metaengine.Engine { return metaengine.NewMemoryEngine() },
			CursorKey: adttest.CursorKeyRaw,
		},
		{
			Name:      "pebble",
			Create:    func(t *testing.T) metaengine.Engine { return newPebbleEngineOrSkip(t) },
			CursorKey: adttest.CursorKeyRaw,
		},
	})
}

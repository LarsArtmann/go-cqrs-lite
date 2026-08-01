//go:build cgo

package duckdbengine_test

import (
	"testing"

	duckdbengine "github.com/larsartmann/go-cqrs-lite/metaengine/duckdbengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
)

// adt_matrix_cgo_test.go runs the cross-engine ADT test matrix across the
// DuckDB engine and the memory engine, asserting parity on every ADT that
// duckdbengine implements (Map, Counter, SortedMap). The harness auto-skips
// ADTs whose backend interface duckdbengine does not implement.

func TestDuckDBADTMatrix(t *testing.T) {
	t.Parallel()

	adttest.RunMatrix(t, []adttest.Factory{
		{
			Name:   "memory",
			Create: func(t *testing.T) metaengine.Engine { return metaengine.NewMemoryEngine() },
		},
		{
			Name: "duckdb",
			Create: func(t *testing.T) metaengine.Engine {
				t.Helper()

				eng, err := duckdbengine.New("")
				if err != nil {
					t.Skipf("DuckDB not available: %v", err)
				}

				return eng
			},
		},
	})
}

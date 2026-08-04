//go:build cgo

package duckdbengine_test

import (
	"testing"

	duckdbengine "github.com/larsartmann/go-cqrs-lite/metaengine/duckdbengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
)

// layout_matrix_cgo_test.go runs the LayoutPlanner capability matrix on the
// DuckDB engine. The memory engine is included but auto-skipped (it does not
// implement LayoutPlanner). Cross-engine parity applies when 2+ engines
// implement the capability.

func TestDuckDBLayoutMatrix(t *testing.T) {
	t.Parallel()

	adttest.RunLayoutMatrix(t, []adttest.Factory{
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

func TestDuckDBLayoutConflict(t *testing.T) {
	t.Parallel()

	adttest.RunLayoutConflictTest(t, []adttest.Factory{
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

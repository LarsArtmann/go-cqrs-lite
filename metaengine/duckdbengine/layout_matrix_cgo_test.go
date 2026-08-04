//go:build cgo

package duckdbengine_test

import (
	"testing"

	duckdbengine "github.com/larsartmann/go-cqrs-lite/metaengine/duckdbengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
)

// layout_matrix_cgo_test.go runs the LayoutPlanner capability matrix across
// the DuckDB engine and the SQLite engine, asserting parity on filter, sort,
// and combined filter+sort queries against planned tables.

func TestDuckDBLayoutMatrix(t *testing.T) {
	t.Parallel()

	adttest.RunLayoutMatrix(t, []adttest.Factory{
		{
			Name:   "sqlite",
			Create: newSQLiteEngineForMatrix,
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
			Name:   "sqlite",
			Create: newSQLiteEngineForMatrix,
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

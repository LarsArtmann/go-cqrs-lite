//go:build cgo

package duckdbengine_test

import (
	"testing"

	duckdbengine "github.com/larsartmann/go-cqrs-lite/metaengine/duckdbengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
)

// TestDuckDBPlannedOpsMatrix runs the D3 planned-ops parity matrix on DuckDB
// (in-memory). Skips without the CGo DuckDB driver.
func TestDuckDBPlannedOpsMatrix(t *testing.T) {
	t.Parallel()

	adttest.RunPlannedOpsMatrix(t, []adttest.Factory{
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

//go:build cgo

package bench_test

import (
	"testing"

	duckdbengine "github.com/larsartmann/go-cqrs-lite/metaengine/duckdbengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// newDuckDBEngine creates an in-memory DuckDB engine for benchmarking.
// Skips the test if DuckDB (CGo) is not available. The caller closes the
// store (which closes the engine).
func newDuckDBEngine(tb testing.TB) metaengine.Engine {
	tb.Helper()

	eng, err := duckdbengine.New("")
	if err != nil {
		tb.Skipf("DuckDB not available: %v", err)
	}

	return eng
}

//go:build cgo

package bench

import (
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/benchkit/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/duckdb/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
)

// BenchmarkBenchkitSuite_DuckDB runs the full benchkit suite against DuckDB.
// Gated behind CGo because DuckDB statically links a C++ engine — the rest of
// stack/bench stays pure-Go. DuckDB's columnar engine is tuned for analytical
// (read/GROUP BY) workloads, so its write profile differs from SQLite/Pebble.
func BenchmarkBenchkitSuite_DuckDB(b *testing.B) {
	dir := b.TempDir()

	benchkit.RunSuite(b, benchkit.Config{
		Profile:     benchkit.ProfileDev,
		PayloadSize: 128,
		Backend:     "duckdb",
		DiskPath:    dir,
	}, func() (*stack.Bundle, error) {
		return duckdb.New(filepath.Join(dir, "bench.db"))
	})
}

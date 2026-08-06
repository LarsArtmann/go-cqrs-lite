//go:build cgo

package bench

import (
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/benchkit/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/sqlite/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
)

// BenchmarkBenchkitSuite_SQLiteCGo runs the full benchkit suite against SQLite
// using the CGo-based mattn/go-sqlite3 driver (3-5x faster than modernc.org/sqlite).
//
// Compare directly with BenchmarkBenchkitSuite_SQLite (pure-Go modernc driver)
// to measure the CGo performance advantage under the same workload. Both
// benchmarks use the same profile, payload size, and disk path pattern — the
// ONLY variable is the driver.
//
// Requires CGO_ENABLED=1 and a C compiler. Gated behind a build tag so non-CGo
// builds are unaffected.
func BenchmarkBenchkitSuite_SQLiteCGo(b *testing.B) {
	dir := b.TempDir()

	benchkit.RunSuite(b, benchkit.Config{
		Profile:     benchkit.ProfileDev,
		PayloadSize: 128,
		Backend:     "sqlite-cgo",
		DiskPath:    dir,
	}, func() (*stack.Bundle, error) {
		return sqlite.New(
			filepath.Join(dir, "bench.db"),
			sqlite.WithDriverName("sqlite3"),
		)
	})
}

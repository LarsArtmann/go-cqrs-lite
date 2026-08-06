package bench

import (
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/benchkit/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	turso "github.com/larsartmann/go-cqrs-lite/stack/turso/v4"
)

// BenchmarkBenchkitSuite_Turso runs the full benchkit suite against Turso
// (embedded libSQL). Turso is SQLite-compatible with built-in sync support.
// The embedded mode used here has no remote server — it's a local file with
// the libSQL engine, which is a fork of SQLite optimized for edge replication.
func BenchmarkBenchkitSuite_Turso(b *testing.B) {
	dir := b.TempDir()

	benchkit.RunSuite(b, benchkit.Config{
		Profile:     benchkit.ProfileDev,
		PayloadSize: 128,
		Backend:     "turso",
		DiskPath:    dir,
	}, func() (*stack.Bundle, error) {
		bundle, err := turso.New(filepath.Join(dir, "bench.db"))
		if err != nil {
			return nil, err
		}

		return bundle.Bundle, nil
	})
}

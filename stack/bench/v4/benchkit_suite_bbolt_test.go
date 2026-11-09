package bench

import (
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/benchkit/v4"
	bbolt "github.com/larsartmann/go-cqrs-lite/stack/bbolt/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
)

// BenchmarkBenchkitSuite_Bolt runs the full benchkit suite against BBolt.
// BBolt is an embedded key-value store (single-file B+tree). It trades write
// throughput for simplicity — a single writer lock serializes all writes.
// Useful for low-traffic deployments that value operational simplicity.
func BenchmarkBenchkitSuite_Bolt(b *testing.B) {
	dir := b.TempDir()

	benchkit.RunSuite(b, benchkit.Config{
		Profile:     benchkit.ProfileDev,
		PayloadSize: 128,
		Backend:     "bbolt",
		DiskPath:    dir,
	}, func() (*stack.Bundle, error) {
		return bbolt.New(filepath.Join(dir, "bench.db"))
	})
}

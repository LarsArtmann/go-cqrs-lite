package bench

import (
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/benchkit/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/memory/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/pebble/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/sqlite/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4/sqlopt"
)

// BenchmarkBenchkitSuite_Memory runs the full benchkit suite against the
// memory backend as a Go testing.B benchmark. Use -benchtime=1x since each
// run is a complete write+read+project workload.
func BenchmarkBenchkitSuite_Memory(b *testing.B) {
	benchkit.RunSuite(b, benchkit.Config{
		Profile:     benchkit.ProfileDev,
		PayloadSize: 128,
		Backend:     "memory",
	}, func() (*stack.Bundle, error) {
		return memory.New()
	})
}

// BenchmarkBenchkitSuite_SQLite runs the full benchkit suite against SQLite.
// Uses optimized pragmas (64 MB cache, mmap, temp in memory) to match the
// cqrs-bench CLI configuration — so test and CLI numbers are directly comparable.
func BenchmarkBenchkitSuite_SQLite(b *testing.B) {
	dir := b.TempDir()

	benchkit.RunSuite(b, benchkit.Config{
		Profile:     benchkit.ProfileDev,
		PayloadSize: 128,
		Backend:     "sqlite",
		DiskPath:    dir,
	}, func() (*stack.Bundle, error) {
		return sqlite.New(
			filepath.Join(dir, "bench.db"),
			sqlite.WithPragmas(sqlopt.WithOptimizations()),
		)
	})
}

// BenchmarkBenchkitSuite_Pebble runs the full benchkit suite against Pebble.
func BenchmarkBenchkitSuite_Pebble(b *testing.B) {
	dir := b.TempDir()

	benchkit.RunSuite(b, benchkit.Config{
		Profile:     benchkit.ProfileDev,
		PayloadSize: 128,
		Backend:     "pebble",
		DiskPath:    dir,
	}, func() (*stack.Bundle, error) {
		bundle, err := pebble.New(dir)
		if err != nil {
			return nil, err
		}

		return bundle.Bundle, nil
	})
}

// ─── ProfileSmall (10K events) variants for more realistic numbers ───

// BenchmarkBenchkitSuite_Memory_Small runs the suite with ProfileSmall (10K
// events) against memory — more realistic than the 500-event ProfileDev.
func BenchmarkBenchkitSuite_Memory_Small(b *testing.B) {
	benchkit.RunSuite(b, benchkit.Config{
		Profile:     benchkit.ProfileSmall,
		PayloadSize: 128,
		Backend:     "memory",
	}, func() (*stack.Bundle, error) {
		return memory.New()
	})
}

// BenchmarkBenchkitSuite_SQLite_Small runs the suite with ProfileSmall (10K
// events) against SQLite with optimized pragmas.
func BenchmarkBenchkitSuite_SQLite_Small(b *testing.B) {
	dir := b.TempDir()

	benchkit.RunSuite(b, benchkit.Config{
		Profile:     benchkit.ProfileSmall,
		PayloadSize: 128,
		Backend:     "sqlite",
		DiskPath:    dir,
	}, func() (*stack.Bundle, error) {
		return sqlite.New(
			filepath.Join(dir, "bench.db"),
			sqlite.WithPragmas(sqlopt.WithOptimizations()),
		)
	})
}

// BenchmarkBenchkitSuite_Pebble_Small runs the suite with ProfileSmall (10K
// events) against Pebble.
func BenchmarkBenchkitSuite_Pebble_Small(b *testing.B) {
	dir := b.TempDir()

	benchkit.RunSuite(b, benchkit.Config{
		Profile:     benchkit.ProfileSmall,
		PayloadSize: 128,
		Backend:     "pebble",
		DiskPath:    dir,
	}, func() (*stack.Bundle, error) {
		bundle, err := pebble.New(dir)
		if err != nil {
			return nil, err
		}

		return bundle.Bundle, nil
	})
}

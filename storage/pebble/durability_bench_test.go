package pebble

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/cockroachdb/pebble"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// openDurabilityBenchStore opens a DISK-backed pebble EventStore for fsync
// measurements. b.TempDir() is tmpfs on many machines where fsync is nearly
// free, which would erase exactly the cost these benchmarks exist to measure;
// the store therefore lives under PEBBLE_DURABILITY_BENCH_DIR (default
// $HOME/.cache/pebble-durability-bench) and the benchmark skips when that
// path is unavailable.
func openDurabilityBenchStore(b *testing.B, opts ...StoreOption) *EventStore {
	b.Helper()

	base := os.Getenv("PEBBLE_DURABILITY_BENCH_DIR")
	if base == "" {
		base = filepath.Join(os.Getenv("HOME"), ".cache", "pebble-durability-bench")
	}

	if err := os.MkdirAll(base, 0o755); err != nil {
		b.Skipf("disk-backed bench dir %s unavailable: %v", base, err)
	}

	dir, err := os.MkdirTemp(base, "bench-*")
	if err != nil {
		b.Skipf("create bench dir under %s: %v", base, err)
	}

	b.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})

	database, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		b.Fatalf("pebble.Open: %v", err)
	}

	b.Cleanup(func() { _ = database.Close() })

	store, err := NewStore(database, slog.Default(), opts...)
	if err != nil {
		b.Fatalf("NewStore: %v", err)
	}

	return store
}

func benchmarkEventAppend(b *testing.B, opts ...StoreOption) {
	store := openDurabilityBenchStore(b, opts...)

	ref := id.NewStreamRef(id.StreamType("Issue"), id.NewStreamID())
	ctx := context.Background()
	version := 0

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()

		evt, err := event.NewEvent(
			"IssueCreated", ref.ID, ref.Type, event.Version(version+1),
			[]byte(`{"title":"durability-bench"}`),
		)
		if err != nil {
			b.Fatalf("create event %d: %v", i, err)
		}

		b.StartTimer()

		if err := store.AppendBatch(ctx, ref, []event.Event{evt}); err != nil {
			b.Fatalf("AppendBatch %d: %v", i, err)
		}

		version++
	}
}

// BenchmarkEventAppendSync measures append throughput with pebble.Sync per
// write (the store default, stack.DurabilityStrict). Every append pays an
// fsync before returning.
func BenchmarkEventAppendSync(b *testing.B) {
	benchmarkEventAppend(b)
}

// BenchmarkEventAppendAsync measures append throughput with async writes
// (WithAsyncWrites, stack.DurabilityNormal): the WAL is written to the page
// cache but the fsync is skipped. The delta vs BenchmarkEventAppendSync is
// the per-append fsync cost that the Normal durability tier saves.
func BenchmarkEventAppendAsync(b *testing.B) {
	benchmarkEventAppend(b, WithAsyncWrites())
}

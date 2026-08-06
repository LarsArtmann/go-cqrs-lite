package metaengine

import (
	"context"
	"database/sql"
	"testing"

)

// BenchmarkCalibration_MapSet measures the per-op cost of MapSet on the
// in-memory engine. Used to calibrate MemoryNsPerOp.
func BenchmarkCalibration_MapSet(b *testing.B) {
	eng := NewMemoryEngine()
	mb := eng.(MapBackend)
	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := mb.MapSet(ctx, "bench", i, i*2); err != nil {
			b.Fatalf("MapSet %d: %v", i, err)
		}
	}
}

// BenchmarkCalibration_MapGet measures the per-op cost of MapGet on the
// in-memory engine. Used to calibrate MemoryNsPerOp.
func BenchmarkCalibration_MapGet(b *testing.B) {
	eng := NewMemoryEngine()
	mb := eng.(MapBackend)
	ctx := context.Background()

	// Pre-populate.
	for i := 0; i < 1000; i++ {
		if err := mb.MapSet(ctx, "bench", i, i*2); err != nil {
			b.Fatalf("pre-populate MapSet %d: %v", i, err)
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, found, err := mb.MapGet(ctx, "bench", i%1000)
		if err != nil {
			b.Fatalf("MapGet %d: %v", i, err)
		}
		if !found {
			b.Fatalf("MapGet %d: key not found", i)
		}
	}
}

// BenchmarkCalibration_SQLiteSet measures the per-op cost of MapSet on the
// SQLite engine. Used to calibrate SQLiteNsPerOp.
func BenchmarkCalibration_SQLiteSet(b *testing.B) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		b.Fatalf("sql.Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	eng, err := newMemoryEngineForTest()
	if err != nil {
		b.Fatalf("NewSQLiteEngine: %v", err)
	}
	mb, ok := eng.(MapBackend)
	if !ok {
		b.Fatal("sqlite engine does not implement MapBackend")
	}
	defer func() { _ = eng.Close() }()

	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := mb.MapSet(ctx, "bench", i, i*2); err != nil {
			b.Fatalf("MapSet %d: %v", i, err)
		}
	}
}

// BenchmarkCalibration_SQLiteGet measures the per-op cost of MapGet on the
// SQLite engine. Used to calibrate SQLiteNsPerOp.
func BenchmarkCalibration_SQLiteGet(b *testing.B) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		b.Fatalf("sql.Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	eng, err := newMemoryEngineForTest()
	if err != nil {
		b.Fatalf("NewSQLiteEngine: %v", err)
	}
	mb, ok := eng.(MapBackend)
	if !ok {
		b.Fatal("sqlite engine does not implement MapBackend")
	}
	defer func() { _ = eng.Close() }()

	ctx := context.Background()

	// Pre-populate.
	for i := 0; i < 1000; i++ {
		if err := mb.MapSet(ctx, "bench", i, i*2); err != nil {
			b.Fatalf("pre-populate MapSet %d: %v", i, err)
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, found, err := mb.MapGet(ctx, "bench", i%1000)
		if err != nil {
			b.Fatalf("MapGet %d: %v", i, err)
		}
		if !found {
			b.Fatalf("MapGet %d: key not found", i)
		}
	}
}

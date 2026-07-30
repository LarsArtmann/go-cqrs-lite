package metaengine

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

// BenchmarkRawReader_Get benchmarks the RawValueReader fast path (1 JSON op)
// vs the standard MapGet path (3 JSON ops) for point lookups.
func BenchmarkRawReader_Get(b *testing.B) {
	db := setupBenchDB(b)
	eng, _ := NewSQLiteEngine(db)

	se := eng.(*sqliteEngine)

	ctx := context.Background()
	col := "bench_raw"

	for range b.N {
		se.MapSet(ctx, col, "key", benchPayload{ID: "key", Status: "open"})
	}

	b.ResetTimer()
	b.Run("RawValueReader", func(b *testing.B) {
		for range b.N {
			_, _, _ = se.GetRawValue(ctx, col, "key")
		}
	})

	b.Run("MapGet", func(b *testing.B) {
		for range b.N {
			_, _, _ = se.MapGet(ctx, col, "key")
		}
	})
}

// BenchmarkRawReader_Scan benchmarks the raw scan path vs pushdown vs closure.
func BenchmarkRawReader_Scan(b *testing.B) {
	db := setupBenchDB(b)
	eng, _ := NewSQLiteEngine(db)

	se := eng.(*sqliteEngine)

	ctx := context.Background()
	col := "bench_scan"

	for i := range 100 {
		se.MapSet(ctx, col, "k"+itoa(i), benchPayload{ID: "k" + itoa(i), Status: "open"})
	}

	b.ResetTimer()

	filters := []FilterSpec{{Column: "status", Op: FilterEq, Value: "open"}}

	b.Run("ScanRawValues", func(b *testing.B) {
		for range b.N {
			_, _ = se.ScanRawValues(ctx, col, filters, nil, nil, 10)
		}
	})

	b.Run("PushdownMapScan", func(b *testing.B) {
		for range b.N {
			_, _ = se.PushdownMapScan(ctx, col, filters, nil, nil, 10)
		}
	})
}

// BenchmarkKeyEncoding benchmarks the optimized key encoder vs JSON for different key types.
func BenchmarkKeyEncoding(b *testing.B) {
	b.Run("string", func(b *testing.B) {
		for range b.N {
			_ = encodeKey("user:abc-123")
		}
	})

	b.Run("int64", func(b *testing.B) {
		for range b.N {
			_ = encodeKey(int64(42))
		}
	})

	b.Run("struct", func(b *testing.B) {
		type complexKey struct {
			Tenant string
			ID     string
		}

		for range b.N {
			_ = encodeKey(complexKey{Tenant: "acme", ID: "user-123"})
		}
	})
}

// BenchmarkStmtCache benchmarks the prepared statement cache effect.
func BenchmarkStmtCache(b *testing.B) {
	db := setupBenchDB(b)
	eng, _ := NewSQLiteEngine(db)

	se := eng.(*sqliteEngine)

	ctx := context.Background()

	// Warm the cache.
	se.MapSet(ctx, "bench", "warmup", "v")

	b.Run("cached", func(b *testing.B) {
		for range b.N {
			_, _ = se.cache.exec(ctx, se.queries.mapSet, "bench", "key", "val")
		}
	})

	b.Run("uncached", func(b *testing.B) {
		for range b.N {
			_, _ = se.db.ExecContext(ctx, se.queries.mapSet, "bench", "key", "val")
		}
	})
}

type benchPayload struct {
	ID     string
	Status string
}

func setupBenchDB(b *testing.B) *sql.DB {
	b.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		b.Fatal(err)
	}

	b.Cleanup(func() { db.Close() })

	return db
}

func itoa(i int) string {
	const digits = "0123456789"
	if i == 0 {
		return "0"
	}

	var buf [20]byte

	pos := len(buf)

	for i > 0 {
		pos--
		buf[pos] = digits[i%10]
		i /= 10
	}

	return string(buf[pos:])
}

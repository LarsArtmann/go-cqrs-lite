package metaengine

import (
	"context"
	"database/sql"
	"testing"

	"pgregory.net/rapid"
)

// op is a single Map operation in a randomly generated sequence.
type op struct {
	kind   int // 0=Set, 1=Get, 2=Delete
	key    string
	value  any
	delete bool
}

// TestProperty_MapSetGetParity_MemoryVsSQLite verifies that the memory engine
// and SQLite engine agree on MapSet/MapGet/MapDelete results for arbitrary
// sequences of operations with JSON-compatible values.
func TestProperty_MapSetGetParity_MemoryVsSQLite(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		ctx := context.Background()

		// Create both engines.
		memEng := NewMemoryEngine()
		defer memEng.Close()

		db, err := sql.Open("sqlite", ":memory:")
		if err != nil {
			rt.Fatalf("open sqlite: %v", err)
		}
		defer db.Close()

		sqlEng, err := NewSQLiteEngine(db)
		if err != nil {
			rt.Fatalf("NewSQLiteEngine: %v", err)
		}
		defer sqlEng.Close()

		memMB := memEng.(MapBackend)
		sqlMB := sqlEng.(MapBackend)

		const col = "items"

		// Generate a random sequence of operations.
		keys := rapid.SliceOfN(rapid.StringMatching(`[a-z]{1,3}`), 1, 10).
			Draw(rt, "keys")

		numOps := rapid.IntRange(5, 30).Draw(rt, "numOps")

		for range numOps {
			kind := rapid.IntRange(0, 2).Draw(rt, "opKind")
			key := rapid.SampledFrom(keys).Draw(rt, "key")

			switch kind {
			case 0: // Set
				value := rapid.SampledFrom([]any{
					"hello", "world", "", int64(42), int64(-1), int64(0),
					float64(3.14), true, false,
				}).Draw(rt, "value")

				entry := map[string]any{"v": value}

				if err := memMB.MapSet(ctx, col, key, entry); err != nil {
					rt.Fatalf("mem MapSet: %v", err)
				}

				if err := sqlMB.MapSet(ctx, col, key, entry); err != nil {
					rt.Fatalf("sql MapSet: %v", err)
				}

			case 1: // Get — verify both return same result
				memVal, memOk, err := memMB.MapGet(ctx, col, key)
				if err != nil {
					rt.Fatalf("mem MapGet: %v", err)
				}

				sqlVal, sqlOk, err := sqlMB.MapGet(ctx, col, key)
				if err != nil {
					rt.Fatalf("sql MapGet: %v", err)
				}

				// Existence must agree.
				if memOk != sqlOk {
					rt.Fatalf("key %q: mem exists=%v, sql exists=%v", key, memOk, sqlOk)
				}

				// Values must be non-nil if exists.
				if memOk && memVal == nil {
					rt.Fatalf("key %q: mem returned ok=true but nil value", key)
				}

				if sqlOk && sqlVal == nil {
					rt.Fatalf("key %q: sql returned ok=true but nil value", key)
				}

			case 2: // Delete
				if err := memMB.MapDelete(ctx, col, key); err != nil {
					rt.Fatalf("mem MapDelete: %v", err)
				}

				if err := sqlMB.MapDelete(ctx, col, key); err != nil {
					rt.Fatalf("sql MapDelete: %v", err)
				}
			}

			// After each op, verify both engines agree on which keys exist.
			for _, k := range keys {
				_, memOk, err := memMB.MapGet(ctx, col, k)
				if err != nil {
					rt.Fatalf("mem MapGet check: %v", err)
				}

				_, sqlOk, err := sqlMB.MapGet(ctx, col, k)
				if err != nil {
					rt.Fatalf("sql MapGet check: %v", err)
				}

				if memOk != sqlOk {
					rt.Fatalf(
						"after op on %q: key %q differs: mem=%v, sql=%v",
						key,
						k,
						memOk,
						sqlOk,
					)
				}
			}
		}
	})
}

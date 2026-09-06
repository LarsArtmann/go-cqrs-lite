package sqlopt_test

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite" // register sqlite driver

	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4/sqlopt"
	"github.com/larsartmann/go-cqrs-lite/storage/v4"
)

func TestSQLiteSynchronousLevel(t *testing.T) {
	t.Parallel()

	cases := []struct {
		tier   stack.DurabilityTier
		expect string
	}{
		{stack.DurabilityStrict, "FULL"},
		{stack.DurabilityNormal, "NORMAL"},
		{stack.DurabilityRelaxed, "OFF"},
		{"", "NORMAL"},      // unset → default
		{"bogus", "NORMAL"}, // unknown → default
	}

	for _, tc := range cases {
		t.Run(string(tc.tier), func(t *testing.T) {
			t.Parallel()

			got := sqlopt.SQLiteSynchronousLevel(tc.tier)
			if got != tc.expect {
				t.Fatalf("SQLiteSynchronousLevel(%q) = %q, want %q", tc.tier, got, tc.expect)
			}
		})
	}
}

func TestApplySQLiteDurability_NormalKeepsNormal(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	defer func() { _ = db.Close() }()

	ctx := context.Background()

	if err := storage.SQLiteEnableWAL(ctx, db); err != nil {
		t.Fatalf("enable WAL: %v", err)
	}

	// Normal is applied explicitly and must keep the WAL default level (1).
	if err := sqlopt.ApplySQLiteDurability(ctx, db, stack.DurabilityNormal); err != nil {
		t.Fatalf("ApplySQLiteDurability Normal: %v", err)
	}

	if level := querySyncLevel(t, db, ctx); level != 1 {
		t.Fatalf("synchronous = %d, want 1 (NORMAL)", level)
	}

	if err := sqlopt.ApplySQLiteDurability(ctx, db, ""); err != nil {
		t.Fatalf("ApplySQLiteDurability empty: %v", err)
	}
}

// TestApplySQLiteDurability_WithoutWAL pins the non-WAL path: the tier must
// reach SQLite even when SQLiteEnableWAL was never called. Before this was
// fixed, the tier application was nested under the WAL flag and Relaxed
// silently stayed at the SQLite FULL (2) default.
func TestApplySQLiteDurability_WithoutWAL(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		tier  stack.DurabilityTier
		level int
	}{
		{"relaxed_is_off_not_full", stack.DurabilityRelaxed, 0},
		{"normal", stack.DurabilityNormal, 1},
		{"strict", stack.DurabilityStrict, 2},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db, err := sql.Open("sqlite", ":memory:")
			if err != nil {
				t.Fatalf("open: %v", err)
			}

			defer func() { _ = db.Close() }()

			ctx := context.Background()

			if err := sqlopt.ApplySQLiteDurability(ctx, db, tc.tier); err != nil {
				t.Fatalf("ApplySQLiteDurability %s: %v", tc.tier, err)
			}

			if level := querySyncLevel(t, db, ctx); level != tc.level {
				t.Fatalf("synchronous = %d, want %d", level, tc.level)
			}
		})
	}
}

func querySyncLevel(t *testing.T, db *sql.DB, ctx context.Context) int {
	t.Helper()

	var level int

	if err := db.QueryRowContext(ctx, "PRAGMA synchronous").Scan(&level); err != nil {
		t.Fatalf("query synchronous: %v", err)
	}

	return level
}

func TestApplySQLiteDurability_StrictOverridesToFull(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	defer func() { _ = db.Close() }()

	ctx := context.Background()

	if err := storage.SQLiteEnableWAL(ctx, db); err != nil {
		t.Fatalf("enable WAL: %v", err)
	}

	if err := sqlopt.ApplySQLiteDurability(ctx, db, stack.DurabilityStrict); err != nil {
		t.Fatalf("ApplySQLiteDurability Strict: %v", err)
	}

	var syncLevel int

	if err := db.QueryRowContext(ctx, "PRAGMA synchronous").Scan(&syncLevel); err != nil {
		t.Fatalf("query synchronous: %v", err)
	}

	// SQLite synchronous FULL = 2
	if syncLevel != 2 {
		t.Fatalf("synchronous = %d, want 2 (FULL)", syncLevel)
	}
}

func TestApplySQLiteDurability_RelaxedOverridesToOff(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	defer func() { _ = db.Close() }()

	ctx := context.Background()

	if err := storage.SQLiteEnableWAL(ctx, db); err != nil {
		t.Fatalf("enable WAL: %v", err)
	}

	if err := sqlopt.ApplySQLiteDurability(ctx, db, stack.DurabilityRelaxed); err != nil {
		t.Fatalf("ApplySQLiteDurability Relaxed: %v", err)
	}

	var syncLevel int

	if err := db.QueryRowContext(ctx, "PRAGMA synchronous").Scan(&syncLevel); err != nil {
		t.Fatalf("query synchronous: %v", err)
	}

	// SQLite synchronous OFF = 0
	if syncLevel != 0 {
		t.Fatalf("synchronous = %d, want 0 (OFF)", syncLevel)
	}
}

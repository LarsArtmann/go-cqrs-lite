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
		{"", "NORMAL"},     // unset → default
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

func TestApplySQLiteDurability_NormalIsNoOp(t *testing.T) {
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

	// Normal should be a no-op (WAL already set NORMAL).
	if err := sqlopt.ApplySQLiteDurability(ctx, db, stack.DurabilityNormal); err != nil {
		t.Fatalf("ApplySQLiteDurability Normal: %v", err)
	}

	if err := sqlopt.ApplySQLiteDurability(ctx, db, ""); err != nil {
		t.Fatalf("ApplySQLiteDurability empty: %v", err)
	}
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

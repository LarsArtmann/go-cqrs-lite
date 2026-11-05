package sqliteengine

import (
	"testing"
)

// TestPragmaOverrideCallerWins pins the NewSQLiteEngineFromDSNWith pragma
// contract: production defaults apply FIRST, caller pragmas LAST, and SQLite
// is last-write-wins per connection — so a caller-supplied cache_size or
// busy_timeout must beat the built-in values on the SAME pooled connection
// (the pool is capped at one conn unless WithMaxOpenConns raises it, which
// makes the readback deterministic). This is the golden test for the
// defaults-then-caller ordering in dsl.go; a reordering (caller pragmas
// applied before the defaults) silently reverts to the 2 MB page cache and
// the 5 s busy timeout — the exact regression class the ordering exists to
// prevent.
func TestPragmaOverrideCallerWins(t *testing.T) {
	t.Parallel()

	eng, err := NewSQLiteEngineFromDSNWith(
		"file:"+t.TempDir()+"/pragma_override.db?mode=rwc",
		[]string{"cache_size=-8192", "busy_timeout=1234"},
	)
	if err != nil {
		t.Fatalf("NewSQLiteEngineFromDSNWith: %v", err)
	}

	t.Cleanup(func() { _ = eng.Close() })

	se, ok := eng.(*sqliteEngine)
	if !ok {
		t.Fatalf("engine is %T, want *sqliteEngine", eng)
	}

	ctx := t.Context()

	var cache int
	if err := se.xd(ctx).QueryRowContext(ctx, "PRAGMA cache_size").Scan(&cache); err != nil {
		t.Fatalf("read PRAGMA cache_size: %v", err)
	}

	if cache != -8192 {
		t.Fatalf("PRAGMA cache_size = %d, want -8192 (caller 8 MiB beats the 64 MiB default)", cache)
	}

	var busy int
	if err := se.xd(ctx).QueryRowContext(ctx, "PRAGMA busy_timeout").Scan(&busy); err != nil {
		t.Fatalf("read PRAGMA busy_timeout: %v", err)
	}

	if busy != 1234 {
		t.Fatalf("PRAGMA busy_timeout = %d, want 1234 (caller beats the 5000 default)", busy)
	}
}

// The same constructor WITHOUT caller pragmas keeps the production defaults:
// the override test only means anything if the default path is pinned too.
func TestPragmaDefaultsWithoutCaller(t *testing.T) {
	t.Parallel()

	eng, err := NewSQLiteEngineFromDSNWith("file:"+t.TempDir()+"/pragma_defaults.db?mode=rwc", nil)
	if err != nil {
		t.Fatalf("NewSQLiteEngineFromDSNWith: %v", err)
	}

	t.Cleanup(func() { _ = eng.Close() })

	se, ok := eng.(*sqliteEngine)
	if !ok {
		t.Fatalf("engine is %T, want *sqliteEngine", eng)
	}

	ctx := t.Context()

	var cache int
	if err := se.xd(ctx).QueryRowContext(ctx, "PRAGMA cache_size").Scan(&cache); err != nil {
		t.Fatalf("read PRAGMA cache_size: %v", err)
	}

	if cache != -65536 {
		t.Fatalf("PRAGMA cache_size = %d, want -65536 (64 MiB production default)", cache)
	}

	var busy int
	if err := se.xd(ctx).QueryRowContext(ctx, "PRAGMA busy_timeout").Scan(&busy); err != nil {
		t.Fatalf("read PRAGMA busy_timeout: %v", err)
	}

	if busy != 5000 {
		t.Fatalf("PRAGMA busy_timeout = %d, want 5000 (production default)", busy)
	}
}

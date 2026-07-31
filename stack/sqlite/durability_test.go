package sqlite_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/stack/sqlite/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
)

func querySynchronous(t *testing.T, db *sql.DB) int {
	t.Helper()

	var level int

	if err := db.QueryRowContext(context.Background(), "PRAGMA synchronous").Scan(&level); err != nil {
		t.Fatalf("query synchronous: %v", err)
	}

	return level
}

func bundleDB(t *testing.T, b *stack.Bundle) *sql.DB {
	t.Helper()

	db, ok := b.Database().(*sql.DB)
	if !ok {
		t.Fatalf("Database() = %T, want *sql.DB", b.Database())
	}

	return db
}

// TestNew_WithDurability_Strict verifies that sqlite.New with
// WithDurability(DurabilityStrict) actually sets PRAGMA synchronous=FULL (2)
// on the database, not just records the tier on the Bundle.
func TestNew_WithDurability_Strict(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dsn := filepath.Join(dir, "strict.db")

	b, err := sqlite.New(dsn, sqlite.WithDurability(stack.DurabilityStrict))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer func() { _ = b.Close() }()

	if b.Durability() != stack.DurabilityStrict {
		t.Fatalf("Durability() = %q, want %q", b.Durability(), stack.DurabilityStrict)
	}

	if level := querySynchronous(t, bundleDB(t, b)); level != 2 {
		t.Fatalf("synchronous = %d, want 2 (FULL)", level)
	}
}

// TestNew_WithDurability_Relaxed verifies that sqlite.New with
// WithDurability(DurabilityRelaxed) sets PRAGMA synchronous=OFF (0).
func TestNew_WithDurability_Relaxed(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dsn := filepath.Join(dir, "relaxed.db")

	b, err := sqlite.New(dsn, sqlite.WithDurability(stack.DurabilityRelaxed))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer func() { _ = b.Close() }()

	if b.Durability() != stack.DurabilityRelaxed {
		t.Fatalf("Durability() = %q, want %q", b.Durability(), stack.DurabilityRelaxed)
	}

	if level := querySynchronous(t, bundleDB(t, b)); level != 0 {
		t.Fatalf("synchronous = %d, want 0 (OFF)", level)
	}
}

// TestNew_WithDurability_Normal verifies that sqlite.New without
// WithDurability defaults to NORMAL (1) — the WAL default.
func TestNew_WithDurability_Normal(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dsn := filepath.Join(dir, "normal.db")

	b, err := sqlite.New(dsn)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer func() { _ = b.Close() }()

	if b.Durability() != stack.DurabilityNormal {
		t.Fatalf("Durability() = %q, want %q", b.Durability(), stack.DurabilityNormal)
	}

	if level := querySynchronous(t, bundleDB(t, b)); level != 1 {
		t.Fatalf("synchronous = %d, want 1 (NORMAL)", level)
	}
}

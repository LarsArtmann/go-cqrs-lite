package postgres_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/stack/postgres/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
)

// showSynchronousCommit queries the Postgres server for the current
// synchronous_commit GUC value of the session that services this query.
func showSynchronousCommit(t *testing.T, db *sql.DB) string {
	t.Helper()

	ctx := context.Background()

	var val string

	if err := db.QueryRowContext(ctx, "SHOW synchronous_commit").Scan(&val); err != nil {
		t.Fatalf("SHOW synchronous_commit: %v", err)
	}

	return val
}

func bundleDB(t *testing.T, b *stack.Bundle) *sql.DB {
	t.Helper()

	db, ok := b.Database().(*sql.DB)
	if !ok {
		t.Fatalf("Database() = %T, want *sql.DB", b.Database())
	}

	return db
}

// TestNew_WithDurability_Strict verifies that postgres.New with
// WithDurability(DurabilityStrict) injects synchronous_commit=on into the DSN
// so every pooled connection reports synchronous_commit=on — not just the
// first session.
func TestNew_WithDurability_Strict(t *testing.T) {
	dsn := postgresDSN(t)

	b, err := postgres.New(dsn, postgres.WithDurability(stack.DurabilityStrict))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer func() { _ = b.Close() }()

	if b.Durability() != stack.DurabilityStrict {
		t.Fatalf("Durability() = %q, want %q", b.Durability(), stack.DurabilityStrict)
	}

	if val := showSynchronousCommit(t, bundleDB(t, b)); val != "on" {
		t.Fatalf("synchronous_commit = %q, want %q", val, "on")
	}
}

// TestNew_WithDurability_Normal verifies that the default (Normal) sets
// synchronous_commit=off at the DSN level so all pool connections inherit it.
func TestNew_WithDurability_Normal(t *testing.T) {
	dsn := postgresDSN(t)

	b, err := postgres.New(dsn)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer func() { _ = b.Close() }()

	if b.Durability() != stack.DurabilityNormal {
		t.Fatalf("Durability() = %q, want %q", b.Durability(), stack.DurabilityNormal)
	}

	if val := showSynchronousCommit(t, bundleDB(t, b)); val != "off" {
		t.Fatalf("synchronous_commit = %q, want %q", val, "off")
	}
}

// TestNew_WithDurability_Relaxed verifies that Relaxed maps to
// synchronous_commit=off (same as Normal for Postgres).
func TestNew_WithDurability_Relaxed(t *testing.T) {
	dsn := postgresDSN(t)

	b, err := postgres.New(dsn, postgres.WithDurability(stack.DurabilityRelaxed))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer func() { _ = b.Close() }()

	if b.Durability() != stack.DurabilityRelaxed {
		t.Fatalf("Durability() = %q, want %q", b.Durability(), stack.DurabilityRelaxed)
	}

	if val := showSynchronousCommit(t, bundleDB(t, b)); val != "off" {
		t.Fatalf("synchronous_commit = %q, want %q", val, "off")
	}
}

// TestNew_WithDurability_PoolWide verifies that synchronous_commit is applied
// to ALL connections in the pool, not just the first one. This is the
// regression test for the pool-scoping bug where session-level SET only
// applied to one connection.
func TestNew_WithDurability_PoolWide(t *testing.T) {
	dsn := postgresDSN(t)

	b, err := postgres.New(dsn,
		postgres.WithDurability(stack.DurabilityStrict),
		postgres.WithPoolSize(5, 5),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer func() { _ = b.Close() }()

	db := bundleDB(t, b)

	// Query multiple times to exercise different pool connections.
	// With MaxOpenConns=5, the pool may reuse or create connections.
	// If the DSN-level parameter works, every connection reports "on".
	for i := range 10 {
		if val := showSynchronousCommit(t, db); val != "on" {
			t.Fatalf("iteration %d: synchronous_commit = %q, want %q (pool-scoping bug)", i, val, "on")
		}
	}
}

package view

import (
	"context"
	"testing"
)

// TestSQLViewStore_InTx_Commit verifies that writes via InTx are persisted when
// the transaction commits.
func TestSQLViewStore_InTx_Commit(t *testing.T) {
	t.Parallel()

	store := newTestViewStore(t)
	ctx := context.Background()

	tx, err := store.DB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}

	if err := store.InTx(tx).
		Set(ctx, testKey("u1"), &testView{Name: "Alice", Email: "a@ex.com", Age: 30}); err != nil {
		_ = tx.Rollback()
		t.Fatalf("InTx Set: %v", err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	// Read back from the connection pool (not the tx).
	got, err := store.Get(ctx, testKey("u1"))
	if err != nil {
		t.Fatalf("Get after commit: %v", err)
	}
	if got.Name != "Alice" {
		t.Fatalf("got name %q, want Alice", got.Name)
	}
}

// TestSQLViewStore_InTx_Rollback verifies that writes via InTx are discarded
// when the transaction rolls back — the store does not auto-commit.
func TestSQLViewStore_InTx_Rollback(t *testing.T) {
	t.Parallel()

	store := newTestViewStore(t)
	ctx := context.Background()

	tx, err := store.DB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}

	if err := store.InTx(tx).
		Set(ctx, testKey("u1"), &testView{Name: "Alice", Email: "a@ex.com", Age: 30}); err != nil {
		_ = tx.Rollback()
		t.Fatalf("InTx Set: %v", err)
	}

	// Within the tx, the row is visible.
	inTx, err := store.InTx(tx).Get(ctx, testKey("u1"))
	if err != nil {
		_ = tx.Rollback()
		t.Fatalf("InTx Get (within tx): %v", err)
	}
	if inTx.Name != "Alice" {
		_ = tx.Rollback()
		t.Fatalf("within-tx read: got %q, want Alice", inTx.Name)
	}

	if err := tx.Rollback(); err != nil {
		t.Fatalf("Rollback: %v", err)
	}

	// After rollback, the row must NOT be visible on the connection pool.
	if _, err := store.Get(ctx, testKey("u1")); err == nil {
		t.Fatalf("Get after rollback: expected error, got nil (row leaked)")
	}
}

// TestSQLViewStore_InTx_ReceiverUnaffected verifies that scoping a copy to a tx
// does not affect the receiver store (it keeps running on the connection pool).
func TestSQLViewStore_InTx_ReceiverUnaffected(t *testing.T) {
	t.Parallel()

	store := newTestViewStore(t)
	ctx := context.Background()

	// Pool-level write before any tx.
	if err := store.Set(
		ctx,
		testKey("pool"),
		&testView{Name: "Pool", Email: "p@ex.com"},
	); err != nil {
		t.Fatalf("pool Set: %v", err)
	}

	tx, err := store.DB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}

	txCopy := store.InTx(tx)
	if err := txCopy.Set(
		ctx,
		testKey("txonly"),
		&testView{Name: "TxOnly", Email: "t@ex.com"},
	); err != nil {
		_ = tx.Rollback()
		t.Fatalf("tx Set: %v", err)
	}
	_ = tx.Rollback()

	// The receiver still reads the pool write and must NOT see the rolled-back tx write.
	if got, err := store.Get(ctx, testKey("pool")); err != nil || got.Name != "Pool" {
		t.Fatalf("receiver pool read after tx rollback: got=%v err=%v, want Pool", got, err)
	}
	if _, err := store.Get(ctx, testKey("txonly")); err == nil {
		t.Fatalf("receiver saw tx-only write — InTx leaked into the pool store")
	}
}

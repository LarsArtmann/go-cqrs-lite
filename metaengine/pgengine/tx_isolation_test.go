package pgengine_test

import (
	"context"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	pgengine "github.com/larsartmann/go-cqrs-lite/metaengine/pgengine/v4"
)

// TestPGEngine_TxIsolationFromForeignContext pins the transaction
// visibility contract: a transaction is visible only to calls whose context
// descends from RunInTx's fn. A concurrent caller with its own context must
// never observe uncommitted writes, because transaction affinity must flow
// through the context — an engine-global "active tx" leaks the transaction
// to unrelated goroutines (dirty reads, and readers dying with
// "sql: Rows are closed" when the foreign tx commits mid-iteration).
// Postgres MVCC keeps uncommitted rows invisible to other sessions, which
// is what makes the foreign read a discriminator. Ported from sqliteengine
// 22ab7b218.
func TestPGEngine_TxIsolationFromForeignContext(t *testing.T) {
	t.Parallel()

	eng, err := pgengine.New(pgDSN(t))
	if err != nil {
		t.Skipf("Postgres not available: %v", err)
	}

	t.Cleanup(func() { metaengine.DeferClose(eng) })

	tx := eng.(metaengine.Transactional)
	mb := eng.(metaengine.MapBackend)

	const col = "tx_iso_map"

	inside := make(chan struct{})
	commit := make(chan struct{})
	done := make(chan error, 1)

	go func() {
		done <- tx.RunInTx(context.Background(), func(tctx context.Context) error {
			if err := mb.MapSet(tctx, col, "ghost", "uncommitted"); err != nil {
				return err
			}

			close(inside)
			<-commit

			return nil
		})
	}()

	<-inside

	_, found, err := mb.MapGet(context.Background(), col, "ghost")
	if err != nil {
		t.Fatalf("MapGet from foreign context while tx open: %v", err)
	}

	if found {
		t.Fatalf("dirty read: uncommitted tx write visible to a foreign context — " +
			"transaction leaked across goroutines via engine-global state")
	}

	close(commit)

	if err := <-done; err != nil {
		t.Fatalf("RunInTx: %v", err)
	}

	_, found, err = mb.MapGet(context.Background(), col, "ghost")
	if err != nil {
		t.Fatalf("MapGet after commit: %v", err)
	}

	if !found {
		t.Fatalf("committed write must be visible after RunInTx returns")
	}
}

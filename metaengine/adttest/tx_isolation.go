package adttest

import (
	"context"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// AssertTxIsolationFromForeignContext pins the transaction-isolation
// contract on one engine instance: writes inside a RunInTx transaction must
// stay invisible to unrelated contexts while the transaction is open (no
// dirty reads through engine-global state) and become visible once it
// commits. Engines that leak the transaction across goroutines fail here —
// historically as dirty reads, or as readers dying with "sql: Rows are
// closed" when the foreign transaction commits mid-iteration. Run it from
// every engine module's test suite; the scenario is engine-agnostic, only
// the engine setup differs per module.
func AssertTxIsolationFromForeignContext(t *testing.T, eng metaengine.Engine) {
	t.Helper()

	tx, ok := eng.(metaengine.Transactional)
	if !ok {
		t.Fatalf("engine does not implement metaengine.Transactional")
	}
	mb, ok := eng.(metaengine.MapBackend)
	if !ok {
		t.Fatalf("engine does not implement metaengine.MapBackend")
	}

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

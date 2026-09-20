//go:build cgo

package duckdbengine_test

import (
	"context"
	"path/filepath"
	"testing"

	duckdbengine "github.com/larsartmann/go-cqrs-lite/metaengine/duckdbengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// openFileEngine builds a file-backed engine with a real connection pool:
// concurrent readers and the transaction can hold connections at the same
// time, which in-memory single-connection setups cannot reproduce (ported
// from sqliteengine's WAL opener, 22ab7b218).
func openFileEngine(t *testing.T) metaengine.Engine {
	t.Helper()

	eng, err := duckdbengine.New(filepath.Join(t.TempDir(), "tx_iso.duckdb"))
	if err != nil {
		t.Skipf("DuckDB not available: %v", err)
	}

	t.Cleanup(func() { metaengine.DeferClose(eng) })

	return eng
}

// TestDuckDBEngine_TxIsolationFromForeignContext pins the transaction
// visibility contract: a transaction is visible only to calls whose context
// descends from RunInTx's fn. A concurrent caller with its own context must
// never observe uncommitted writes, because transaction affinity must flow
// through the context — an engine-global "active tx" leaks the transaction
// to unrelated goroutines (dirty reads, and readers dying with
// "sql: Rows are closed" when the foreign tx commits mid-iteration).
func TestDuckDBEngine_TxIsolationFromForeignContext(t *testing.T) {
	t.Parallel()

	eng := openFileEngine(t)

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

// The sqliteengine stress companion (concurrent StreamRead vs
// StreamAppendExpected) is deliberately NOT ported: through the
// go-duckdb driver, concurrent prepare on separate pool connections to a
// file DSN convolves inside the C bindings (13-minute prepare blocks,
// observed 2026-09-20) — a driver-level serialization, not a transaction
// affinity signal, so the pattern cannot discriminate here.

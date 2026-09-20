//go:build cgo

package duckdbengine_test

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

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

// TestDuckDBEngine_ConcurrentStreamReadVsAppendExpected reproduces the
// production flake class behind CRM timeline loads: a StreamRead landing
// while a concurrent StreamAppendExpected transaction is open gets handed
// the foreign *sql.Tx by engine-global state; the tx commits and the
// reader's rows die with "sql: Rows are closed". Reads and writes on
// separate streams must be fully independent.
func TestDuckDBEngine_ConcurrentStreamReadVsAppendExpected(t *testing.T) {
	t.Parallel()

	eng := openFileEngine(t)

	sb := eng.(metaengine.StreamLogBackend)
	aa := eng.(metaengine.AtomicAppender)

	const (
		readCol  = "tx_stress_read"
		writeCol = "tx_stress_write"
		sid      = "s1"
	)

	seed := make([]any, 64)
	for i := range seed {
		seed[i] = fmt.Sprintf("seed-%02d", i)
	}

	if err := sb.StreamAppend(context.Background(), readCol, sid, seed); err != nil {
		t.Fatalf("seed StreamAppend: %v", err)
	}

	const (
		writers  = 2
		readers  = 2
		writes   = 100
		deadline = 20 * time.Second
	)

	var (
		wg      sync.WaitGroup
		failure atomic.Value // string
	)

	deadlineCh := time.After(deadline)

	for range writers {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for i := range writes {
				select {
				case <-deadlineCh:
					return
				default:
				}


				if err := aa.StreamAppendExpected(
					context.Background(), writeCol, sid, int64(i), []any{fmt.Sprintf("w-%03d", i)},
				); err != nil {
					failure.Store(fmt.Sprintf("StreamAppendExpected: %v", err))

					return
				}
			}
		}()
	}

	for range readers {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for {
				select {
				case <-deadlineCh:
					return
				default:
				}


				vals, err := sb.StreamRead(context.Background(), readCol, sid)
				if err != nil {
					failure.Store(fmt.Sprintf("StreamRead: %v", err))

					return
				}

				if len(vals) != len(seed) {
					failure.Store(fmt.Sprintf("StreamRead len=%d want %d", len(vals), len(seed)))

					return
				}
			}
		}()
	}

	wg.Wait()

	if msg, ok := failure.Load().(string); ok {
		t.Fatalf("concurrent stream read vs append-expected cross-talk: %s", msg)
	}
}

package sqliteengine_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	sqliteengine "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// openWALEngine builds a file-backed engine in WAL mode with a real
// connection pool: readers and writers can hold connections concurrently,
// which in-memory single-connection setups cannot reproduce.
func openWALEngine(t *testing.T) (metaengine.Engine, *sql.DB) {
	t.Helper()

	db, err := sql.Open(
		"sqlite",
		"file:"+filepath.Join(
			t.TempDir(),
			"tx_iso.db",
		)+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)",
	)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}

	db.SetMaxOpenConns(4)
	t.Cleanup(func() { _ = db.Close() })

	eng, err := sqliteengine.NewSQLiteEngine(db)
	if err != nil {
		t.Fatalf("NewSQLiteEngine: %v", err)
	}

	t.Cleanup(func() { metaengine.DeferClose(eng) })

	return eng, db
}

// TestSQLiteEngine_TxIsolationFromForeignContext pins the transaction
// visibility contract: a transaction is visible only to calls whose context
// descends from RunInTx's fn. A concurrent caller with its own context must
// never observe uncommitted writes, because transaction affinity must flow
// through the context — an engine-global "active tx" leaks the transaction
// to unrelated goroutines (dirty reads, and readers dying with
// "sql: Rows are closed" when the foreign tx commits mid-iteration).
func TestSQLiteEngine_TxIsolationFromForeignContext(t *testing.T) {
	t.Parallel()

	eng, _ := openWALEngine(t)

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

// TestSQLiteEngine_ConcurrentStreamReadVsAppendExpected reproduces the
// production flake behind CRM timeline loads: a StreamRead landing while a
// concurrent StreamAppendExpected transaction is open gets handed the
// foreign *sql.Tx by engine-global state; the tx commits and the reader's
// rows die with "sql: Rows are closed". Reads and writes on separate
// streams must be fully independent.
func TestSQLiteEngine_ConcurrentStreamReadVsAppendExpected(t *testing.T) {
	t.Parallel()

	eng, _ := openWALEngine(t)

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
		writes   = 400
		deadline = 5 * time.Second
	)

	var (
		wg      sync.WaitGroup
		failure atomic.Value // string
	)

	fail := func(who string, err error) {
		if err == nil {
			return
		}

		if prev, ok := failure.Load().(string); ok {
			_ = prev

			return
		}

		failure.Store(fmt.Sprintf("%s: %v", who, err))
	}

	ctx, cancel := context.WithTimeout(context.Background(), deadline)
	defer cancel()

	stop := make(chan struct{})
	var writerWG sync.WaitGroup

	for range writers {
		writerWG.Add(1)

		go func() {
			defer writerWG.Done()

			for i := range writes {
				if err := ctx.Err(); err != nil {
					return
				}

				if err := aa.StreamAppendExpected(ctx, writeCol, sid, int64(i),
					[]any{fmt.Sprintf("w-%04d", i)}); err != nil &&
					!errors.Is(err, metaengine.ErrVersionConflict) {
					// A deadline landing mid-append is shutdown, not failure —
					// the pre-loop check cannot cover the in-flight window.
					if ctx.Err() != nil {
						return
					}

					fail("writer StreamAppendExpected", err)

					return
				}
			}
		}()
	}

	go func() {
		writerWG.Wait()
		close(stop)
	}()

	for range readers {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for {
				select {
				case <-stop:
					return
				case <-ctx.Done():
					return
				default:
				}

				vals, err := sb.StreamRead(ctx, readCol, sid)
				if err != nil {
					// Deadline landing mid-read is shutdown, not failure: under
					// -race the writers can outlive the 5s budget and a reader
					// call in flight at expiry returns ctx.Err().
					if ctx.Err() != nil {
						return
					}

					fail("reader StreamRead", err)

					return
				}

				if len(vals) != len(seed) {
					fail(
						"reader StreamRead",
						fmt.Errorf("got %d values, want %d", len(vals), len(seed)),
					)

					return
				}
			}
		}()
	}

	wg.Wait()

	if msg, ok := failure.Load().(string); ok {
		t.Fatalf("concurrent stream read vs append: %s", msg)
	}
}

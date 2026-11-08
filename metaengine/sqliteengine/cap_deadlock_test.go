package sqliteengine_test

import (
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"

	sqliteengine "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

type capDeadlockEvent struct {
	ID  string
	Seq int
}

type capDeadlockItem struct {
	ID  string
	Seq int
}

// TestSQLiteEnforceCapDoesNotDeadlockAgainstFolds pins the metaengine
// lock-order fix: a cap trim must acquire the engine transaction (and with
// it the pooled connection) BEFORE the query's fold lock — the same
// RunInTx → foldLock order the fold path holds (dispatchFoldsCoreLocked
// applies each fold's lock INSIDE the engine transaction). The pre-fix trim
// held the fold lock across engine scans and deletes; on this one-connection
// pool a concurrent fold batch held the only connection inside RunInTx
// while waiting for the fold lock the trim held — an order-dependent
// permanent hang, observed in webphone's projection host as 60-600s stalls
// (~1-in-2 runs under a contacts+transcripts battery).
//
// The race loop makes the interleaving near-certain; the watchdog fails the
// test on a hang instead of sitting until the package timeout.
func TestSQLiteEnforceCapDoesNotDeadlockAgainstFolds(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite",
		fmt.Sprintf("file:sqliteengine_capdeadlock_%d?mode=memory&cache=shared", time.Now().UnixNano()))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	t.Cleanup(func() { _ = db.Close() })

	eng, err := sqliteengine.NewSQLiteEngine(db)
	if err != nil {
		t.Fatalf("NewSQLiteEngine: %v", err)
	}

	t.Cleanup(func() { _ = eng.Close() })

	query := metaengine.Query[struct{}, capDeadlockItem]("cap_deadlock_items",
		metaengine.OnRecord(capDeadlockEvent{}, func(_ record.Record, e capDeadlockEvent) (string, capDeadlockItem) {
			return e.ID, capDeadlockItem(e)
		}),
		metaengine.WithCap(50, "Seq"),
	)

	store, err := metaengine.Plan([]metaengine.Engine{eng}, query)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	t.Cleanup(func() { _ = store.Close() })

	ctx := t.Context()

	const writes = 400
	const trims = 60

	errs := make(chan error, 2)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		for i := 1; i <= writes; i++ {
			if err := store.Apply(ctx, "capDeadlockEvent", capDeadlockEvent{
				ID:  fmt.Sprintf("e%04d", i),
				Seq: i,
			}); err != nil {
				errs <- fmt.Errorf("Apply %d: %w", i, err)

				return
			}
		}
	}()

	go func() {
		defer wg.Done()

		for range trims {
			if _, err := store.EnforceCap(ctx, "cap_deadlock_items"); err != nil {
				errs <- fmt.Errorf("EnforceCap: %w", err)

				return
			}
		}
	}()

	finished := make(chan struct{})
	go func() { wg.Wait(); close(finished) }()

	select {
	case <-finished:
	case <-time.After(60 * time.Second):
		t.Fatal(
			"EnforceCap racing fold dispatches did not complete within 60s — fold-lock/connection lock-order regression",
		)
	}

	close(errs)

	for err := range errs {
		t.Error(err)
	}
}

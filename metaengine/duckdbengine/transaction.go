package duckdbengine

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// txMarker keys the context value carrying the active transaction.
type txMarker struct{}

// conn returns the ambient transaction when ctx descends from a RunInTx
// callback, otherwise the engine's *sql.DB. Transaction affinity flows
// through the context — NEVER engine-global state — so a transaction is
// visible only to the call tree that opened it. A concurrent caller with
// its own ctx always talks to the base pool and can neither observe
// uncommitted writes nor have its rows closed by a foreign tx commit
// (the engine-global activeTx produced exactly that cross-talk; ported
// from sqliteengine 22ab7b218, 2026-09-20).
func (e *duckdbEngine) conn(ctx context.Context) metaengine.SQLExec {
	if tx := txFromCtx(ctx); tx != nil {
		return tx
	}

	return e.db
}

// txFromCtx returns the transaction carried by ctx, or nil when ctx does
// not descend from a RunInTx callback.
func txFromCtx(ctx context.Context) *sql.Tx {
	tx, _ := ctx.Value(txMarker{}).(*sql.Tx)

	return tx
}

// RunInTx executes fn within a database transaction. If fn returns nil the
// transaction is committed; otherwise it is rolled back. Concurrent RunInTx
// calls (and all stream-append operations) are serialized via e.mu — only one
// transaction is active at a time.
//
// The transaction is carried in fn's context: every engine operation that
// receives a ctx descended from fn joins the transaction, everything else
// talks to the base pool. A nested RunInTx is rejected via the same marker.
func (e *duckdbEngine) RunInTx(ctx context.Context, fn func(context.Context) error) error {
	//art-dupl:accept same ctx-marker nested-tx rejection as sqliteengine — separate go.mod
	if ctx.Value(txMarker{}) != nil {
		return errors.New("duckdbengine.RunInTx: nested transactions are not supported")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("duckdbengine: begin tx: %w", err)
	}

	fnErr := fn(context.WithValue(ctx, txMarker{}, tx))

	if fnErr != nil { // art-dupl:accept cross-module tx error handling — separate go.mod
		_ = tx.Rollback()

		return fnErr
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("duckdbengine: commit tx: %w", err)
	}

	return nil
}

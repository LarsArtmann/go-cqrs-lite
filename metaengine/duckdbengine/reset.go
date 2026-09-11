package duckdbengine

import (
	"context"
	"database/sql"
	"fmt"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// resetBaseTables are the engine-owned storage tables. A reset clears their
// rows (never their schema) plus the planned tables registered via
// ApplyLayout/ApplyLayoutPlan — the layouts survive, mirroring the post-Plan
// state [metaengine.Store.Reset] reverts to.
var resetBaseTables = []string{
	"meta_map",
	"meta_counter",
	"meta_stream_log",
	"meta_graph_edges",
}

// ResetEngine implements [metaengine.EngineResetter]: it clears every
// engine-owned table (base tables plus all layout-planned tables) in one
// database transaction, returning the engine to its empty post-construction
// state so a journal replay rebuilds every collection from zero. Table
// schemas and layout declarations survive; the seq_stream_log sequence
// deliberately KEEPS advancing across a reset — sequence numbers must stay
// monotonic forever, so a consumer holding a pre-reset resumption token
// (journal seq > N) never skips replayed entries.
//
// Serialized against RunInTx and stream appends via mu; the single
// transaction makes the clear atomic (a partial reset cannot commit).
func (e *duckdbEngine) ResetEngine(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("duckdbengine.ResetEngine: begin: %w", err)
	}

	e.layoutMu.RLock()
	planned := make([]string, 0, len(e.plans))
	for _, plan := range e.plans {
		planned = append(planned, plan.Table)
	}
	e.layoutMu.RUnlock()

	tables := append(append([]string(nil), resetBaseTables...), planned...)

	for _, table := range tables {
		if _, err := tx.ExecContext(ctx, "DELETE FROM "+metaengine.QuoteIdent(table)); err != nil {
			return rollbackReturning(tx, fmt.Errorf("duckdbengine.ResetEngine: clear %s: %w", table, err))
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("duckdbengine.ResetEngine: commit: %w", err)
	}

	return nil
}

// rollbackReturning rolls a failed reset transaction back and returns the
// wrapping cause, so a partial clear can never commit.
func rollbackReturning(tx *sql.Tx, cause error) error {
	_ = tx.Rollback()

	return cause //nolint:wrapcheck // caller already wrapped the cause
}

// Compile-time assertion: the engine satisfies the reset capability.
var _ metaengine.EngineResetter = (*duckdbEngine)(nil)

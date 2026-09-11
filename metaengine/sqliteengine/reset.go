package sqliteengine

import (
	"context"
	"database/sql"
	"fmt"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// resetBaseTables are the base meta_* tables holding materialized state.
// Planned tables (from e.plans) are cleared alongside them, but their LAYOUT
// survives — [metaengine.Store.Reset] reverts the Store to its post-Plan
// state, and re-planning after a reset must land on the same tables.
var resetBaseTables = []string{
	"meta_map",
	"meta_set",
	"meta_counter",
	"meta_multimap",
	"meta_log",
	"meta_stream_log",
	"meta_graph_edges",
	"meta_snapshot",
}

// ResetEngine implements [metaengine.EngineResetter]: it drops ALL
// materialized state — every base meta_* table row, every planned-table row
// (the layout itself survives), and every materialized view (dropped and
// recreated against the emptied meta_map, because IVM DELETE-propagation is
// not trusted for grouped views — they carry known upstream maintenance
// defects) — and resets the cached multimap sequence counters, returning the
// engine to its empty post-construction state so a journal replay rebuilds
// every collection from zero.
//
// AUTOINCREMENT counters (meta_log.id, meta_stream_log.seq) deliberately keep
// advancing across a reset: journal positions must stay monotonic forever, so
// a consumer holding a pre-reset resumption token (seq > N) never skips
// replayed events.
//
// Serialized against RunInTx via txMu; a reset never interleaves with an
// active transaction.
func (e *sqliteEngine) ResetEngine(ctx context.Context) error {
	e.txMu.Lock()
	defer e.txMu.Unlock()

	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("sqliteengine.ResetEngine: begin: %w", err)
	}

	for _, table := range resetBaseTables {
		if _, err := tx.ExecContext(ctx, "DELETE FROM "+table); err != nil {
			return rollbackReturning(tx, fmt.Errorf("sqliteengine.ResetEngine: clear %s: %w", table, err))
		}
	}

	for col, plan := range e.plans {
		if _, err := tx.ExecContext(ctx, "DELETE FROM "+metaengine.QuoteIdent(plan.Table)); err != nil {
			return rollbackReturning(tx, fmt.Errorf(
				"sqliteengine.ResetEngine: clear planned table %s (collection %s): %w", plan.Table, col, err))
		}
	}

	for _, mv := range e.matViews {
		if _, err := tx.ExecContext(ctx,
			"DROP MATERIALIZED VIEW IF EXISTS "+metaengine.QuoteIdent(mv.name)); err != nil {
			return rollbackReturning(tx, fmt.Errorf(
				"sqliteengine.ResetEngine: drop materialized view %s: %w", mv.name, err))
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("sqliteengine.ResetEngine: commit: %w", err)
	}

	// Recreate the views against the now-empty meta_map. Recreating (instead
	// of trusting IVM to propagate the DELETE wave) is the only path that is
	// exactly-empty for every view shape on every driver version.
	if err := e.createMatViews(ctx); err != nil {
		return fmt.Errorf("sqliteengine.ResetEngine: %w", err)
	}

	// Cached multimap sequence counters are stale (high) after the delete;
	// drop them so the next use re-seeds from the emptied table.
	e.multiSeq.Range(func(key, _ any) bool {
		e.multiSeq.Delete(key)

		return true
	})

	return nil
}

// rollbackReturning rolls a failed reset transaction back and returns the
// wrapping cause, so a partial clear can never commit.
func rollbackReturning(tx *sql.Tx, cause error) error {
	_ = tx.Rollback()

	return cause //nolint:wrapcheck // caller already wrapped the cause
}

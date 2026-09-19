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
//
// The journal tables (meta_log, meta_stream_log) are deliberately ABSENT:
// journal entries are facts on the ADR-0136 invertibility ladder — the
// replay SOURCE, not derived data — so a reset must never delete them
// (ADR-0143; the fact journal meta_facts already followed this rule). A
// deployment hosting its event store on the engine relies on it.
var resetBaseTables = []string{
	"meta_map",
	"meta_set",
	"meta_counter",
	"meta_multimap",
	"meta_graph_edges",
	"meta_snapshot",
	"meta_vector",
	"meta_cell_versions",
	// ADR-0142 write-side collections (claimkit): timers and dedup windows
	// are derived data on the ADR-0136 ladder — reset + replay, so a reset
	// clears them like every other materialized collection.
	"meta_due_claims",
	"meta_dedup",
}

// ResetEngine implements [metaengine.EngineResetter]: it drops ALL
// materialized state — every base meta_* table row, every planned-table row
// (the layout itself survives), and every materialized view (dropped and
// recreated against the emptied meta_map, because IVM DELETE-propagation is
// not trusted for grouped views — they carry known upstream maintenance
// defects) — and resets the cached multimap sequence counters.
//
// The JOURNAL (meta_log, meta_stream_log) survives: journal entries are facts
// (ADR-0136 top rung / ADR-0143) — the replay source a reset rebuilds FROM,
// never derived state a reset clears. Only their AUTOINCREMENT counters'
// monotonicity matters to consumers, and rows surviving keeps positions
// stable by construction.
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
			return rollbackReturning(
				tx,
				fmt.Errorf("sqliteengine.ResetEngine: clear %s: %w", table, err),
			)
		}
	}

	for col, plan := range e.plans {
		if _, err := tx.ExecContext(
			ctx,
			"DELETE FROM "+metaengine.QuoteIdent(plan.Table),
		); err != nil {
			return rollbackReturning(tx, fmt.Errorf(
				"sqliteengine.ResetEngine: clear planned table %s (collection %s): %w",
				plan.Table,
				col,
				err,
			))
		}
	}

	for _, mv := range e.matViews {
		// libSQL drops materialized views with plain DROP VIEW (verified
		// against turso-go; DROP MATERIALIZED VIEW is a parse error there).
		if _, err := tx.ExecContext(ctx,
			"DROP VIEW IF EXISTS "+metaengine.QuoteIdent(mv.name)); err != nil {
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

	//art-dupl:accept intentional cross-module mirror — each dep-isolated engine module carries its own reset test/body (ADR-0136); see AGENTS.md #19
	return nil
}

// rollbackReturning rolls a failed reset transaction back and returns the
// wrapping cause, so a partial clear can never commit.
func rollbackReturning(tx *sql.Tx, cause error) error {
	_ = tx.Rollback()

	return cause //nolint:wrapcheck // caller already wrapped the cause
}

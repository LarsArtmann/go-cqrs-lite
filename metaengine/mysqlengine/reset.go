package mysqlengine

import (
	"context"
	"database/sql"
	"fmt"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// resetBaseTables are the engine-owned storage tables. A reset clears their
// rows (never their schema) plus the planned tables registered via
// ApplyLayout — the layouts survive, mirroring the post-Plan state
// [metaengine.Store.Reset] reverts to. The journal table (meta_stream_log)
// is deliberately ABSENT: journal entries are facts on the ADR-0136 ladder
// (ADR-0143) — the replay source, never derived data.
var resetBaseTables = []string{
	"meta_map",
	"meta_set",
	"meta_counter",
	"meta_graph_edges",
	"meta_vector",
	// ADR-0142 write-side collections (claimkit): timers and dedup windows
	// are derived data on the ADR-0136 ladder — reset + replay, so a reset
	// clears them like every other materialized collection.
	"meta_due_claims",
	"meta_dedup",
}

// ResetEngine implements [metaengine.EngineResetter]: it clears every
// engine-owned table (base tables plus all layout-planned tables) in one
// database transaction. Table schemas and layout declarations survive.
// The JOURNAL (meta_stream_log) survives — its rows are facts (ADR-0136 top
// rung / ADR-0143): the replay source a reset rebuilds FROM, never derived
// state a reset clears; its AUTO_INCREMENT keeps positions monotonic.
//
// DELETE FROM (not TRUNCATE) keeps the clear inside a transaction — MySQL
// TRUNCATE implicitly commits and could leave a half-reset behind on
// failure.
//
// Serialized against RunInTx via mu.
func (e *mysqlEngine) ResetEngine(ctx context.Context) error {
	//art-dupl:accept intentional cross-module mirror — each dep-isolated engine module carries its own reset test/body (ADR-0136); see AGENTS.md #19
	e.mu.Lock()
	defer e.mu.Unlock()

	e.layoutMu.Lock()
	planned := make([]string, 0, len(e.plans))
	for _, plan := range e.plans {
		planned = append(planned, plan.Table)
	}
	e.layoutMu.Unlock()

	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("mysqlengine.ResetEngine: begin: %w", err)
	}

	tables := append(append([]string(nil), resetBaseTables...), planned...)

	for _, table := range tables {
		// Table names are engine-owned and unreserved; no quoting needed
		// (MySQL backticks are only required for the reserved `key` column).
		if _, err := tx.ExecContext(ctx, "DELETE FROM "+table); err != nil {
			return rollbackReturning(
				tx,
				fmt.Errorf("mysqlengine.ResetEngine: clear %s: %w", table, err),
			)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("mysqlengine.ResetEngine: commit: %w", err)
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
var _ metaengine.EngineResetter = (*mysqlEngine)(nil)

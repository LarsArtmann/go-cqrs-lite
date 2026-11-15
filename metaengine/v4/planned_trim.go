package metaengine

import (
	"context"
	"errors"
	"fmt"
)

// PlannedTableSweeper is the age-based PLANNED-TABLE erasure capability:
// physical deletion of every row of a query's planned table whose extracted
// INTEGER column (addressed by JSON field name) is older than a cutoff.
//
// The same operator-maintenance doctrine as StreamTrimmer/JournalSweeper
// applies: never read/write-path behavior; the caller owns the domain
// semantics (retention policy cadence, replay implications) and this erases
// the bytes. Journal sweeps (JournalSweeper) and planned-table sweeps are
// INDEPENDENT halves of retention for collections that keep both: trimming
// the journal does not trim the planned projection and vice versa — call
// both when both exist.
//
// Replay contract: a planned-table trim does not append events. Rebuilding
// the projection (reset + replay) restores the full row set, so a replay
// must be followed by another trim pass — the same derived-state posture as
// EnforceCap (ADR-0136's invertibility ladder).
type PlannedTableSweeper interface {
	// TrimPlannedBefore deletes every row of the collection's planned table
	// whose field (JSON name of an extracted column) is numerically older
	// than cutoff (comparison direction: field < cutoff), and returns the
	// number of rows removed. The field must be declared as a filter/sort
	// field with an INTEGER column type; engines reject anything else — a
	// TEXT-typed time column compared numerically would trim arbitrary rows.
	TrimPlannedBefore(ctx context.Context, collection, field string, cutoff int64) (int64, error)
}

// errTrimEngineNotPlanned is returned when the sweep target has no engine
// assignment (unregistered query) — surfaced through the sentinel so callers
// can distinguish wiring bugs from data errors.
var errTrimEngineNotPlanned = errors.New("query has no engine assignment")

// TrimPlannedBefore sweeps one query's planned table by an extracted INTEGER
// column (see PlannedTableSweeper): the store resolves the query's engine and
// dispatches; the engine resolves its own table + column from the layout
// plan. Convenience over asserting the capability against an engine handle —
// stores own their engines privately, so callers cannot do the assertion
// themselves.
func (s *Store) TrimPlannedBefore(ctx context.Context, queryName, field string, cutoff int64) (int64, error) {
	q, ok := s.lookupQuery(queryName)
	if !ok {
		return 0, fmt.Errorf("%w: %q", ErrNoQueryForName, queryName)
	}

	eng := q.QueryEngine()
	if eng == nil {
		return 0, fmt.Errorf("metaengine.TrimPlannedBefore(%q): %w", queryName, errTrimEngineNotPlanned)
	}

	sweeper, ok := eng.(PlannedTableSweeper)
	if !ok {
		return 0, fmt.Errorf(
			"metaengine.TrimPlannedBefore(%q): engine %T does not support planned-table sweeps",
			queryName, eng,
		)
	}

	removed, err := sweeper.TrimPlannedBefore(ctx, queryName, field, cutoff)
	if err != nil {
		return removed, fmt.Errorf(
			"metaengine.TrimPlannedBefore(%q): %w", queryName, err,
		) //nolint:wrapcheck // single wrap site, engine errors already carry table context
	}

	return removed, nil
}

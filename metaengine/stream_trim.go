package metaengine

import (
	"context"
	"time"
)

// StreamTrimmer is the per-stream journal erasure capability (#118, GDPR
// erasure): physical deletion of one stream's journal rows and its snapshot.
//
// Trim methods are OPERATOR-INVOKED MAINTENANCE, never read/write-path
// behavior: the caller settles the domain semantics first — emit the
// deletion domain event, let the projections fold it, delete the blobs — and
// erases the journal bytes last. Doctrine stays tombstone-over-delete on the
// EVENT level (ADR-0114: deletion is a domain event); this capability exists
// for the layer below, where "delete must actually delete" is the invariant
// (GDPR erasure, retention sweeps) and a keep-forever journal is the bug.
//
// Replay contract: a trimmed stream folds to NOTHING afterwards (loads
// answer stream-not-found; replay never sees the rows). Projection
// checkpoints are position-based and stream-agnostic, so a checkpoint past
// trimmed rows simply never re-reads them.
type StreamTrimmer interface {
	// TrimStream deletes every journal row of one stream plus its snapshot
	// row, atomically, and returns the number of journal rows removed. The
	// stream afterwards behaves as never-written: the next append with
	// expected version 0 succeeds as a fresh stream.
	TrimStream(ctx context.Context, collection, streamID string) (int64, error)
}

// JournalSweeper is the age-based journal erasure capability (#118): physical
// deletion of journal rows older than a cutoff, across a collection.
//
// The operator owns the correctness precondition: remaining rows must still
// fold to honest state (true for wholly-dead streams, snapshot-free state
// rebuilds, and windowed aggregates whose evictions the decision already
// owns). Snapshots of streams the sweep removed rows from are co-deleted — a
// snapshot newer than a stream's remaining history would resurrect trimmed
// state on replay. Rows without a typed occurrence timestamp (legacy rows,
// NULL occurred_at) are conservatively KEPT.
type JournalSweeper interface {
	// TrimJournalBefore deletes every journal row whose occurred_at is older
	// than cutoff (plus the co-trimmed snapshots) and returns the number of
	// journal rows removed.
	TrimJournalBefore(ctx context.Context, collection string, cutoff time.Time) (int64, error)
}

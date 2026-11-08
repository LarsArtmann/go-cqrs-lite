package sqliteengine

import (
	"context"
	"fmt"
	"time"
)

// --- metaengine.StreamTrimmer + metaengine.JournalSweeper (#118) ---

// TrimStream deletes every meta_stream_log row of one stream plus its
// meta_snapshot row, atomically, and returns the number of journal rows
// removed. Implements metaengine.StreamTrimmer: the GDPR-erasure maintenance
// primitive — operator-invoked, never read/write-path (the deletion domain
// event and its folds settle the semantics first; this erases the bytes).
// Runs inside RunInTx, so it serializes against concurrent appends on the
// engine's transaction mutex instead of racing them.
func (e *sqliteEngine) TrimStream(ctx context.Context, col, sid string) (int64, error) {
	var removed int64

	err := e.RunInTx(ctx, func(ctx context.Context) error {
		res, err := e.xd(ctx).ExecContext(ctx,
			`DELETE FROM meta_stream_log WHERE collection = ? AND stream_id = ?`,
			col, sid)
		if err != nil {
			return fmt.Errorf("sqliteengine: trim stream %s/%s: %w", col, sid, err)
		}

		removed, err = res.RowsAffected()
		if err != nil {
			return fmt.Errorf("sqliteengine: trim stream %s/%s: rows affected: %w", col, sid, err)
		}

		if _, err := e.xd(ctx).ExecContext(ctx,
			`DELETE FROM meta_snapshot WHERE collection = ? AND stream_id = ?`,
			col, sid,
		); err != nil {
			return fmt.Errorf("sqliteengine: trim snapshot %s/%s: %w", col, sid, err)
		}

		return nil
	})
	if err != nil {
		return removed, err //nolint:wrapcheck // RunInTx errors already carry full context
	}

	return removed, nil
}

// TrimJournalBefore deletes every meta_stream_log row of the collection whose
// occurred_at (unix nanos) is older than cutoff, plus the snapshots of the
// streams it removed rows from — the snapshot subquery runs BEFORE the row
// delete in the same transaction, so it still sees the pre-sweep state.
// Implements metaengine.JournalSweeper. Rows without a typed occurred_at
// (legacy, NULL) are conservatively kept. occurred_at carries no index: the
// sweep is a maintenance scan, sized by journal width, not a hot path.
func (e *sqliteEngine) TrimJournalBefore(ctx context.Context, col string, cutoff time.Time) (int64, error) {
	var removed int64

	err := e.RunInTx(ctx, func(ctx context.Context) error {
		if _, err := e.xd(ctx).ExecContext(ctx,
			`DELETE FROM meta_snapshot WHERE collection = ? AND stream_id IN (
				SELECT DISTINCT stream_id FROM meta_stream_log
				WHERE collection = ? AND occurred_at IS NOT NULL AND occurred_at < ?
			)`,
			col, col, cutoff.UnixNano(),
		); err != nil {
			return fmt.Errorf("sqliteengine: sweep snapshots %s: %w", col, err)
		}

		res, err := e.xd(ctx).ExecContext(ctx,
			`DELETE FROM meta_stream_log
			WHERE collection = ? AND occurred_at IS NOT NULL AND occurred_at < ?`,
			col, cutoff.UnixNano())
		if err != nil {
			return fmt.Errorf("sqliteengine: sweep journal %s: %w", col, err)
		}

		removed, err = res.RowsAffected()
		if err != nil {
			return fmt.Errorf("sqliteengine: sweep journal %s: rows affected: %w", col, err)
		}

		return nil
	})
	if err != nil {
		return removed, err //nolint:wrapcheck // RunInTx errors already carry full context
	}

	return removed, nil
}

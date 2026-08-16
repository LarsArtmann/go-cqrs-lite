package pgengine

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// --- StreamLogBackend implementation ---

func (e *pgEngine) StreamAppend(ctx context.Context, col, sid string, values []any) error {
	if e.copyMin > 0 && len(values) >= e.copyMin && e.activeTx.Load() == nil {
		if err := e.copyAppend(ctx, col, sid, values); !errors.Is(err, errCopyUnavailable) {
			return err
		}
		// COPY unavailable (non-pgx driver): fall through to INSERTs.
	}

	return e.inTx(ctx, func(conn metaengine.SQLExec) error {
		return streamInsertBatch(ctx, conn, col, sid, values)
	})
}

// StreamAppendExpected appends values atomically if the stream version matches.
// Uses a transaction for true atomic version-check-then-append. When called
// inside RunInTx, participates in the outer transaction instead of starting
// a nested one.
// Returns metaengine.ErrVersionConflict if the current version does not match.
func (e *pgEngine) StreamAppendExpected(
	ctx context.Context,
	col, sid string,
	expectedVersion int64,
	values []any,
) error {
	return e.inTx(ctx, func(conn metaengine.SQLExec) error {
		var current int64

		err := conn.QueryRowContext(
			ctx,
			`SELECT COUNT(*) FROM meta_stream_log WHERE collection = $1 AND stream_id = $2`,
			col, sid,
		).Scan(&current)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("pgengine.StreamAppendExpected version: %w", err)
		}

		if current != expectedVersion {
			return metaengine.ErrVersionConflict
		}

		return streamInsertBatch(ctx, conn, col, sid, values)
	})
}

func (e *pgEngine) StreamRead(ctx context.Context, col, sid string) ([]any, error) {
	return e.scanStreamValues(ctx,
		`SELECT value FROM meta_stream_log WHERE collection = $1 AND stream_id = $2 ORDER BY seq`,
		col, sid)
}

func (e *pgEngine) StreamVersion(ctx context.Context, col, sid string) (int64, error) {
	var count int64

	err := e.conn().QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM meta_stream_log WHERE collection = $1 AND stream_id = $2`,
		col, sid,
	).Scan(&count)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}

		return 0, fmt.Errorf("pgengine.StreamVersion: %w", err)
	}

	return count, nil
}

func (e *pgEngine) JournalReadAll(ctx context.Context, col string) ([]any, error) {
	return e.scanStreamValues(ctx,
		`SELECT value FROM meta_stream_log WHERE collection = $1 ORDER BY seq`,
		col)
}

// JournalReadAllWithSeq returns every journal entry with its resume token
// (the BIGSERIAL seq). Implements metaengine.SeqSeekableStreamLog.
func (e *pgEngine) JournalReadAllWithSeq(
	ctx context.Context,
	col string,
) ([]metaengine.StreamLogEntry, error) {
	return e.scanStreamEntries(ctx,
		`SELECT seq, value FROM meta_stream_log WHERE collection = $1 ORDER BY seq`,
		col)
}

// JournalReadFromSeq returns up to limit entries with Seq > afterSeq via a
// pure index range seek on idx_stream_log_journal(collection, seq). Implements
// metaengine.SeqSeekableStreamLog. Unlike the OFFSET-based JournalReadFrom,
// resume cost is O(log n) per page and sequence gaps (interleaved collections,
// rolled-back inserts) cannot shift the cursor.
func (e *pgEngine) JournalReadFromSeq(
	ctx context.Context,
	col string,
	afterSeq int64,
	limit int,
) ([]metaengine.StreamLogEntry, error) {
	if limit <= 0 {
		return e.scanStreamEntries(
			ctx,
			`SELECT seq, value FROM meta_stream_log WHERE collection = $1 AND seq > $2 ORDER BY seq`,
			col,
			afterSeq,
		)
	}

	return e.scanStreamEntries(
		ctx,
		`SELECT seq, value FROM meta_stream_log WHERE collection = $1 AND seq > $2 ORDER BY seq LIMIT $3`,
		col,
		afterSeq,
		limit,
	)
}

func (e *pgEngine) JournalReadFrom(
	ctx context.Context,
	col string,
	afterSeq int64,
	limit int,
) ([]any, error) {
	// afterSeq is a journal POSITION within the collection, not a raw seq:
	// seq is a BIGSERIAL shared across collections, so filtering on seq values
	// re-delivers entries when collections interleave. Skip via OFFSET over
	// the collection-filtered, seq-ordered result instead.
	if limit <= 0 {
		return e.scanStreamValues(ctx,
			`SELECT value FROM meta_stream_log WHERE collection = $1 ORDER BY seq OFFSET $2`,
			col, afterSeq)
	}

	return e.scanStreamValues(
		ctx,
		`SELECT value FROM meta_stream_log WHERE collection = $1 ORDER BY seq LIMIT $2 OFFSET $3`,
		col,
		limit,
		afterSeq,
	)
}

func (e *pgEngine) scanStreamValues(
	ctx context.Context,
	query string,
	args ...any,
) ([]any, error) {
	rows, err := e.conn().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("pgengine.scanStreamValues: %w", err)
	}

	defer metaengine.DeferClose(rows)
	//art-dupl:accept cross-module SQL engine pattern — separate go.mod

	var result []any

	for rows.Next() {
		var valStr string
		if err := rows.Scan(&valStr); err != nil {
			return nil, fmt.Errorf("pgengine.scanStreamValues scan: %w", err)
		}

		result = append(result, metaengine.DecodeStreamValue(valStr))
	}

	if result == nil {
		result = []any{}
	}

	return result, rows.Err()
}

// scanStreamEntries executes a (seq, value) query and scans rows as journal
// entries carrying their resume tokens.
func (e *pgEngine) scanStreamEntries(
	ctx context.Context,
	query string,
	args ...any,
) ([]metaengine.StreamLogEntry, error) {
	rows, err := e.conn().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("pgengine.scanStreamEntries: %w", err)
	}

	defer metaengine.DeferClose(rows)
	//art-dupl:accept cross-module SQL engine pattern — separate go.mod

	var result []metaengine.StreamLogEntry

	for rows.Next() {
		var (
			seq    int64
			valStr string
		)
		if err := rows.Scan(&seq, &valStr); err != nil {
			return nil, fmt.Errorf("pgengine.scanStreamEntries scan: %w", err)
		}

		result = append(result, metaengine.StreamLogEntry{
			Seq:   seq,
			Value: metaengine.DecodeStreamValue(valStr),
		})
	}

	if result == nil {
		result = []metaengine.StreamLogEntry{}
	}

	return result, rows.Err()
}

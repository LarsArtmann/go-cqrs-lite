package mysqlengine

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// --- StreamLogBackend implementation ---
//art-dupl:accept cross-module SQL engine pattern — separate go.mod

func (e *mysqlEngine) StreamAppend(ctx context.Context, col, sid string, values []any) error {
	return e.inTx(ctx, func(conn metaengine.SQLExec) error {
		for _, v := range values {
			encoded := metaengine.EncodeStreamValue(v)
			if _, err := conn.ExecContext(
				ctx,
				`INSERT INTO meta_stream_log (collection, stream_id, value) VALUES (?, ?, ?)`,
				col, sid, encoded,
			); err != nil {
				return fmt.Errorf("mysqlengine.StreamAppend: %w", err)
			}
		}

		return nil
	})
}

// StreamAppendExpected appends values atomically if the stream version matches.
// Returns metaengine.ErrVersionConflict if the current version does not match.
func (e *mysqlEngine) StreamAppendExpected(
	ctx context.Context,
	col, sid string,
	expectedVersion int64,
	values []any,
) error {
	return e.inTx(ctx, func(conn metaengine.SQLExec) error {
		var current int64

		err := conn.QueryRowContext(
			ctx,
			`SELECT COUNT(*) FROM meta_stream_log WHERE collection = ? AND stream_id = ?`,
			col, sid,
		).Scan(&current)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("mysqlengine.StreamAppendExpected version: %w", err)
		}

		if current != expectedVersion {
			return metaengine.ErrVersionConflict
		}

		for _, v := range values {
			encoded := metaengine.EncodeStreamValue(v)
			if _, err := conn.ExecContext(
				ctx,
				`INSERT INTO meta_stream_log (collection, stream_id, value) VALUES (?, ?, ?)`,
				col, sid, encoded,
			); err != nil {
				return fmt.Errorf("mysqlengine.StreamAppendExpected insert: %w", err)
			}
		}

		return nil
	})
}

func (e *mysqlEngine) StreamRead(ctx context.Context, col, sid string) ([]any, error) {
	return e.scanStreamValues(ctx,
		`SELECT value FROM meta_stream_log WHERE collection = ? AND stream_id = ? ORDER BY seq`,
		col, sid)
}

func (e *mysqlEngine) StreamVersion(ctx context.Context, col, sid string) (int64, error) {
	var count int64

	err := e.conn().QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM meta_stream_log WHERE collection = ? AND stream_id = ?`,
		col, sid,
	).Scan(&count)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}

		return 0, fmt.Errorf("mysqlengine.StreamVersion: %w", err)
	}

	return count, nil
}

func (e *mysqlEngine) JournalReadAll(ctx context.Context, col string) ([]any, error) {
	return e.scanStreamValues(ctx,
		`SELECT value FROM meta_stream_log WHERE collection = ? ORDER BY seq`,
		col)
}

func (e *mysqlEngine) JournalReadFrom(
	ctx context.Context,
	col string,
	afterSeq int64,
	limit int,
) ([]any, error) {
	// afterSeq is a journal POSITION within the collection, not a raw seq:
	// seq is an AUTO_INCREMENT shared across collections, so filtering on seq
	// values re-delivers entries when collections interleave. Skip via OFFSET
	// over the collection-filtered, seq-ordered result instead. MySQL requires
	// a LIMIT clause before OFFSET; 2^64-1 is the conventional "no limit".
	if limit <= 0 {
		return e.scanStreamValues(ctx,
			`SELECT value FROM meta_stream_log WHERE collection = ? ORDER BY seq LIMIT 18446744073709551615 OFFSET ?`,
			col, afterSeq)
	}

	return e.scanStreamValues(
		ctx,
		`SELECT value FROM meta_stream_log WHERE collection = ? ORDER BY seq LIMIT ? OFFSET ?`,
		col,
		limit,
		afterSeq,
	)
}

func (e *mysqlEngine) scanStreamValues(
	ctx context.Context,
	query string,
	args ...any,
) ([]any, error) {
	rows, err := e.conn().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("mysqlengine.scanStreamValues: %w", err)
	}

	defer metaengine.DeferClose(rows)
	//art-dupl:accept cross-module SQL engine pattern — separate go.mod

	var result []any

	for rows.Next() {
		var valStr string
		if err := rows.Scan(&valStr); err != nil {
			return nil, fmt.Errorf("mysqlengine.scanStreamValues scan: %w", err)
		}

		result = append(result, metaengine.DecodeStreamValue(valStr))
	}

	if result == nil {
		result = []any{}
	}

	return result, rows.Err()
}

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
	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("pgengine.StreamAppend begin tx: %w", err)
	}

	defer func() { _ = tx.Rollback() }()

	for _, v := range values {
		encoded := metaengine.EncodeStreamValue(v)
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO meta_stream_log (collection, stream_id, value) VALUES ($1, $2, $3)`,
			col, sid, encoded,
		); err != nil {
			return fmt.Errorf("pgengine.StreamAppend: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("pgengine.StreamAppend commit: %w", err)
	}

	return nil
}

// StreamAppendExpected appends values atomically if the stream version matches.
// Uses a transaction for true atomic version-check-then-append.
// Returns metaengine.ErrVersionConflict if the current version does not match.
func (e *pgEngine) StreamAppendExpected(
	ctx context.Context,
	col, sid string,
	expectedVersion int64,
	values []any,
) error {
	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("pgengine.StreamAppendExpected begin tx: %w", err)
	}

	defer func() { _ = tx.Rollback() }()

	var current int64

	err = tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM meta_stream_log WHERE collection = $1 AND stream_id = $2`,
		col, sid,
	).Scan(&current)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("pgengine.StreamAppendExpected version: %w", err)
	}

	if current != expectedVersion {
		return metaengine.ErrVersionConflict
	}

	for _, v := range values {
		encoded := metaengine.EncodeStreamValue(v)
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO meta_stream_log (collection, stream_id, value) VALUES ($1, $2, $3)`,
			col, sid, encoded,
		); err != nil {
			return fmt.Errorf("pgengine.StreamAppendExpected insert: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("pgengine.StreamAppendExpected commit: %w", err)
	}

	return nil
}

func (e *pgEngine) StreamRead(ctx context.Context, col, sid string) ([]any, error) {
	return e.scanStreamValues(ctx,
		`SELECT value FROM meta_stream_log WHERE collection = $1 AND stream_id = $2 ORDER BY seq`,
		col, sid)
}

func (e *pgEngine) StreamVersion(ctx context.Context, col, sid string) (int64, error) {
	var count int64

	err := e.db.QueryRowContext(ctx,
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

func (e *pgEngine) JournalReadFrom(
	ctx context.Context,
	col string,
	afterSeq int64,
	limit int,
) ([]any, error) {
	if limit <= 0 {
		return e.scanStreamValues(ctx,
			`SELECT value FROM meta_stream_log WHERE collection = $1 AND seq > $2 ORDER BY seq`,
			col, afterSeq)
	}

	return e.scanStreamValues(ctx,
		`SELECT value FROM meta_stream_log WHERE collection = $1 AND seq > $2 ORDER BY seq LIMIT $3`,
		col, afterSeq, limit)
}

func (e *pgEngine) scanStreamValues(
	ctx context.Context,
	query string,
	args ...any,
) ([]any, error) {
	rows, err := e.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("pgengine.scanStreamValues: %w", err)
	}

	defer rows.Close()

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

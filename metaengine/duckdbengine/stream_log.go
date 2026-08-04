package duckdbengine

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"errors"
	"fmt"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// --- StreamLogBackend implementation ---

func (e *duckdbEngine) StreamAppend(ctx context.Context, col, sid string, values []any) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, v := range values {
		encoded := encodeDuckDBStreamValue(v)
		if _, err := e.db.ExecContext(ctx,
			`INSERT INTO meta_stream_log (collection, stream_id, value) VALUES ($1, $2, $3)`,
			col, sid, encoded,
		); err != nil {
			return fmt.Errorf("duckdbengine.StreamAppend: %w", err)
		}
	}

	return nil
}

// StreamAppendExpected appends values atomically if the stream version matches.
// The mutex serializes the version-check-then-append, eliminating the TOCTOU race.
// Returns metaengine.ErrVersionConflict if the current version does not match.
func (e *duckdbEngine) StreamAppendExpected(
	ctx context.Context,
	col, sid string,
	expectedVersion int64,
	values []any,
) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	var current int64

	err := e.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM meta_stream_log WHERE collection = $1 AND stream_id = $2`,
		col, sid,
	).Scan(&current)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("duckdbengine.StreamAppendExpected: %w", err)
	}

	if current != expectedVersion {
		return metaengine.ErrVersionConflict
	}

	for _, v := range values {
		encoded := encodeDuckDBStreamValue(v)
		if _, err := e.db.ExecContext(ctx,
			`INSERT INTO meta_stream_log (collection, stream_id, value) VALUES ($1, $2, $3)`,
			col, sid, encoded,
		); err != nil {
			return fmt.Errorf("duckdbengine.StreamAppendExpected insert: %w", err)
		}
	}

	return nil
}

func (e *duckdbEngine) StreamRead(ctx context.Context, col, sid string) ([]any, error) {
	return e.scanStreamValues(ctx,
		`SELECT value FROM meta_stream_log WHERE collection = $1 AND stream_id = $2 ORDER BY seq`,
		col, sid)
}

func (e *duckdbEngine) StreamVersion(ctx context.Context, col, sid string) (int64, error) {
	var count int64

	err := e.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM meta_stream_log WHERE collection = $1 AND stream_id = $2`,
		col, sid,
	).Scan(&count)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}

		return 0, fmt.Errorf("duckdbengine.StreamVersion: %w", err)
	}

	return count, nil
}

func (e *duckdbEngine) JournalReadAll(ctx context.Context, col string) ([]any, error) {
	return e.scanStreamValues(ctx,
		`SELECT value FROM meta_stream_log WHERE collection = $1 ORDER BY seq`,
		col)
}

func (e *duckdbEngine) JournalReadFrom(
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

func encodeDuckDBStreamValue(v any) string {
	if s, ok := v.(string); ok {
		return s
	}

	data, _ := json.Marshal(v)
	return string(data)
}

func decodeDuckDBStreamValue(s string) any {
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return s
	}

	return v
}

func (e *duckdbEngine) scanStreamValues(
	ctx context.Context,
	query string,
	args ...any,
) ([]any, error) {
	rows, err := e.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("duckdbengine.scanStreamValues: %w", err)
	}

	defer rows.Close()

	var result []any

	for rows.Next() {
		var valStr string
		if err := rows.Scan(&valStr); err != nil {
			return nil, fmt.Errorf("duckdbengine.scanStreamValues scan: %w", err)
		}

		result = append(result, decodeDuckDBStreamValue(valStr))
	}

	if result == nil {
		result = []any{}
	}

	return result, rows.Err()
}

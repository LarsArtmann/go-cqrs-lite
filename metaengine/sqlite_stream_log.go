package metaengine

import (
	"context"
	"database/sql"
	"errors"
)

// --- StreamLogBackend ---

func (e *sqliteEngine) StreamAppend(ctx context.Context, col, sid string, values []any) error {
	for _, v := range values {
		encoded := encodeStreamValue(v)
		if _, err := e.xc().exec(ctx, e.queries.streamAppend, col, sid, encoded); err != nil {
			return err
		}
	}

	return nil
}

func (e *sqliteEngine) StreamRead(ctx context.Context, col, sid string) ([]any, error) {
	return e.scanStreamValues(ctx, e.queries.streamRead, col, sid)
}

func (e *sqliteEngine) StreamVersion(ctx context.Context, col, sid string) (int64, error) {
	var count int64

	err := e.xc().queryRow(ctx, e.queries.streamVersion, col, sid).Scan(&count)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}

		return 0, err //nolint:wrapcheck // passthrough
	}

	return count, nil
}

func (e *sqliteEngine) JournalReadAll(ctx context.Context, col string) ([]any, error) {
	return e.scanStreamValues(ctx, e.queries.journalReadAll, col)
}

func (e *sqliteEngine) JournalReadFrom(
	ctx context.Context,
	col string,
	afterSeq int64,
	limit int,
) ([]any, error) {
	if limit <= 0 {
		return e.scanStreamValues(ctx,
			`SELECT value FROM meta_stream_log WHERE collection = ? AND seq > ? ORDER BY seq`,
			col, afterSeq)
	}

	return e.scanStreamValues(ctx, e.queries.journalReadFrom, col, afterSeq, limit)
}

// StreamAppendExpected appends values atomically if the stream version matches.
// Uses RunInTx for true transactional isolation.
func (e *sqliteEngine) StreamAppendExpected(
	ctx context.Context,
	col, sid string,
	expectedVersion int64,
	values []any,
) error {
	return e.RunInTx(ctx, func(ctx context.Context) error {
		var current int64

		err := e.xc().queryRow(ctx, e.queries.streamVersion, col, sid).Scan(&current)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err //nolint:wrapcheck // passthrough
		}

		if current != expectedVersion {
			return ErrVersionConflict
		}

		for _, v := range values {
			encoded := encodeStreamValue(v)
			if _, err := e.xc().exec(ctx, e.queries.streamAppend, col, sid, encoded); err != nil {
				return err
			}
		}

		return nil
	})
}

// encodeStreamValue serializes a value for storage in the stream log TEXT column.
// Strings are stored as-is; all other types are JSON-encoded via encodeJSON.
func encodeStreamValue(v any) string {
	if s, ok := v.(string); ok {
		return s
	}

	return encodeJSON(v)
}

// scanStreamValues executes a query and scans all rows as stream log values.
// Each value is decoded from its stored representation (string passthrough or
// JSON decode).
func (e *sqliteEngine) scanStreamValues(
	ctx context.Context,
	query string,
	args ...any,
) ([]any, error) {
	rows, err := e.xd().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err //nolint:wrapcheck // passthrough
	}

	defer rows.Close()

	var result []any

	for rows.Next() {
		var valStr string
		if err := rows.Scan(&valStr); err != nil {
			return nil, err //nolint:wrapcheck // passthrough
		}

		result = append(result, decodeJSONValue(valStr))
	}

	if result == nil {
		result = []any{}
	}

	return result, rows.Err() //nolint:wrapcheck // passthrough
}

// Compile-time assertions for sqliteEngine.
var (
	_ StreamLogBackend = (*sqliteEngine)(nil)
	_ AtomicAppender   = (*sqliteEngine)(nil)
)

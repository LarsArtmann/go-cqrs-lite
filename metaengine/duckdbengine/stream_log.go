package duckdbengine

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// --- StreamLogBackend implementation ---

func (e *duckdbEngine) StreamAppend(ctx context.Context, col, sid string, values []any) error {
	if e.activeTx.Load() == nil {
		e.mu.Lock()
		defer e.mu.Unlock()
	}

	for _, v := range values {
		encoded := metaengine.EncodeStreamValue(v)
		if _, err := e.conn().ExecContext(
			ctx,
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
	if e.activeTx.Load() == nil {
		e.mu.Lock()
		defer e.mu.Unlock()
	}

	var current int64

	err := e.conn().QueryRowContext(
		ctx,
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
		encoded := metaengine.EncodeStreamValue(v)
		if _, err := e.conn().ExecContext(
			ctx,
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
	//art-dupl:accept cross-module SQL engine pattern — dep-isolated go.mod modules
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

		return 0, fmt.Errorf("duckdbengine.StreamVersion: %w", err)
	}

	return count, nil
}

func (e *duckdbEngine) JournalReadAll(ctx context.Context, col string) ([]any, error) {
	return e.scanStreamValues(ctx,
		`SELECT value FROM meta_stream_log WHERE collection = $1 ORDER BY seq`,
		col)
}

// JournalReadAllWithSeq returns every journal entry with its resume token
// (the seq_stream_log SEQUENCE value). Implements
// metaengine.SeqSeekableStreamLog.
func (e *duckdbEngine) JournalReadAllWithSeq(
	ctx context.Context,
	col string,
) ([]metaengine.StreamLogEntry, error) {
	return e.scanStreamEntries(ctx,
		`SELECT seq, value FROM meta_stream_log WHERE collection = $1 ORDER BY seq`,
		col)
}

// JournalReadFromSeq returns up to limit entries with Seq > afterSeq via a
// pure index range seek on the (collection, seq) primary key ordering.
// Implements metaengine.SeqSeekableStreamLog. Unlike the OFFSET-based
// JournalReadFrom, resume cost is O(log n) per page and sequence gaps
// (interleaved collections, rolled-back inserts) cannot shift the cursor.
func (e *duckdbEngine) JournalReadFromSeq(
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

func (e *duckdbEngine) JournalReadFrom(
	ctx context.Context,
	col string,
	afterSeq int64,
	limit int,
) ([]any, error) {
	// afterSeq is a journal POSITION within the collection, not a raw seq:
	// seq is shared across collections, so filtering on seq values re-delivers
	// entries when collections interleave. Skip via OFFSET over the
	// collection-filtered, seq-ordered result instead.
	if limit <= 0 {
		return e.scanStreamValues(
			ctx,
			`SELECT value FROM meta_stream_log WHERE collection = $1 ORDER BY seq LIMIT ALL OFFSET $2`,
			col,
			afterSeq,
		)
	}

	return e.scanStreamValues(
		ctx,
		`SELECT value FROM meta_stream_log WHERE collection = $1 ORDER BY seq LIMIT $2 OFFSET $3`,
		col,
		limit,
		afterSeq,
	)
}

func (e *duckdbEngine) scanStreamValues(
	ctx context.Context,
	query string,
	args ...any,
) ([]any, error) {
	rows, err := e.conn().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("duckdbengine.scanStreamValues: %w", err)
	}

	defer metaengine.DeferClose(rows)
	//art-dupl:accept cross-module SQL engine pattern — separate go.mod

	var result []any

	for rows.Next() {
		var valStr string
		if err := rows.Scan(&valStr); err != nil {
			return nil, fmt.Errorf("duckdbengine.scanStreamValues scan: %w", err)
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
func (e *duckdbEngine) scanStreamEntries(
	ctx context.Context,
	query string,
	args ...any,
) ([]metaengine.StreamLogEntry, error) {
	rows, err := e.conn().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("duckdbengine.scanStreamEntries: %w", err)
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
			return nil, fmt.Errorf("duckdbengine.scanStreamEntries scan: %w", err)
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

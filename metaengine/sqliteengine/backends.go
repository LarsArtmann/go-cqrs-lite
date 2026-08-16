package sqliteengine

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"sync"
	"sync/atomic"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// --- metaengine.SetBackend ---

func (e *sqliteEngine) SetAdd(ctx context.Context, col string, key any) error {
	_, err := e.xc().exec(ctx, e.queries.setAdd, col, encodeKey(key))

	return err
}

func (e *sqliteEngine) SetContains(ctx context.Context, col string, key any) (bool, error) {
	var one int

	err := e.xc().queryRow(ctx, e.queries.setContains, col, encodeKey(key)).Scan(&one)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}

		return false, err //nolint:wrapcheck // passthrough
	}

	return true, nil
}

// --- metaengine.CounterBackend ---

func (e *sqliteEngine) CounterIncrement(
	ctx context.Context,
	col string,
	deltas metaengine.Delta,
) error {
	// When inside an outer transaction, reuse its executor.
	if e.txExec() != nil {
		xc := e.xc()
		for k, d := range deltas {
			if _, err := xc.exec(ctx, e.queries.counterIncrement, col, k, d); err != nil {
				return err
			}
		}

		return nil
	}

	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return err //nolint:wrapcheck // passthrough
	}

	defer func() { _ = tx.Rollback() }()

	for k, d := range deltas {
		if _, err := tx.ExecContext(ctx, e.queries.counterIncrement, col, k, d); err != nil {
			return err //nolint:wrapcheck // passthrough
		}
	}

	return tx.Commit() //nolint:wrapcheck // passthrough
}

func (e *sqliteEngine) CounterGet(ctx context.Context, col string) (map[string]int64, error) {
	rows, err := e.xd().QueryContext(ctx, e.queries.counterGet, col) //nolint:sqlclosecheck
	if err != nil {
		return nil, err //nolint:wrapcheck // passthrough
	}

	defer metaengine.DeferClose(rows)

	result := make(map[string]int64)

	for rows.Next() {
		var k string

		var v int64

		if err := rows.Scan(&k, &v); err != nil {
			return nil, err //nolint:wrapcheck // passthrough
		}

		result[k] = v
	}

	return result, rows.Err() //nolint:wrapcheck // passthrough
}

// scanJSONValues executes query with args, then walks the returned single-column
// JSON-encoded rows, decoding each into an any. Rows that fail JSON decoding
// fall back to their raw string form. Used by PushdownMapScan, MultiGet, and
// LogTail — all paths where direct callers expect decoded values.
func scanJSONValues(
	ctx context.Context,
	db metaengine.SQLExec,
	query string,
	args ...any,
) ([]any, error) {
	return scanSingleColumn(ctx, db, query, metaengine.DecodeStreamValue, args...)
}

// --- metaengine.MultimapBackend ---

func (e *sqliteEngine) MultiAdd(ctx context.Context, col string, key any, value any) error {
	seq, err := e.nextMultiSeq(ctx, col)
	if err != nil {
		return err
	}

	_, err = e.xc().exec(
		ctx,
		e.queries.multiAdd,
		col,
		encodeKey(key),
		seq,
		encodeValue(value),
	)

	return err
}

func (e *sqliteEngine) MultiGet(ctx context.Context, col string, key any) ([]any, error) {
	return scanJSONValues(ctx, e.xd(), e.queries.multiGet, col, encodeKey(key))
}

// multiSeqCounter is a lazily-initialized sequence counter for a multimap
// collection. On first use after process start, sync.Once gates a DB read
// that seeds the counter from MAX(seq), preventing PK collisions with rows
// persisted by a previous process. All subsequent calls hit the atomic fast
// path with no DB access.
//
// The trailing pad pushes the allocation to the 128-byte size class so two
// per-collection counters can never share a cache line — Go's small-object
// allocator otherwise packs 32-byte objects 16-per-512B span and two hot
// multimap collections false-share (measured 2.2-2.8x Add slowdown,
// docs/benchmarks/2026-08-16_false-sharing-contention.md). 128 covers
// 128-byte-line ARM cores as well as the 64-byte x86 line.
type multiSeqCounter struct {
	once    sync.Once
	counter atomic.Int64
	initErr error
	_       [96]byte
}

func (e *sqliteEngine) nextMultiSeq(ctx context.Context, col string) (int64, error) {
	actual, _ := e.multiSeq.LoadOrStore(col, &multiSeqCounter{})
	c := actual.(*multiSeqCounter)

	c.once.Do(func() {
		var maxSeq sql.NullInt64

		queryErr := e.xd().QueryRowContext(ctx,
			"SELECT MAX(seq) FROM meta_multimap WHERE collection = ?", col).Scan(&maxSeq)
		if queryErr != nil {
			c.initErr = queryErr

			return
		}

		if maxSeq.Valid {
			c.counter.Store(maxSeq.Int64)
		}
	})

	if c.initErr != nil {
		return 0, c.initErr
	}

	return c.counter.Add(1), nil
}

// --- metaengine.LogBackend ---

func (e *sqliteEngine) LogAppend(ctx context.Context, col string, value any) error {
	_, err := e.xc().exec(ctx, e.queries.logAppend, col, encodeValue(value))

	return err
}

func (e *sqliteEngine) LogTail(ctx context.Context, col string, limit int) ([]any, error) {
	if limit <= 0 {
		limit = -1
	}

	// Query is DESC; reverse the result for chronological order.
	fwd, err := scanJSONValues(ctx, e.xd(), e.queries.logTail, col, limit)
	if err != nil {
		return nil, err
	}

	slices.Reverse(fwd)

	return fwd, nil
}

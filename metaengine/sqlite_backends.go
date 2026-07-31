package metaengine

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"errors"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"
)

// --- SetBackend ---

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

// --- CounterBackend ---

func (e *sqliteEngine) CounterIncrement(ctx context.Context, col string, deltas Delta) error {
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
	rows, err := e.xd().QueryContext(ctx, e.queries.counterGet, col)
	if err != nil {
		return nil, err //nolint:wrapcheck // passthrough
	}

	defer func() { _ = rows.Close() }()

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
func scanJSONValues(ctx context.Context, db dbExec, query string, args ...any) ([]any, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err //nolint:wrapcheck // passthrough
	}

	defer func() { _ = rows.Close() }()

	var out []any

	for rows.Next() {
		var valStr string

		if err := rows.Scan(&valStr); err != nil {
			return nil, err //nolint:wrapcheck // passthrough
		}

		out = append(out, decodeJSONValue(valStr))
	}

	return out, rows.Err() //nolint:wrapcheck // passthrough
}

// decodeJSONValue unmarshals a JSON string into an any. If the string is not
// valid JSON, it returns the raw string. Used by scanJSONValues and MapScan
// where Go-side filter/sort functions need decoded values.
func decodeJSONValue(valStr string) any {
	var val any

	if jErr := json.Unmarshal([]byte(valStr), &val); jErr != nil {
		return valStr
	}

	return val
}

// --- MultimapBackend ---

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
type multiSeqCounter struct {
	once    sync.Once
	counter atomic.Int64
	initErr error
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

// --- LogBackend ---

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

// --- GraphBackend ---

func (e *sqliteEngine) GraphAddEdge(ctx context.Context, col string, edge Edge) error {
	from := encodeKey(edge.From)
	to := encodeKey(edge.To)

	_, err := e.xd().ExecContext(ctx, e.queries.graphAddEdge, col, from, to)

	return err //nolint:wrapcheck // passthrough
}

func (e *sqliteEngine) GraphNeighbors(
	ctx context.Context,
	col string,
	node any,
	depth int,
) ([]any, error) {
	// Simple BFS in Go (avoids recursive CTE complexity across dialects).
	visited := map[string]bool{encodeKey(node): true}
	frontier := []any{node}
	result := []any{}

	for range depth {
		if len(frontier) == 0 {
			break
		}

		var next []any

		for _, n := range frontier {
			keys, err := e.scanNeighborKeys(ctx, col, encodeKey(n))
			if err != nil {
				return nil, fmt.Errorf("scan neighbor keys: %w", err)
			}

			for _, toKey := range keys {
				if visited[toKey] {
					continue
				}

				visited[toKey] = true

				var neighbor any

				if jErr := json.Unmarshal([]byte(toKey), &neighbor); jErr != nil {
					neighbor = toKey
				}

				result = append(result, neighbor)
				next = append(next, neighbor)
			}
		}

		frontier = next
	}

	return result, nil
}

// scanNeighborKeys returns the raw to_node keys for a single from-node. Extracted
// from GraphNeighbors so rows.Close is handled by defer (sqlclosecheck) rather
// than manual close calls inside a loop.
func (e *sqliteEngine) scanNeighborKeys(
	ctx context.Context,
	col, fromKey string,
) ([]string, error) {
	rows, err := e.xd().QueryContext(
		ctx,
		`SELECT to_node FROM meta_graph_edges WHERE collection = ? AND from_node = ?`,
		col, fromKey,
	)
	if err != nil {
		return nil, err //nolint:wrapcheck // passthrough
	}

	defer func() { _ = rows.Close() }()

	var keys []string

	for rows.Next() {
		var toKey string

		if err := rows.Scan(&toKey); err != nil {
			return nil, err //nolint:wrapcheck // passthrough
		}

		keys = append(keys, toKey)
	}

	return keys, rows.Err() //nolint:wrapcheck // passthrough
}

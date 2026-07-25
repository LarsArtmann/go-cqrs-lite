package metaengine

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

// sqliteEngine implements all ADT backends backed by a SQL database.
// It is the first persistent engine for the metaengine, enabling data
// to survive process restarts.
type sqliteEngine struct {
	db      *sql.DB
	queries sqliteQuerySet
	// seq counters for multimap and log (SQLite AUTOINCREMENT handles log).
	multiSeq sync.Map // collection→*atomic.Int64
}

// sqliteQuerySet holds pre-built SQL strings for each operation.
// Two variants handle placeholder differences (? vs $N).
type sqliteQuerySet struct {
	// Map
	mapSet    string
	mapGet    string
	mapDelete string
	// Set
	setAdd      string
	setContains string
	// Counter
	counterIncrement string
	counterGet       string
	// Multimap
	multiAdd string
	multiGet string
	// Log
	logAppend string
	logTail   string
	// Graph
	graphAddEdge   string
	graphNeighbors string
	// DDL
	ddl string
}

func defaultSQLiteQueries() sqliteQuerySet {
	return sqliteQuerySet{
	ddl: `CREATE TABLE IF NOT EXISTS meta_map (
		collection TEXT NOT NULL, key TEXT NOT NULL, value TEXT NOT NULL,
		PRIMARY KEY (collection, key)
	);
	CREATE TABLE IF NOT EXISTS meta_set (
		collection TEXT NOT NULL, key TEXT NOT NULL,
		PRIMARY KEY (collection, key)
	);
	CREATE TABLE IF NOT EXISTS meta_counter (
		collection TEXT NOT NULL, key TEXT NOT NULL, value INTEGER NOT NULL DEFAULT 0,
		PRIMARY KEY (collection, key)
	);
	CREATE TABLE IF NOT EXISTS meta_multimap (
		collection TEXT NOT NULL, key TEXT NOT NULL, seq INTEGER NOT NULL, value TEXT NOT NULL,
		PRIMARY KEY (collection, key, seq)
	);
	CREATE TABLE IF NOT EXISTS meta_log (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		collection TEXT NOT NULL, value TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS meta_graph_edges (
		collection TEXT NOT NULL, from_node TEXT NOT NULL, to_node TEXT NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_graph_from ON meta_graph_edges(collection, from_node);
	CREATE INDEX IF NOT EXISTS idx_graph_to ON meta_graph_edges(collection, to_node);`,
	mapSet:           `INSERT OR REPLACE INTO meta_map (collection, key, value) VALUES (?, ?, ?)`,
	mapGet:           `SELECT value FROM meta_map WHERE collection = ? AND key = ?`,
	mapDelete:        `DELETE FROM meta_map WHERE collection = ? AND key = ?`,
	setAdd:           `INSERT OR IGNORE INTO meta_set (collection, key) VALUES (?, ?)`,
	setContains:      `SELECT 1 FROM meta_set WHERE collection = ? AND key = ?`,
	counterIncrement: `INSERT INTO meta_counter (collection, key, value) VALUES (?, ?, ?) ON CONFLICT(collection, key) DO UPDATE SET value = value + excluded.value`,
	counterGet:       `SELECT key, value FROM meta_counter WHERE collection = ?`,
	multiAdd:         `INSERT INTO meta_multimap (collection, key, seq, value) VALUES (?, ?, ?, ?)`,
	multiGet:         `SELECT value FROM meta_multimap WHERE collection = ? AND key = ? ORDER BY seq`,
	logAppend:        `INSERT INTO meta_log (collection, value) VALUES (?, ?)`,
	logTail:          `SELECT value FROM meta_log WHERE collection = ? ORDER BY id DESC LIMIT ?`,
	graphAddEdge:     `INSERT INTO meta_graph_edges (collection, from_node, to_node) VALUES (?, ?, ?)`,
	graphNeighbors:   `WITH RECURSIVE bfs(depth, node) AS (SELECT 0, ? UNION ALL SELECT bfs.depth + 1, e.to_node FROM meta_graph_edges e JOIN bfs ON e.from_node = bfs.node AND e.collection = ? WHERE bfs.depth < ?) SELECT DISTINCT node FROM bfs WHERE node != ?`,
	}
}

// NewSQLiteEngine creates a SQLite-backed metaengine engine. The caller owns
// the *sql.DB. Tables are created automatically if they don't exist.
func NewSQLiteEngine(db *sql.DB) (Engine, error) {
	eng := &sqliteEngine{
		db:      db,
		queries: defaultSQLiteQueries(),
	}

	if _, err := db.ExecContext(context.Background(), eng.queries.ddl); err != nil {
		return nil, fmt.Errorf("metaengine: create tables: %w", err)
	}

	return eng, nil
}

func (e *sqliteEngine) Profile() EngineProfile {
	return SQLiteEngineProfile()
}

func (e *sqliteEngine) Close() error { return nil }

func encodeKey(key any) string {
	b, err := json.Marshal(key)
	if err != nil {
		return fmt.Sprintf("%v", key)
	}

	return string(b)
}

func encodeValue(value any) string {
	b, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%v", value)
	}

	return string(b)
}

// --- MapBackend ---

func (e *sqliteEngine) MapSet(ctx context.Context, col string, key any, value any) error {
	_, err := e.db.ExecContext(ctx, e.queries.mapSet, col, encodeKey(key), encodeValue(value))

	return err //nolint:wrapcheck // passthrough
}

func (e *sqliteEngine) MapGet(ctx context.Context, col string, key any) (any, bool, error) {
	var valStr string

	err := e.db.QueryRowContext(ctx, e.queries.mapGet, col, encodeKey(key)).Scan(&valStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}

		return nil, false, err //nolint:wrapcheck // passthrough
	}

	var val any

	if jErr := json.Unmarshal([]byte(valStr), &val); jErr != nil {
		val = valStr
	}

	return val, true, nil
}

func (e *sqliteEngine) MapDelete(ctx context.Context, col string, key any) error {
	_, err := e.db.ExecContext(ctx, e.queries.mapDelete, col, encodeKey(key))

	return err //nolint:wrapcheck // passthrough
}

// --- MapUpdater ---

func (e *sqliteEngine) MapUpdate(
	ctx context.Context,
	col string,
	key any,
	update func(prev any) any,
) error {
	prev, exists, err := e.MapGet(ctx, col, key)
	if err != nil {
		return err
	}

	if !exists {
		prev = nil
	}

	newVal := update(prev)

	return e.MapSet(ctx, col, key, newVal)
}

// --- ScanBackend ---

func (e *sqliteEngine) MapScan(
	ctx context.Context,
	col string,
	filters []filterPredicate,
	sortFunc func(a, b any) int,
	cursor any,
	limit int,
) ([]any, error) {
	rows, err := e.db.Query(`SELECT value FROM meta_map WHERE collection = ?`, col)
	if err != nil {
		return nil, err //nolint:wrapcheck // passthrough
	}

	defer func() { _ = rows.Close() }()

	type kv struct {
		key   string
		value any
	}

	var pairs []kv

	for rows.Next() {
		var valStr string

		if err := rows.Scan(&valStr); err != nil {
			return nil, err //nolint:wrapcheck // passthrough
		}

		var val any

		if jErr := json.Unmarshal([]byte(valStr), &val); jErr != nil {
			val = valStr
		}

		if !passesFilters(val, filters) {
			continue
		}

		pairs = append(pairs, kv{key: valStr, value: val})
	}

	if err := rows.Err(); err != nil {
		return nil, err //nolint:wrapcheck // passthrough
	}

	sort.Slice(pairs, func(i, j int) bool {
		if sortFunc != nil {
			if c := sortFunc(pairs[i].value, pairs[j].value); c != 0 {
				return c < 0
			}
		}

		return strings.Compare(pairs[i].key, pairs[j].key) < 0
	})

	if cursor != nil && sortFunc != nil {
		filtered := pairs[:0]

		for _, p := range pairs {
			if sortFunc(p.value, cursor) <= 0 {
				continue
			}

			filtered = append(filtered, p)
		}

		pairs = filtered
	}

	truncLimit := 0
	if limit > 0 {
		truncLimit = limit + 1
	}

	if truncLimit > 0 && len(pairs) > truncLimit {
		pairs = pairs[:truncLimit]
	}

	results := make([]any, len(pairs))
	for i, p := range pairs {
		results[i] = p.value
	}

	return results, nil
}

// --- SetBackend ---

func (e *sqliteEngine) SetAdd(ctx context.Context, col string, key any) error {
	_, err := e.db.ExecContext(ctx, e.queries.setAdd, col, encodeKey(key))

	return err //nolint:wrapcheck // passthrough
}

func (e *sqliteEngine) SetContains(ctx context.Context, col string, key any) (bool, error) {
	var one int

	err := e.db.QueryRowContext(ctx, e.queries.setContains, col, encodeKey(key)).Scan(&one)
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
	rows, err := e.db.Query(e.queries.counterGet, col)
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

// --- MultimapBackend ---

func (e *sqliteEngine) MultiAdd(ctx context.Context, col string, key any, value any) error {
	seq := e.nextMultiSeq(col)

	_, err := e.db.ExecContext(ctx, e.queries.multiAdd, col, encodeKey(key), seq, encodeValue(value))

	return err //nolint:wrapcheck // passthrough
}

func (e *sqliteEngine) MultiGet(ctx context.Context, col string, key any) ([]any, error) {
	rows, err := e.db.Query(e.queries.multiGet, col, encodeKey(key))
	if err != nil {
		return nil, err //nolint:wrapcheck // passthrough
	}

	defer func() { _ = rows.Close() }()

	var result []any

	for rows.Next() {
		var valStr string

		if err := rows.Scan(&valStr); err != nil {
			return nil, err //nolint:wrapcheck // passthrough
		}

		var val any

		if jErr := json.Unmarshal([]byte(valStr), &val); jErr != nil {
			val = valStr
		}

		result = append(result, val)
	}

	return result, rows.Err() //nolint:wrapcheck // passthrough
}

func (e *sqliteEngine) nextMultiSeq(col string) int64 {
	actual, _ := e.multiSeq.LoadOrStore(col, &atomic.Int64{})

	return actual.(*atomic.Int64).Add(1) //nolint:forcetypeassert // stored as *atomic.Int64
}

// --- LogBackend ---

func (e *sqliteEngine) LogAppend(ctx context.Context, col string, value any) error {
	_, err := e.db.ExecContext(ctx, e.queries.logAppend, col, encodeValue(value))

	return err //nolint:wrapcheck // passthrough
}

func (e *sqliteEngine) LogTail(ctx context.Context, col string, limit int) ([]any, error) {
	if limit <= 0 {
		limit = -1
	}

	rows, err := e.db.Query(e.queries.logTail, col, limit)
	if err != nil {
		return nil, err //nolint:wrapcheck // passthrough
	}

	defer func() { _ = rows.Close() }()

	var fwd []any

	for rows.Next() {
		var valStr string

		if err := rows.Scan(&valStr); err != nil {
			return nil, err //nolint:wrapcheck // passthrough
		}

		var val any

		if jErr := json.Unmarshal([]byte(valStr), &val); jErr != nil {
			val = valStr
		}

		fwd = append(fwd, val)
	}

	if err := rows.Err(); err != nil {
		return nil, err //nolint:wrapcheck // passthrough
	}

	// Reverse to get chronological order (query was DESC).
	slices.Reverse(fwd)

	return fwd, nil
}

// --- GraphBackend ---

func (e *sqliteEngine) GraphAddEdge(ctx context.Context, col string, edge Edge) error {
	from := encodeKey(edge.From)
	to := encodeKey(edge.To)

	_, err := e.db.ExecContext(ctx, e.queries.graphAddEdge, col, from, to)

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

	for d := 0; d < depth && len(frontier) > 0; d++ {
		var next []any

		for _, n := range frontier {
			nKey := encodeKey(n)

			rows, err := e.db.Query(
				`SELECT to_node FROM meta_graph_edges WHERE collection = ? AND from_node = ?`,
				col, nKey,
			)
			if err != nil {
				return nil, err //nolint:wrapcheck // passthrough
			}

			for rows.Next() {
				var toKey string

				if err := rows.Scan(&toKey); err != nil {
					_ = rows.Close()

					return nil, err //nolint:wrapcheck // passthrough
				}

				if !visited[toKey] {
					visited[toKey] = true

					var to any

					if jErr := json.Unmarshal([]byte(toKey), &to); jErr != nil {
						to = toKey
					}

					result = append(result, to)
					next = append(next, to)
				}
			}

			_ = rows.Close()
		}

		frontier = next
	}

	return result, nil
}

// Compile-time assertions.
var (
	_ Engine          = (*sqliteEngine)(nil)
	_ MapBackend      = (*sqliteEngine)(nil)
	_ MapUpdater      = (*sqliteEngine)(nil)
	_ ScanBackend     = (*sqliteEngine)(nil)
	_ SetBackend      = (*sqliteEngine)(nil)
	_ CounterBackend  = (*sqliteEngine)(nil)
	_ GraphBackend    = (*sqliteEngine)(nil)
	_ MultimapBackend = (*sqliteEngine)(nil)
	_ LogBackend      = (*sqliteEngine)(nil)
)

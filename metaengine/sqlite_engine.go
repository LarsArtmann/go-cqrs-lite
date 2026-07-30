package metaengine

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"errors"
	"fmt"
	"iter"
	"sort"
	"strings"
	"sync"
)

// sqliteEngine implements all ADT backends backed by a SQL database.
// It is the first persistent engine for the metaengine, enabling data
// to survive process restarts.
type sqliteEngine struct {
	db      *sql.DB
	queries sqliteQuerySet
	// seq counters for multimap and log (SQLite AUTOINCREMENT handles log).
	// Lazily seeded from MAX(seq) on first use — see multiSeqCounter.
	multiSeq sync.Map // collection→*multiSeqCounter
	// plans maps collection → LayoutPlan for layout-planned tables.
	// Populated by ApplyLayout (called by Plan() for queries with
	// FilterOnField/SortOnField, or by NewPlannedSQLiteEngine).
	plans map[string]LayoutPlan
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
	graphAddEdge string
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
	}
}

// NewSQLiteEngine creates a SQLite-backed metaengine engine. The caller owns
// the *sql.DB. Tables are created automatically if they don't exist.
func NewSQLiteEngine(database *sql.DB) (Engine, error) {
	eng := &sqliteEngine{
		db:      database,
		queries: defaultSQLiteQueries(),
	}

	if _, err := database.ExecContext(context.Background(), eng.queries.ddl); err != nil {
		return nil, fmt.Errorf("metaengine: create tables: %w", err)
	}

	return eng, nil
}

func (e *sqliteEngine) Profile() EngineProfile {
	return SQLiteEngineProfile()
}

func (e *sqliteEngine) Close() error { return nil }

func encodeKey(key any) string {
	return encodeJSON(key)
}

func encodeValue(value any) string {
	return encodeJSON(value)
}

// encodeJSON marshals v to a JSON string, falling back to fmt.Sprintf("%v", v)
// if v is not JSON-serializable. Centralized so encodeKey/encodeValue stay
// in sync — both are the same operation on different conceptual inputs.
func encodeJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}

	return string(b)
}

// --- MapBackend ---

func (e *sqliteEngine) MapSet(ctx context.Context, col string, key any, value any) error {
	if plan, ok := e.plans[col]; ok {
		return e.mapSetPlanned(ctx, plan, key, value)
	}

	_, err := e.db.ExecContext(ctx, e.queries.mapSet, col, encodeKey(key), encodeValue(value))

	return err //nolint:wrapcheck // passthrough
}

func (e *sqliteEngine) MapGet(ctx context.Context, col string, key any) (any, bool, error) {
	if plan, ok := e.plans[col]; ok {
		return e.mapGetPlanned(ctx, plan, key)
	}

	var valStr string

	err := e.db.QueryRowContext(ctx, e.queries.mapGet, col, encodeKey(key)).Scan(&valStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}

		return nil, false, err //nolint:wrapcheck // passthrough
	}

	return jsonValue(valStr), true, nil
}

func (e *sqliteEngine) MapDelete(ctx context.Context, col string, key any) error {
	if plan, ok := e.plans[col]; ok {
		_, err := e.db.ExecContext(ctx,
			fmt.Sprintf("DELETE FROM %s WHERE key = ?", plan.Table),
			encodeKey(key))
		return err //nolint:wrapcheck // passthrough
	}

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
	// Wrap the read-modify-write in a single transaction so concurrent
	// MapUpdate calls on the same key cannot interleave their reads and
	// writes (lost-update). The tx pins one connection from the pool; the
	// SELECT and INSERT commit atomically. This matches the memory engine's
	// lock-based atomicity as closely as database/sql permits without a
	// driver-specific BEGIN IMMEDIATE.
	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return err //nolint:wrapcheck // passthrough
	}

	defer func() { _ = tx.Rollback() }()

	var valStr string

	queryErr := tx.QueryRowContext(ctx, e.queries.mapGet, col, encodeKey(key)).Scan(&valStr)

	var prev any

	if queryErr == nil {
		prev = decodeJSONValue(valStr)
	} else if !errors.Is(queryErr, sql.ErrNoRows) {
		return queryErr //nolint:wrapcheck // passthrough
	}

	newVal := update(prev)

	if _, err := tx.ExecContext(
		ctx,
		e.queries.mapSet,
		col,
		encodeKey(key),
		encodeValue(newVal),
	); err != nil {
		return err //nolint:wrapcheck // passthrough
	}

	return tx.Commit() //nolint:wrapcheck // passthrough
}

// --- ScanBackend ---

func (e *sqliteEngine) MapScan(
	ctx context.Context,
	col string,
	filterFn func(item any) bool,
	sortFunc func(a, b any) int,
	cursor any,
	limit int,
) ([]any, error) {
	// Planned collections store rows in a dedicated table; MapScan (the
	// closure-based fallback) must read from it, not meta_map.
	var rows *sql.Rows

	var err error

	if plan, ok := e.plans[col]; ok {
		rows, err = e.db.QueryContext(ctx, fmt.Sprintf("SELECT value FROM %s", plan.Table))
	} else {
		rows, err = e.db.QueryContext(ctx, `SELECT value FROM meta_map WHERE collection = ?`, col)
	}

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

		val := decodeJSONValue(valStr)

		if filterFn != nil && !filterFn(val) {
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

// --- PushdownScan ---

// PushdownMapScan pushes WHERE/ORDER BY/LIMIT into SQL using json_extract(),
// avoiding the full-table scan that MapScan performs. Filters become
// json_extract(value, '$.field') = ?, sort becomes ORDER BY json_extract(...),
// and limit becomes LIMIT. Cursor-based keyset pagination is pushed as an
// additional WHERE clause on the sort column.
func (e *sqliteEngine) PushdownMapScan(
	ctx context.Context,
	col string,
	filters []FilterSpec,
	sort *SortSpec,
	cursor any,
	limit int,
) ([]any, error) {
	if plan, ok := e.plans[col]; ok {
		return e.pushdownMapScanPlanned(ctx, plan, filters, sort, cursor, limit)
	}

	var b strings.Builder

	args := []any{col}

	b.WriteString(`SELECT value FROM meta_map WHERE collection = ?`)

	// Push filter predicates into WHERE.
	for _, f := range filters {
		path := jsonPath(f.Column)

		b.WriteString(` AND json_extract(value, '`)
		b.WriteString(path)
		b.WriteString(`') `)
		b.WriteString(string(f.Op))
		b.WriteString(` ?`)

		args = append(args, f.Value)
	}

	// Push keyset cursor into WHERE (must come before ORDER BY).
	if sort != nil && cursor != nil {
		path := jsonPath(sort.Column)

		op := ">"
		if sort.Desc {
			op = "<"
		}

		b.WriteString(` AND json_extract(value, '`)
		b.WriteString(path)
		b.WriteString(`') `)
		b.WriteString(op)
		b.WriteString(` ?`)

		args = append(args, cursor)
	}

	// Push sort into ORDER BY.
	if sort != nil {
		path := jsonPath(sort.Column)

		b.WriteString(` ORDER BY json_extract(value, '`)
		b.WriteString(path)
		b.WriteString(`')`)

		if sort.Desc {
			b.WriteString(` DESC`)
		}
	}

	// Push limit (with +1 for has-more detection).
	if limit > 0 {
		b.WriteString(` LIMIT ?`)

		args = append(args, limit+1)
	}

	return scanJSONValues(ctx, e.db, b.String(), args...)
}

// jsonPath converts a field name to a JSON path for json_extract.
// E.g. "status" → "$.status".
func jsonPath(field string) string {
	return "$." + field
}

// --- StreamingScan ---

// StreamScan returns an iterator over collection rows, applying filter and sort
// pushdown. Planned collections use indexed column references; standard ones use
// json_extract. Each row is decoded lazily — no materialization of the full
// result set. This prevents OOM for large collections.
func (e *sqliteEngine) StreamScan(
	ctx context.Context,
	col string,
	filters []FilterSpec,
	sort *SortSpec,
) iter.Seq2[any, error] {
	return func(yield func(any, error) bool) {
		query, args := e.buildStreamQuery(col, filters, sort)

		rows, err := e.db.QueryContext(ctx, query, args...)
		if err != nil {
			yield(nil, err)

			return
		}

		defer func() { _ = rows.Close() }()

		for rows.Next() {
			var valStr string

			if err := rows.Scan(&valStr); err != nil {
				yield(nil, err)

				return
			}

			if !yield(decodeJSONValue(valStr), nil) {
				return
			}
		}

		if err := rows.Err(); err != nil {
			yield(nil, err)
		}
	}
}

// buildStreamQuery constructs the SELECT statement for StreamScan, using direct
// column references for planned collections and json_extract for standard ones.
func (e *sqliteEngine) buildStreamQuery(
	col string,
	filters []FilterSpec,
	sort *SortSpec,
) (string, []any) {
	var b strings.Builder

	if plan, ok := e.plans[col]; ok {
		fmt.Fprintf(&b, "SELECT value FROM %s", plan.Table)

		args := []any{}

		for i, f := range filters {
			if i == 0 {
				b.WriteString(" WHERE ")
			} else {
				b.WriteString(" AND ")
			}

			fmt.Fprintf(&b, "%s %s ?", f.Column, string(f.Op))
			args = append(args, f.Value)
		}

		if sort != nil {
			fmt.Fprintf(&b, " ORDER BY %s", sort.Column)

			if sort.Desc {
				b.WriteString(" DESC")
			}
		}

		return b.String(), args
	}

	args := make([]any, 0, 1+len(filters))
	args = append(args, col)

	b.WriteString(`SELECT value FROM meta_map WHERE collection = ?`)

	for _, f := range filters {
		path := jsonPath(f.Column)

		b.WriteString(` AND json_extract(value, '`)
		b.WriteString(path)
		b.WriteString(`') `)
		b.WriteString(string(f.Op))
		b.WriteString(` ?`)

		args = append(args, f.Value)
	}

	if sort != nil {
		path := jsonPath(sort.Column)

		b.WriteString(` ORDER BY json_extract(value, '`)
		b.WriteString(path)
		b.WriteString(`')`)

		if sort.Desc {
			b.WriteString(` DESC`)
		}
	}

	return b.String(), args
}

// Compile-time assertions.
var (
	_ Engine          = (*sqliteEngine)(nil)
	_ MapBackend      = (*sqliteEngine)(nil)
	_ MapUpdater      = (*sqliteEngine)(nil)
	_ ScanBackend     = (*sqliteEngine)(nil)
	_ PushdownScan    = (*sqliteEngine)(nil)
	_ StreamingScan   = (*sqliteEngine)(nil)
	_ LayoutPlanner   = (*sqliteEngine)(nil)
	_ SetBackend      = (*sqliteEngine)(nil)
	_ CounterBackend  = (*sqliteEngine)(nil)
	_ GraphBackend    = (*sqliteEngine)(nil)
	_ MultimapBackend = (*sqliteEngine)(nil)
	_ LogBackend      = (*sqliteEngine)(nil)
)

package sqliteengine

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"iter"
	"sort"
	"strings"
	"sync"
	"time"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/claimkit"
)

// sqliteEngine implements all metaengine.ADT backends on a SQL database.
type sqliteEngine struct {
	metaengine.Calibration

	db *sql.DB
	// ownsDB: engine opened its own *sql.DB (NewSQLiteEngineFromDSN), so
	// Close also closes the database; caller-supplied pools leave it false.
	ownsDB            bool
	synchronousPragma string
	queries           sqliteQuerySet
	cache             *stmtCache
	// graphCTE: single-query recursive-CTE traversal when WITH RECURSIVE is available (probed).
	graphCTE bool
	// vectorSQL: lazily-probed libSQL SQL-side k-NN availability (turso yes,
	// modernc no); cached, so construction stays query-free for the common
	// modernc deployments that never search.
	vectorSQL func() bool
	// seq counters for multimap and log (SQLite AUTOINCREMENT handles log).
	multiSeq sync.Map // collection→*multiSeqCounter
	plans    map[string]metaengine.LayoutPlan
	txMu     sync.Mutex
	probeFn  func(context.Context) (time.Duration, error)
	// matViews holds operator-declared materialized view specs (Turso IVM
	// acceleration); matViewErr captures a spec-validation failure surfaced
	// at construction.
	matViews   []matView
	matViewErr error
	// sortIndexes holds operator-declared meta_map expression indexes
	// (WithSortIndex, issue #122); sortIndexErr captures a declaration
	// validation failure surfaced at construction.
	sortIndexes  []mapSortIndex
	sortIndexErr error
	// Cell versioning (ADR-0141): when true, map writes on non-planned
	// collections also record timestamped versions in meta_cell_versions;
	// versionRetention trims history per write when set.
	versioning       bool
	versionRetention *metaengine.RetentionPolicy

	// claimkit runtimes (ADR-0142): DueClaimer + FactSink + DedupStore by
	// method promotion — the ONE shared SQL claim/dedup implementation;
	// see dueclaim.go.
	*claimkit.Claims
	*claimkit.Dedup
}

// NewSQLiteEngine creates a SQLite-backed metaengine engine. The caller owns
// the *sql.DB. Tables are created automatically if they don't exist.
func NewSQLiteEngine(database *sql.DB, opts ...EngineOption) (metaengine.Engine, error) {
	eng := &sqliteEngine{
		db:        database,
		queries:   defaultSQLiteQueries(),
		cache:     newStmtCache(database),
		graphCTE:  probeRecursiveCTE(database),
		vectorSQL: sync.OnceValue(func() bool { return probeVectorSQL(database) }),
	}

	for _, opt := range opts {
		opt(eng)
	}

	if _, err := database.ExecContext(context.Background(), eng.queries.ddl); err != nil {
		return nil, fmt.Errorf("metaengine: create tables: %w", err)
	}

	// Typed WAL migration: guarded ALTER + backfill + unique indexes over
	// legacy databases (fresh ones already match — the call is idempotent).
	if err := ensureTypedStreamLog(database); err != nil {
		return nil, err
	}

	// Enable memory-mapped I/O for faster point lookups on file-backed databases.
	// 256MB mmap window; harmless on :memory: databases.
	_, _ = database.ExecContext(context.Background(), `PRAGMA mmap_size = 268435456`)

	if err := eng.createMatViews(context.Background()); err != nil {
		return nil, err
	}

	if err := eng.createSortIndexes(context.Background()); err != nil {
		return nil, err
	}

	if err := eng.wireClaimkit(); err != nil {
		return nil, err
	}

	return eng, nil
}

func (e *sqliteEngine) Profile() metaengine.EngineProfile {
	p := metaengine.SQLiteEngineProfile()
	e.ApplyCalibration(&p)

	return p
}

func (e *sqliteEngine) Close() error {
	if e.cache != nil {
		e.cache.close()
	}

	if e.ownsDB {
		return e.db.Close() //nolint:wrapcheck // driver-provided close error
	}

	return nil
}

// HealthCheck pings the underlying database to verify connectivity.
// Implements [metaengine.HealthChecker] for Kubernetes-style liveness probes.
func (e *sqliteEngine) HealthCheck(ctx context.Context) error {
	return e.db.PingContext(ctx)
}

// --- metaengine.MapBackend ---

func (e *sqliteEngine) MapSet(ctx context.Context, col string, key any, value any) error {
	if plan, ok := e.plans[col]; ok {
		return e.mapSetPlanned(ctx, plan, key, value)
	}

	return e.mapSetEncoded(ctx, col, key, value, time.Now())
}

func (e *sqliteEngine) MapGet(ctx context.Context, col string, key any) (any, bool, error) {
	if plan, ok := e.plans[col]; ok {
		return e.mapGetPlanned(ctx, plan, key)
	}

	var valStr string

	err := e.xc(ctx).queryRow(ctx, e.queries.mapGet, col, encodeKey(key)).Scan(&valStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}

		return nil, false, err //nolint:wrapcheck // passthrough
	}

	return metaengine.DecodeStreamValue(valStr), true, nil
}

func (e *sqliteEngine) MapDelete(ctx context.Context, col string, key any) error {
	if plan, ok := e.plans[col]; ok {
		_, err := e.xd(ctx).ExecContext(ctx,
			fmt.Sprintf("DELETE FROM %s WHERE key = ?", metaengine.QuoteIdent(plan.Table)),
			encodeKey(key))

		return err //nolint:wrapcheck // passthrough
	}

	keyStr := encodeKey(key)

	if _, err := e.xc(ctx).exec(ctx, e.queries.mapDelete, col, keyStr); err != nil {
		return err //nolint:wrapcheck // passthrough
	}

	if e.versioning {
		if err := e.recordVersionRow(ctx, col, keyStr, nil, time.Now()); err != nil {
			return err
		}
	}

	return nil
}

// --- metaengine.MapUpdater ---

func (e *sqliteEngine) MapUpdate(
	ctx context.Context,
	col string,
	key any,
	update func(prev any) any,
) error {
	// Planned collections read/write from the planned table, not meta_map.
	if plan, ok := e.plans[col]; ok {
		return e.mapUpdatePlanned(ctx, plan, key, update)
	}

	// Wrap the read-modify-write in a single transaction so concurrent
	// MapUpdate calls on the same key cannot interleave their reads and
	// writes (lost-update). The tx pins one connection from the pool; the
	// SELECT and INSERT commit atomically.
	//
	// When inside an outer transaction (RunInTx), reuse it instead of
	// starting a nested BeginTx (SQLite does not support nested BEGIN).
	if e.txExec(ctx) != nil {
		return readModifyWriteCached(
			ctx,
			e.xc(ctx),
			e.queries.mapGet,
			e.queries.mapSet,
			col,
			key,
			update,
		)
	}

	return runTxReadModifyWrite(
		ctx, e.db, update,
		func(ctx context.Context, exec metaengine.SQLExec) (any, error) {
			var valStr string
			if err := exec.QueryRowContext(ctx, e.queries.mapGet, col, encodeKey(key)).
				Scan(&valStr); err != nil {
				return nil, err //nolint:wrapcheck // ErrNoRows handled by caller
			}

			return metaengine.DecodeStreamValue(valStr), nil
		},
		func(ctx context.Context, exec metaengine.SQLExec, newVal any) error {
			encoded, err := encodeValue(newVal)
			if err != nil {
				return err
			}

			_, err = exec.ExecContext(ctx, e.queries.mapSet, col, encodeKey(key), encoded)

			return err //nolint:wrapcheck // passthrough
		},
	)
}

// --- metaengine.ScanBackend ---

func (e *sqliteEngine) MapScan(
	ctx context.Context,
	col string,
	filterFn func(item any) bool,
	sortFunc func(a, b any) int,
	cursor any,
	limit int,
) (metaengine.ScanResult, error) {
	// Planned collections store rows in a dedicated table; MapScan (the
	// closure-based fallback) must read from it, not meta_map. Both tables
	// expose the map key, and the tiebreak key is that raw key string —
	// mirroring pgengine — so compound SortKeyCursor pagination is externally
	// derivable (the F22 conformance matrix: tiebreaking on the stored value
	// bytes made cursor keys un-derivable, since re-marshalling a decoded map
	// does not reproduce json v2's non-canonical map key order).
	var rows *sql.Rows

	var err error

	if plan, ok := e.plans[col]; ok {
		rows, err = e.xd(ctx).
			QueryContext(ctx, "SELECT key, value FROM "+metaengine.QuoteIdent(plan.Table))
	} else {
		rows, err = e.xd(ctx).
			QueryContext(ctx, `SELECT key, value FROM meta_map WHERE collection = ?`, col)
	}

	if err != nil {
		return metaengine.ScanResult{}, err //nolint:wrapcheck // passthrough
	}

	defer metaengine.DeferClose(rows)

	type kv struct {
		key   string
		value any
	}

	var pairs []kv

	for rows.Next() {
		var key, valStr string

		if err := rows.Scan(&key, &valStr); err != nil {
			return metaengine.ScanResult{}, err //nolint:wrapcheck // passthrough
		}

		val := metaengine.DecodeStreamValue(valStr)

		if filterFn != nil && !filterFn(val) {
			continue
		}

		pairs = append(pairs, kv{key: key, value: val})
	}

	if err := rows.Err(); err != nil {
		return metaengine.ScanResult{}, err //nolint:wrapcheck // passthrough
	}

	if sortFunc == nil {
		sort.Slice(pairs, func(i, j int) bool {
			return bytes.Compare([]byte(pairs[i].key), []byte(pairs[j].key)) < 0
		})

		return metaengine.PairsToScanResult(pairs, func(p kv) any { return p.value }, limit), nil
	}

	return metaengine.PairsToScanResult(
		metaengine.SortPaginate(
			pairs,
			func(p kv) []byte { return []byte(p.key) },
			func(p kv) any { return p.value },
			sortFunc,
			cursor,
			limit,
		),
		func(p kv) any { return p.value },
		limit,
	), nil
}

// --- metaengine.PushdownScan ---

// PushdownMapScan pushes WHERE/ORDER BY/LIMIT into SQL using json_extract(),
// avoiding the full-table scan that MapScan performs. Filters become
// json_extract(value, '$.field') = ?, sort becomes ORDER BY json_extract(...),
// and limit becomes LIMIT. Cursor-based keyset pagination is pushed as an
// additional WHERE clause on the sort column.
func (e *sqliteEngine) PushdownMapScan(
	ctx context.Context,
	col string,
	filters []metaengine.FilterSpec,
	sort *metaengine.SortSpec,
	cursor any,
	limit int,
) (metaengine.ScanResult, error) {
	if err := metaengine.ValidateFilterSpecs(filters); err != nil {
		return metaengine.ScanResult{}, err
	}

	for _, f := range filters {
		if err := metaengine.ValidateIdentifier(f.Column); err != nil {
			return metaengine.ScanResult{}, fmt.Errorf("filter column: %w", err)
		}
	}

	if sort != nil {
		if err := metaengine.ValidateIdentifier(sort.Column); err != nil {
			return metaengine.ScanResult{}, fmt.Errorf("sort column: %w", err)
		}
	}

	if plan, ok := e.plans[col]; ok {
		return e.pushdownMapScanPlanned(ctx, plan, filters, sort, cursor, limit)
	}

	query, args := buildStandardScanQuery(col, filters, sort, cursor, limit)

	rows, keys, err := scanJSONValuesWithKeys(ctx, e.xd(ctx), query, args...)
	if err != nil {
		return metaengine.ScanResult{}, err
	}

	hasMore := limit > 0 && len(rows) > limit
	if hasMore {
		rows = rows[:limit]
		keys = keys[:limit]
	}

	var next any
	if sort != nil && len(rows) > 0 {
		next = metaengine.LastDecodedRowCursor(
			rows[len(rows)-1],
			sort.Column,
			[]byte(keys[len(keys)-1]),
		)
	}

	return metaengine.ScanResult{Items: rows, HasMore: hasMore, NextCursor: next}, nil
}

// --- metaengine.StreamingScan ---

// StreamScan returns an iterator over collection rows, applying filter and sort
// pushdown. Planned collections use indexed column references; standard ones use
// json_extract. Each row is decoded lazily — no materialization of the full
// result set. This prevents OOM for large collections.
func (e *sqliteEngine) StreamScan(
	ctx context.Context,
	col string,
	filters []metaengine.FilterSpec,
	sort *metaengine.SortSpec,
) iter.Seq2[any, error] {
	return func(yield func(any, error) bool) {
		query, args, err := e.buildStreamQuery(col, filters, sort)
		if err != nil {
			yield(nil, err)

			return
		}

		rows, err := e.xd(ctx).QueryContext(ctx, query, args...) //nolint:sqlclosecheck
		if err != nil {
			yield(nil, err)

			return
		}

		defer metaengine.DeferClose(rows)

		for rows.Next() {
			var valStr string

			if err := rows.Scan(&valStr); err != nil {
				yield(nil, err)

				return
			}

			if !yield(metaengine.DecodeStreamValue(valStr), nil) {
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
	filters []metaengine.FilterSpec,
	sort *metaengine.SortSpec,
) (string, []any, error) {
	if err := metaengine.ValidateFilterSpecs(filters); err != nil {
		return "", nil, err
	}

	for _, f := range filters {
		if err := metaengine.ValidateIdentifier(f.Column); err != nil {
			return "", nil, fmt.Errorf("filter column: %w", err)
		}
	}

	if sort != nil {
		if err := metaengine.ValidateIdentifier(sort.Column); err != nil {
			return "", nil, fmt.Errorf("sort column: %w", err)
		}
	}

	var b strings.Builder

	if plan, ok := e.plans[col]; ok {
		fmt.Fprintf(&b, "SELECT value FROM %s", metaengine.QuoteIdent(plan.Table))

		args := make([]any, 0, len(filters))

		for i, f := range filters {
			if i == 0 {
				b.WriteString(" WHERE ")
			} else {
				b.WriteString(" AND ")
			}

			fmt.Fprintf(&b, "%s %s ?", metaengine.QuoteIdent(f.Column), string(f.Op))
			args = append(args, metaengine.FilterBindValue(f.Value))
		}

		if sort != nil {
			fmt.Fprintf(&b, " ORDER BY %s", metaengine.QuoteIdent(sort.Column))

			if sort.Desc {
				b.WriteString(" DESC")
			}
		}

		return b.String(), args, nil
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

		args = append(args, metaengine.FilterBindValue(f.Value))
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

	return b.String(), args, nil
}

// Compile-time assertions.
var (
	_ metaengine.Engine            = (*sqliteEngine)(nil)
	_ metaengine.MapBackend        = (*sqliteEngine)(nil)
	_ metaengine.MapUpdater        = (*sqliteEngine)(nil)
	_ metaengine.ScanBackend       = (*sqliteEngine)(nil)
	_ metaengine.PushdownScan      = (*sqliteEngine)(nil)
	_ metaengine.StreamingScan     = (*sqliteEngine)(nil)
	_ metaengine.LayoutPlanner     = (*sqliteEngine)(nil)
	_ metaengine.LayoutPlanApplier = (*sqliteEngine)(nil)
	_ metaengine.RawValueReader    = (*sqliteEngine)(nil)
	_ metaengine.RawScanReader     = (*sqliteEngine)(nil)
	_ metaengine.SetBackend        = (*sqliteEngine)(nil)
	_ metaengine.CounterBackend    = (*sqliteEngine)(nil)
	_ metaengine.MultimapBackend   = (*sqliteEngine)(nil)
	_ metaengine.LogBackend        = (*sqliteEngine)(nil)
	_ metaengine.Calibratable      = (*sqliteEngine)(nil)
	_ metaengine.TrackerHost       = (*sqliteEngine)(nil)
	_ metaengine.Prober            = (*sqliteEngine)(nil)
	_ metaengine.CapGroupTrimmer   = (*sqliteEngine)(nil)
)

package relational

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"

	errorfamily "github.com/larsartmann/go-error-family"

	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

// Row is an ordered-by-name set of column/value pairs for a sink write.
//
// Values may be any SQL-compatible Go type (string, int, bool, []byte, etc.).
// time.Time values are formatted via the configured [sqlpkg.Dialect] so the
// same projection handler runs on both SQLite and PostgreSQL without change.
//
// The "any" value type here is the accepted database/sql interop exception to
// the library's no-"any"-in-domain rule — this is storage infrastructure, not
// domain logic.
type Row map[string]any

// ProjectionSink is a transactional, dialect-agnostic write context passed to
// relational projection handlers. All writes performed through a sink during a
// single [RelationalProjection.Handle] call commit atomically — if the handler
// returns an error, every write is rolled back.
//
// Handlers never touch *sql.DB directly. The dialect (SQLite or PostgreSQL) is
// chosen at deployment time when the projection is constructed, not when the
// handler is written. This is what makes relational projections portable: the
// same handler code writes across multiple related tables on either backend.
//
// The sink methods generate parameterised SQL from structured inputs — column
// names are trusted (declared in the schema), user input only ever reaches the
// bound arguments, so the generated statements are injection-safe.
//
// Scope: this is an SQL abstraction, not a universal one. Row/column/table and
// the set-predicate operations (Update, DeleteWhere, QueryOne) are relational
// concepts — they have no meaning on a KV store (no columns, no predicates) and
// only partial meaning on a graph (FK columns should be edges, not properties).
// For KV/document backends use stack.Materialize + kv.ViewStore[V,K]; a graph
// backend would need a distinct sink exposing node/edge operations instead.
type ProjectionSink interface {
	// Upsert inserts a row, or on conflict with conflictCols updates the
	// conflicting row's other columns to the new values. An empty conflictCols
	// defaults to the table's declared primary key. With no non-conflict
	// columns, the upsert degrades to "DO NOTHING" (idempotent insert).
	Upsert(ctx context.Context, table string, row Row, conflictCols ...string) error

	// Ensure inserts a row only if no conflicting row exists; an existing row
	// is left untouched (INSERT OR IGNORE semantics). Use it for "ensure parent
	// exists" upserts (e.g. ensure a guild/channel/user row exists before the
	// referencing message). The conflict target defaults to the table's primary key.
	Ensure(ctx context.Context, table string, row Row) error

	// Update sets columns on rows matching all of match's equal conditions.
	Update(ctx context.Context, table string, set, match Row) error

	// DeleteWhere removes rows matching all of match's equal conditions.
	DeleteWhere(ctx context.Context, table string, match Row) error

	// QueryOne reads a single column from the first row matching match. Returns
	// [errSinkNoRows] (wrapping sql.ErrNoRow) when no row matches. Use it for
	// read-then-write patterns inside a projection (e.g. read current content
	// before recording an edit history row).
	QueryOne(ctx context.Context, table, column string, match Row) (any, error)

	// Increment atomically adds delta to counterCol on the row identified by
	// key. If the row does not exist, it is inserted with delta as the initial
	// counter value.
	//
	// The key Row must include the table's primary key columns and must not
	// contain counterCol. Generated SQL (portable across SQLite and PostgreSQL):
	//
	//	INSERT INTO <table> (<keycols>, <counterCol>) VALUES (?, ?, ?)
	//	ON CONFLICT(<pk>) DO UPDATE SET <counterCol> = COALESCE(<counterCol>, 0) + excluded.<counterCol>
	//
	// COALESCE handles multi-counter tables where other counter columns are NULL
	// when the row is first created by a different Increment call: without it,
	// NULL + N would yield NULL in SQL.
	//
	// Use this for incremental rollup tables — pre-computed counters and sums
	// maintained as events flow, so dashboard reads are O(1) instead of
	// O(full table scan). The counter is NOT clamped to zero on negative delta:
	// a counter going below zero signals inconsistent events (more deletes than
	// creates), and silent clamping would hide that data-loss bug.
	Increment(ctx context.Context, table string, key Row, counterCol string, delta int64) error

	// UpsertCols inserts a row, or on conflict with conflictCols updates only
	// the columns listed in updateCols (instead of all non-conflict columns
	// like [Upsert]). This is the "partial upsert": the INSERT writes every
	// column in row, but the ON CONFLICT DO UPDATE SET clause only touches the
	// declared subset.
	//
	// Use it when an event carries a partial payload — e.g. a MessageUpdated
	// event that should overwrite content and edited_at but must not clobber
	// created_at or author_id. An empty updateCols degrades to "DO NOTHING"
	// (idempotent insert).
	//
	// conflictCols defaults to the table's declared primary key.
	UpsertCols(ctx context.Context, table string, row Row, updateCols []string, conflictCols ...string) error

	// UpsertExpr inserts a row, or on conflict applies the given SetExpr list
	// to the conflicting row. Each SetExpr is a raw SQL expression that may
	// reference both excluded.<col> (the incoming values) and <table>.<col>
	// (the existing row). This enables COALESCE/NULLIF patterns:
	//
	//	SetExpr{Column: "content", Expr: "COALESCE(NULLIF(excluded.content, ''), messages.content)"}
	//
	// The row supplies the INSERT VALUES; the setExprs supply the
	// ON CONFLICT DO UPDATE SET clause. conflictCols defaults to the table's
	// primary key. If setExprs is empty, the upsert degrades to "DO NOTHING".
	UpsertExpr(ctx context.Context, table string, row Row, setExprs []SetExpr, conflictCols ...string) error

	// Tx returns the underlying *sql.Tx that all sink writes execute within.
	// It is the escape hatch for operations the structured methods cannot
	// express: recursive CTEs, INSERT INTO ... SELECT, bulk updates with
	// complex predicates, or calls to dialect-specific functions.
	//
	// The projection handler does NOT call Commit or Rollback — the
	// RelationalProjection owns the transaction lifecycle (commit on
	// handler-return-nil, rollback on handler-return-error). Raw SQL executed
	// via Tx() participates in the same atomic commit/rollback as every sink
	// call, so a handler can mix structured sink writes and raw SQL freely
	// within one Handle call.
	//
	// Use sparingly. Every raw SQL statement is SQL the consumer must
	// maintain. If a pattern recurs across handlers, promote it to a sink
	// method instead.
	Tx() *sql.Tx
}

// SetExpr is one column=expression assignment in an [ProjectionSink.UpsertExpr]
// ON CONFLICT DO UPDATE SET clause. Expr is a raw SQL expression — it may
// reference excluded.<col> for incoming values and <table>.<col> for the
// existing row. Args supplies any bound parameters the expression needs.
type SetExpr struct {
	Column string
	Expr   string
	Args   []any
}

// sqlSink implements ProjectionSink over a *sql.Tx with a fixed dialect.
type sqlSink struct {
	tx      *sql.Tx
	schema  RelationalSchema
	dialect sqlpkg.Dialect
}

func newSQLSink(tx *sql.Tx, schema RelationalSchema, dialect sqlpkg.Dialect) *sqlSink {
	return &sqlSink{tx: tx, schema: schema, dialect: dialect}
}

func (s *sqlSink) Upsert(ctx context.Context, table string, row Row, conflictCols ...string) error {
	cols, vals, err := s.rowColumns(table, row)
	if err != nil {
		return err
	}

	if len(conflictCols) == 0 {
		conflictCols = s.conflictTarget(table)
	}

	nonConflict, _ := partitionColumns(cols, conflictCols)

	setClause := excludedSet(nonConflict)
	pholders := placeholders(s.dialect, len(cols))

	onConflict := "DO NOTHING"
	if setClause != "" {
		onConflict = "DO UPDATE SET " + setClause
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) ON CONFLICT(%s) %s",
		table, strings.Join(cols, ", "), pholders, strings.Join(conflictCols, ", "), onConflict,
	)

	if _, err := s.tx.ExecContext(ctx, query, vals...); err != nil {
		return errorfamily.WrapTransient(err, "relational.sink_upsert",
			"upsert into "+table)
	}

	return nil
}

func (s *sqlSink) Ensure(ctx context.Context, table string, row Row) error {
	cols, vals, err := s.rowColumns(table, row)
	if err != nil {
		return err
	}

	pholders := placeholders(s.dialect, len(cols))

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) ON CONFLICT DO NOTHING",
		table, strings.Join(cols, ", "), pholders,
	)

	if _, err := s.tx.ExecContext(ctx, query, vals...); err != nil {
		return errorfamily.WrapTransient(err, "relational.sink_ensure",
			"ensure into "+table)
	}

	return nil
}

func (s *sqlSink) Update(ctx context.Context, table string, set, match Row) error {
	setCols, setVals, err := s.rowColumns(table, set)
	if err != nil {
		return err
	}

	matchCols, matchVals, matchErr := s.rowColumns(table, match)
	if matchErr != nil {
		return matchErr
	}

	pairs := make([]string, len(setCols))

	for i, c := range setCols {
		pairs[i] = fmt.Sprintf("%s = %s", c, s.dialect.Placeholder(i+1))
	}

	args := setVals

	where, whereArgs := eqWhere(matchCols, matchVals, s.dialect, len(args)+1)
	args = append(args, whereArgs...)

	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s", table, strings.Join(pairs, ", "), where)

	if _, err := s.tx.ExecContext(ctx, query, args...); err != nil {
		return errorfamily.WrapTransient(err, "relational.sink_update",
			"update "+table)
	}

	return nil
}

func (s *sqlSink) DeleteWhere(ctx context.Context, table string, match Row) error {
	matchCols, matchVals, err := s.rowColumns(table, match)
	if err != nil {
		return err
	}

	where, whereArgs := eqWhere(matchCols, matchVals, s.dialect, 1)

	query := fmt.Sprintf("DELETE FROM %s WHERE %s", table, where)

	if _, err := s.tx.ExecContext(ctx, query, whereArgs...); err != nil {
		return errorfamily.WrapTransient(err, "relational.sink_delete",
			"delete from "+table)
	}

	return nil
}

func (s *sqlSink) QueryOne(ctx context.Context, table, column string, match Row) (any, error) {
	matchCols, matchVals, err := s.rowColumns(table, match)
	if err != nil {
		return nil, err
	}

	where, whereArgs := eqWhere(matchCols, matchVals, s.dialect, 1)

	query := fmt.Sprintf("SELECT %s FROM %s WHERE %s LIMIT 1", column, table, where)

	var result any

	err = s.tx.QueryRowContext(ctx, query, whereArgs...).Scan(&result)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errorfamily.WrapRejection(errSinkNoRows,
				"relational.sink_no_rows",
				fmt.Sprintf("query %s.%s", table, column))
		}

		return nil, errorfamily.WrapCorruption(err, "relational.sink_query",
			fmt.Sprintf("query %s.%s", table, column))
	}

	return result, nil
}


func (s *sqlSink) Tx() *sql.Tx { return s.tx }

func (s *sqlSink) UpsertCols(
	ctx context.Context,
	table string,
	row Row,
	updateCols []string,
	conflictCols ...string,
) error {
	cols, vals, err := s.rowColumns(table, row)
	if err != nil {
		return err
	}

	if len(conflictCols) == 0 {
		conflictCols = s.conflictTarget(table)
	}

	nonConflict, _ := partitionColumns(cols, conflictCols)

	var targetCols []string
	if len(updateCols) > 0 {
		updateSet := make(map[string]struct{}, len(updateCols))
		for _, c := range updateCols {
			updateSet[c] = struct{}{}
		}

		for _, c := range nonConflict {
			if _, ok := updateSet[c]; ok {
				targetCols = append(targetCols, c)
			}
		}
	}

	setClause := excludedSet(targetCols)
	pholders := placeholders(s.dialect, len(cols))

	onConflict := "DO NOTHING"
	if setClause != "" {
		onConflict = "DO UPDATE SET " + setClause
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) ON CONFLICT(%s) %s",
		table, strings.Join(cols, ", "), pholders, strings.Join(conflictCols, ", "), onConflict,
	)

	if _, err := s.tx.ExecContext(ctx, query, vals...); err != nil {
		return errorfamily.WrapTransient(err, "relational.sink_upsert_cols",
			"upsert cols into "+table)
	}

	return nil
}

func (s *sqlSink) UpsertExpr(
	ctx context.Context,
	table string,
	row Row,
	setExprs []SetExpr,
	conflictCols ...string,
) error {
	cols, vals, err := s.rowColumns(table, row)
	if err != nil {
		return err
	}

	if len(conflictCols) == 0 {
		conflictCols = s.conflictTarget(table)
	}

	knownCols := s.schema.Table(table)
	if knownCols == nil {
		return errorfamily.WrapRejection(errSinkUnknownTable,
			"relational.sink_unknown_table",
			fmt.Sprintf("table %q", table))
	}

	colSet := make(map[string]struct{}, len(knownCols.Columns))
	for _, c := range knownCols.Columns {
		colSet[c.Name] = struct{}{}
	}

	onConflict := "DO NOTHING"
	var args []any
	args = append(args, vals...)

	if len(setExprs) > 0 {
		parts := make([]string, 0, len(setExprs))

		for _, se := range setExprs {
			if _, ok := colSet[se.Column]; !ok {
				return errorfamily.WrapRejection(errSinkUnknownColumn,
					"relational.sink_unknown_column",
					fmt.Sprintf("table %q: SetExpr column %q", table, se.Column))
			}

			parts = append(parts, se.Column+" = "+se.Expr)
			args = append(args, se.Args...)
		}

		onConflict = "DO UPDATE SET " + strings.Join(parts, ", ")
	}

	pholders := placeholders(s.dialect, len(cols))

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) ON CONFLICT(%s) %s",
		table, strings.Join(cols, ", "), pholders, strings.Join(conflictCols, ", "), onConflict,
	)

	if _, err := s.tx.ExecContext(ctx, query, args...); err != nil {
		return errorfamily.WrapTransient(err, "relational.sink_upsert_expr",
			"upsert expr into "+table)
	}

	return nil
}

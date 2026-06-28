package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/larsartmann/go-cqrs-lite/kv/v3"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v3/sql"
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
	// [errSinkNoRows] (wrapping sql.ErrNoRows) when no row matches. Use it for
	// read-then-write patterns inside a projection (e.g. read current content
	// before recording an edit history row).
	QueryOne(ctx context.Context, table, column string, match Row) (any, error)
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
	cols, vals, err := s.rowColumns(row)
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
		return fmt.Errorf("sink: upsert %s: %w", table, err)
	}

	return nil
}

func (s *sqlSink) Ensure(ctx context.Context, table string, row Row) error {
	cols, vals, err := s.rowColumns(row)
	if err != nil {
		return err
	}

	pholders := placeholders(s.dialect, len(cols))

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) ON CONFLICT DO NOTHING",
		table, strings.Join(cols, ", "), pholders,
	)

	if _, err := s.tx.ExecContext(ctx, query, vals...); err != nil {
		return fmt.Errorf("sink: ensure %s: %w", table, err)
	}

	return nil
}

func (s *sqlSink) Update(ctx context.Context, table string, set, match Row) error {
	setCols, setVals, err := s.rowColumns(set)
	if err != nil {
		return err
	}

	matchCols, matchVals, matchErr := s.rowColumns(match)
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
		return fmt.Errorf("sink: update %s: %w", table, err)
	}

	return nil
}

func (s *sqlSink) DeleteWhere(ctx context.Context, table string, match Row) error {
	matchCols, matchVals, err := s.rowColumns(match)
	if err != nil {
		return err
	}

	where, whereArgs := eqWhere(matchCols, matchVals, s.dialect, 1)

	query := fmt.Sprintf("DELETE FROM %s WHERE %s", table, where)

	if _, err := s.tx.ExecContext(ctx, query, whereArgs...); err != nil {
		return fmt.Errorf("sink: delete %s: %w", table, err)
	}

	return nil
}

func (s *sqlSink) QueryOne(ctx context.Context, table, column string, match Row) (any, error) {
	matchCols, matchVals, err := s.rowColumns(match)
	if err != nil {
		return nil, err
	}

	where, whereArgs := eqWhere(matchCols, matchVals, s.dialect, 1)

	query := fmt.Sprintf("SELECT %s FROM %s WHERE %s LIMIT 1", column, table, where)

	var result any

	err = s.tx.QueryRowContext(ctx, query, whereArgs...).Scan(&result)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("sink: query %s.%s: %w", table, column, errSinkNoRows)
		}

		return nil, fmt.Errorf("sink: query %s.%s: %w", table, column, err)
	}

	return result, nil
}

// rowColumns turns a Row into sorted column names plus dialect-formatted
// values. Columns are sorted for deterministic SQL (stable golden output).
func (s *sqlSink) rowColumns(row Row) ([]string, []any, error) {
	if len(row) == 0 {
		return nil, nil, errSinkEmptyRow
	}

	cols := make([]string, 0, len(row))
	for name := range row {
		cols = append(cols, name)
	}

	sort.Strings(cols)

	vals := make([]any, len(cols))
	for i, c := range cols {
		vals[i] = s.format(row[c])
	}

	return cols, vals, nil
}

func (s *sqlSink) format(v any) any {
	if t, ok := v.(time.Time); ok {
		return s.dialect.FormatTime(t)
	}

	return v
}

// conflictTarget returns the table's declared primary key, or []string{"id"}
// when no primary key is declared.
func (s *sqlSink) conflictTarget(table string) []string {
	t := s.schema.Table(table)
	if t == nil || len(t.PrimaryKey) == 0 {
		return []string{"id"}
	}

	return t.PrimaryKey
}

func partitionColumns(all, subset []string) ([]string, []string) {
	subsetSet := make(map[string]struct{}, len(subset))

	for _, c := range subset {
		subsetSet[c] = struct{}{}
	}

	var nonSubset, isSubset []string

	for _, c := range all {
		if _, ok := subsetSet[c]; ok {
			isSubset = append(isSubset, c)
		} else {
			nonSubset = append(nonSubset, c)
		}
	}

	return nonSubset, isSubset
}

func excludedSet(cols []string) string {
	if len(cols) == 0 {
		return ""
	}

	parts := make([]string, len(cols))

	for i, c := range cols {
		parts[i] = c + " = excluded." + c
	}

	return strings.Join(parts, ", ")
}

func placeholders(dialect sqlpkg.Dialect, n int) string {
	ph := make([]string, n)

	for i := range n {
		ph[i] = dialect.Placeholder(i + 1)
	}

	return strings.Join(ph, ", ")
}

func eqWhere(cols []string, vals []any, dialect sqlpkg.Dialect, startIdx int) (string, []any) {
	parts := make([]string, len(cols))
	args := make([]any, len(cols))

	for i, c := range cols {
		parts[i] = fmt.Sprintf("%s = %s", c, dialect.Placeholder(startIdx+i))
		args[i] = vals[i]
	}

	return strings.Join(parts, " AND "), args
}

// formatConditions returns a copy of conditions whose time.Time values are
// rendered through the dialect — so reads match the dialect-formatted
// timestamps the sink wrote. Without this, a WHERE created_at < ? bound with a
// raw time.Time would not compare correctly against TEXT-stored (SQLite) or
// TIMESTAMP-stored (Postgres) values.
func formatConditions(conditions []kv.Condition, dialect sqlpkg.Dialect) []kv.Condition {
	if len(conditions) == 0 {
		return conditions
	}

	out := make([]kv.Condition, len(conditions))

	for i, c := range conditions {
		c.Value = formatArg(c.Value, dialect)

		if len(c.Values) > 0 {
			vals := make([]any, len(c.Values))
			for j, v := range c.Values {
				vals[j] = formatArg(v, dialect)
			}

			c.Values = vals
		}

		out[i] = c
	}

	return out
}

func formatArg(v any, dialect sqlpkg.Dialect) any {
	if t, ok := v.(time.Time); ok {
		return dialect.FormatTime(t)
	}

	return v
}

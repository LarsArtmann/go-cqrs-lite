package view

import (
	"context"
	"fmt"
	"strings"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/kv/v4"
)

// Query runs a filtered, ordered, paginated query. See [kv.ViewQuery] for
// details. This implements [kv.ViewQuerier].
//
// Pagination: when [kv.ViewQuery.Keyset] is set, keyset (seek) pagination is
// used — a WHERE predicate restricts to rows strictly after the cursor, Offset
// is ignored, and Limit caps the page. Otherwise Offset/Limit pagination is
// used. Keyset pagination is stable under concurrent inserts and has constant
// performance regardless of depth.
func (s *SQLViewStore[V, K]) Query(ctx context.Context, q kv.ViewQuery) ([]*V, error) {
	var b strings.Builder

	fmt.Fprintf(&b, "SELECT %s FROM %s", s.selectCols, s.mapper.Table)

	// Resolve the effective ORDER BY clauses. q.Order (multi-column, per-column
	// direction) takes precedence over the legacy OrderBy+Desc pair.
	orderClauses := q.Order
	if len(orderClauses) == 0 {
		orderClauses = []kv.OrderClause{{Column: q.OrderBy, Desc: q.Desc}}
	}
	orderClauses = normaliseOrderClauses(orderClauses)

	// Accumulate WHERE fragments and their args, tracking a single sequential
	// placeholder index across all sources (conditions, raw, keyset). This is
	// required for Postgres-style $N placeholders; SQLite ? placeholders are
	// positional and index-agnostic, but correct numbering keeps both honest
	// (and fixes a latent bug where RawWhere args did not advance the index
	// before LIMIT).
	var whereFragments []string
	args := []any{}
	paramIdx := 1

	if condClause, condArgs := buildWhereClause(
		q.Conditions,
		s.Dialect.Placeholder,
	); condClause != "" {
		whereFragments = append(whereFragments, condClause)
		args = append(args, condArgs...)
		paramIdx += len(condArgs)
	}

	if q.RawWhere != "" {
		whereFragments = append(whereFragments, "("+q.RawWhere+")")
		args = append(args, q.RawArgs...)
		paramIdx += len(q.RawArgs)
	}

	if q.Keyset != nil && len(q.Keyset.Values) > 0 {
		cursorCols := q.Keyset.Columns
		if len(cursorCols) == 0 {
			cursorCols = defaultKeysetColumns(orderClauses)
		}

		if keyClause, keyArgs := buildKeysetClause(
			cursorCols, q.Keyset.Values, orderClauses, s.Dialect.Placeholder, paramIdx,
		); keyClause != "" {
			whereFragments = append(whereFragments, keyClause)
			args = append(args, keyArgs...)
			paramIdx += len(keyArgs)
		}
	}

	if len(whereFragments) > 0 {
		fmt.Fprintf(&b, " WHERE %s", strings.Join(whereFragments, " AND "))
	}

	fmt.Fprintf(&b, " ORDER BY %s", orderClauseSQL(orderClauses))

	// Pagination. Keyset pagination ignores Offset (the cursor is the
	// position); Limit caps the page size. When no keyset is set, fall back
	// to Limit/Offset.
	if q.Limit > 0 {
		fmt.Fprintf(&b, " LIMIT %s", s.Dialect.Placeholder(paramIdx))
		args = append(args, q.Limit)
		paramIdx++

		if q.Keyset == nil && q.Offset > 0 {
			fmt.Fprintf(&b, " OFFSET %s", s.Dialect.Placeholder(paramIdx))
			args = append(args, q.Offset)
		}
	} else if q.Keyset == nil && q.Offset > 0 {
		fmt.Fprintf(&b, " LIMIT %s OFFSET %s",
			s.Dialect.Placeholder(paramIdx), s.Dialect.Placeholder(paramIdx+1))
		args = append(args, -1, q.Offset)
	}

	rows, err := s.executor().QueryContext(ctx, b.String(), args...)
	if err != nil {
		return nil, errorfamily.WrapTransient(err, "storage.view.query", "query view records")
	}

	defer func() { _ = rows.Close() }()

	return s.scanRows(rows)
}

// normaliseOrderClauses returns a copy of clauses with empty Column names
// replaced by the key column. It does not mutate the input.
func normaliseOrderClauses(clauses []kv.OrderClause) []kv.OrderClause {
	out := make([]kv.OrderClause, len(clauses))
	for i, c := range clauses {
		if c.Column == "" {
			c.Column = keyColumnName
		}
		out[i] = c
	}
	return out
}

// defaultKeysetColumns derives the cursor columns from the effective ORDER BY
// clauses, guaranteeing a unique tiebreaker by appending the key column when
// it is not already the last sort column.
func defaultKeysetColumns(order []kv.OrderClause) []string {
	cols := make([]string, 0, len(order)+1)
	for _, c := range order {
		cols = append(cols, c.Column)
	}
	if len(cols) == 0 || cols[len(cols)-1] != keyColumnName {
		cols = append(cols, keyColumnName)
	}
	return cols
}

// orderClauseSQL renders clauses as "col1 DESC, col2 ASC".
func orderClauseSQL(order []kv.OrderClause) string {
	parts := make([]string, len(order))
	for i, c := range order {
		dir := "ASC"
		if c.Desc {
			dir = "DESC"
		}
		parts[i] = c.Column + " " + dir
	}
	return strings.Join(parts, ", ")
}

// buildKeysetClause builds a parenthesised, dialect-agnostic keyset predicate
// (without the WHERE keyword). For a cursor over columns c1..cN with values
// v1..vN, where column ci is descending when order[i].Desc is true, it emits
// the standard row-value comparison expansion:
//
//	(c1 < p1 OR (c1 = p2 AND (c2 < p3 OR (c2 = p4 AND ... cN < pK))))
//
// (with ">" for ascending columns). This is safe across SQLite and Postgres —
// no row-value syntax is used. startIdx is the 1-based placeholder index of
// the first value; returned args are in placeholder order.
func buildKeysetClause(
	cols []string, vals []any, order []kv.OrderClause,
	placeholder func(int) string, startIdx int,
) (string, []any) {
	n := len(cols)
	if nv := len(vals); nv < n {
		n = nv
	}
	if n == 0 {
		return "", nil
	}

	op := func(i int) string {
		if i < len(order) && order[i].Desc {
			return "<"
		}
		return ">"
	}

	idx := startIdx
	var args []any

	var expr func(i int) string
	expr = func(i int) string {
		col := cols[i]
		p := placeholder(idx)
		idx++
		args = append(args, vals[i])

		if i == n-1 {
			return col + " " + op(i) + " " + p
		}

		// Nested: col OP p OR (col = p AND <inner>)
		pEq := placeholder(idx)
		idx++
		args = append(args, vals[i])

		return col + " " + op(i) + " " + p + " OR (" + col + " = " + pEq + " AND " + expr(i+1) + ")"
	}

	return "(" + expr(0) + ")", args
}

// QueryByTombstone implements [kv.TombstoneQuerier]. When a TombstoneColumn is
// configured in the [ViewMapper], it pushes the tombstone filter to SQL.
// When no TombstoneColumn is set, it returns all records (the caller should
// apply Go-level filtering as a fallback).
func (s *SQLViewStore[V, K]) QueryByTombstone(
	ctx context.Context,
	excludeTombstoned, onlyTombstoned bool,
) ([]*V, error) {
	if s.mapper.TombstoneColumn == "" {
		return s.Scan(ctx, nil)
	}

	q := kv.ViewQuery{OrderBy: keyColumnName}

	col := s.mapper.TombstoneColumn

	if onlyTombstoned {
		q.Conditions = []kv.Condition{{Column: col, Op: kv.OpNeq, Value: 0}}
	} else if excludeTombstoned {
		q.Conditions = []kv.Condition{{Column: col, Op: kv.OpEq, Value: 0}}
	}

	return s.Query(ctx, q)
}

// Compile-time interface assertions.
var (
	_ kv.ViewStore[any, dummyViewKey]       = (*SQLViewStore[any, dummyViewKey])(nil)
	_ kv.ViewQuerier[any]                   = (*SQLViewStore[any, dummyViewKey])(nil)
	_ kv.TombstoneQuerier[any]              = (*SQLViewStore[any, dummyViewKey])(nil)
	_ kv.ViewCounter[any]                   = (*SQLViewStore[any, dummyViewKey])(nil)
	_ kv.ViewResetter[any]                  = (*SQLViewStore[any, dummyViewKey])(nil)
	_ kv.ViewBatchSetter[any, dummyViewKey] = (*SQLViewStore[any, dummyViewKey])(nil)
)

type dummyViewKey string

func (dummyViewKey) String() string { return "" }

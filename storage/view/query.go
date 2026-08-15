package view

import (
	"context"
	"fmt"
	"strings"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/kv/v4"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
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

	if err := s.validateQuery(q, orderClauses); err != nil {
		return nil, err
	}

	// Accumulate WHERE fragments and their args, tracking a single sequential
	// placeholder index across all sources (conditions, raw, keyset). This is
	// required for Postgres-style $N placeholders; SQLite ? placeholders are
	// positional and index-agnostic, but correct numbering keeps both honest
	// (and fixes a latent bug where RawWhere args did not advance the index
	// before LIMIT).
	var whereFragments []string
	args := []any{}
	paramIdx := 1

	if condClause, condArgs, err := sqlpkg.BuildWhereClauseChecked(
		q.Conditions,
		s.Dialect.Placeholder,
	); err != nil {
		return nil, errorfamily.WrapRejection(err, "storage.view.conditions", "validate query conditions")
	} else if condClause != "" {
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

	fmt.Fprintf(&b, " ORDER BY %s", orderClauseSQL(orderClauses, s.Dialect.QuoteIdentifier))

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

	defer sqlpkg.CloseRows(rows)

	return s.scanRows(rows)
}

// validateQuery rejects column references that are not declared mapper
// columns (fail-closed against request-derived names reaching SQL) and
// operators outside the supported set. It covers filter conditions, ORDER BY
// clauses, and keyset cursor columns.
func (s *SQLViewStore[V, K]) validateQuery(q kv.ViewQuery, order []kv.OrderClause) error {
	if err := s.validateConditions(q.Conditions); err != nil {
		return err
	}

	for i, clause := range order {
		if _, ok := s.allowedCols[clause.Column]; !ok {
			return s.rejectUnknownColumn(clause.Column, fmt.Sprintf("order clause %d", i))
		}
	}

	if q.Keyset == nil {
		return nil
	}

	for i, col := range q.Keyset.Columns {
		if _, ok := s.allowedCols[col]; !ok {
			return s.rejectUnknownColumn(col, fmt.Sprintf("keyset column %d", i))
		}
	}

	return nil
}

// validateConditions rejects conditions whose column is not a declared mapper
// column or whose operator is not supported.
func (s *SQLViewStore[V, K]) validateConditions(conditions []kv.Condition) error {
	for i, cond := range conditions {
		if _, ok := s.allowedCols[cond.Column]; !ok {
			return s.rejectUnknownColumn(cond.Column, fmt.Sprintf("condition %d", i))
		}

		if !sqlpkg.ValidateOperator(cond.Op) {
			return errorfamily.NewRejection("storage.view.unsupported_operator",
				fmt.Sprintf("condition %d: unsupported operator %q", i, cond.Op))
		}
	}

	return nil
}

func (s *SQLViewStore[V, K]) rejectUnknownColumn(column, source string) error {
	return errorfamily.NewRejection("storage.view.unknown_column",
		fmt.Sprintf("%s: column %q is not declared in the view mapper", source, column))
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

// orderClauseSQL renders clauses as "col1 DESC, col2 ASC". Column names are
// passed through quote so dialects that require delimiters (MySQL backticks)
// emit them; callers must have validated the names already.
func orderClauseSQL(order []kv.OrderClause, quote func(string) string) string {
	parts := make([]string, len(order))
	for i, c := range order {
		dir := "ASC"
		if c.Desc {
			dir = "DESC"
		}
		parts[i] = quote(c.Column) + " " + dir
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

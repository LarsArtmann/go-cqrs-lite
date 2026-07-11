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
func (s *SQLViewStore[V, K]) Query(ctx context.Context, q kv.ViewQuery) ([]*V, error) {
	var b strings.Builder

	fmt.Fprintf(&b, "SELECT %s FROM %s", s.selectCols, s.mapper.Table)

	whereClause, whereArgs := buildWhereClause(q.Conditions, s.Dialect.Placeholder)
	args := whereArgs
	paramIdx := 1 + len(args)

	if whereClause != "" {
		fmt.Fprintf(&b, " WHERE %s", whereClause)
	}

	orderCol := q.OrderBy
	if orderCol == "" {
		orderCol = keyColumnName
	}

	dir := "ASC"
	if q.Desc {
		dir = "DESC"
	}

	fmt.Fprintf(&b, " ORDER BY %s %s", orderCol, dir)

	if q.Limit > 0 {
		fmt.Fprintf(&b, " LIMIT %s", s.Dialect.Placeholder(paramIdx))
		args = append(args, q.Limit)
		paramIdx++

		if q.Offset > 0 {
			fmt.Fprintf(&b, " OFFSET %s", s.Dialect.Placeholder(paramIdx))
			args = append(args, q.Offset)
		}
	} else if q.Offset > 0 {
		fmt.Fprintf(&b, " LIMIT %s OFFSET %s",
			s.Dialect.Placeholder(paramIdx), s.Dialect.Placeholder(paramIdx+1))
		args = append(args, -1, q.Offset)
	}

	rows, err := s.DB.QueryContext(ctx, b.String(), args...)
	if err != nil {
		return nil, errorfamily.WrapTransient(err, "storage.view.query", "query view records")
	}

	defer func() { _ = rows.Close() }()

	return s.scanRows(rows)
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

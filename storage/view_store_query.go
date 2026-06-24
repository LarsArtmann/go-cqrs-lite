package storage

import (
	"context"
	"fmt"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/kv/v3"
)

// Query runs a filtered, ordered, paginated query. See [kv.ViewQuery] for
// details. This implements [kv.ViewQuerier].
func (s *SQLViewStore[V, K]) Query(ctx context.Context, q kv.ViewQuery) ([]*V, error) {
	var b strings.Builder

	fmt.Fprintf(&b, "SELECT %s FROM %s", s.selectCols, s.mapper.Table)

	args := make([]any, 0, len(q.Args))
	paramIdx := 1

	if q.Where != "" {
		fmt.Fprintf(&b, " WHERE %s", q.Where)
		args = append(args, q.Args...)
		paramIdx += len(q.Args)
	}

	orderCol := q.OrderBy
	if orderCol == "" {
		orderCol = "key"
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
		return nil, fmt.Errorf("view-store: query: %w", err)
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

	q := kv.ViewQuery{OrderBy: "key"}

	col := s.mapper.TombstoneColumn

	if onlyTombstoned {
		q.Where = col + " != 0"
	} else if excludeTombstoned {
		q.Where = col + " = 0"
	}

	return s.Query(ctx, q)
}

// Compile-time interface assertions.
var (
	_ kv.ViewStore[any, dummyViewKey] = (*SQLViewStore[any, dummyViewKey])(nil)
	_ kv.ViewQuerier[any]             = (*SQLViewStore[any, dummyViewKey])(nil)
	_ kv.TombstoneQuerier[any]        = (*SQLViewStore[any, dummyViewKey])(nil)
)

type dummyViewKey string

func (dummyViewKey) String() string { return "" }

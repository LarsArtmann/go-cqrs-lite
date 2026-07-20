package view

import (
	"context"
	"fmt"
	"strings"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/kv/v4"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

// Count returns the number of records matching the query's conditions, without
// loading any rows. This implements [kv.ViewCounter].
//
// When q.Conditions is empty, all records are counted. The OrderBy, Limit, and
// Offset fields of q are ignored — only Conditions are used.
func (s *SQLViewStore[V, K]) Count(ctx context.Context, q kv.ViewQuery) (int64, error) {
	var b strings.Builder

	fmt.Fprintf(&b, "SELECT COUNT(*) FROM %s", s.mapper.Table)

	whereClause, args := sqlpkg.BuildWhereClause(q.Conditions, s.Dialect.Placeholder)

	if whereClause != "" {
		fmt.Fprintf(&b, " WHERE %s", whereClause)
	}

	if q.RawWhere != "" {
		if whereClause != "" {
			fmt.Fprintf(&b, " AND (%s)", q.RawWhere)
		} else {
			fmt.Fprintf(&b, " WHERE %s", q.RawWhere)
		}

		args = append(args, q.RawArgs...)
	}

	var count int64

	err := s.executor().QueryRowContext(ctx, b.String(), args...).Scan(&count)
	if err != nil {
		return 0, errorfamily.WrapTransient(err, "storage.view.count", "count records")
	}

	return count, nil
}

// (buildWhereClause moved to storage/sql.BuildWhereClause — shared across
// relational and view sub-packages.)

package metaengine

import (
	"context"
	"database/sql"
	"fmt"
)

// drainQuery runs the query, closes the rows when done, and drains every row
// through scan into a result slice. The label is the error prefix shared by
// the query and rows-iteration errors; scan wraps its own errors. Shared by
// the single-column and grouped-aggregate scan helpers below.
func drainQuery[T any](
	ctx context.Context,
	q SQLExec,
	query string,
	args []any,
	label string,
	scan func(rows *sql.Rows) (T, error),
) ([]T, error) {
	rows, err := q.QueryContext(ctx, query, args...) //nolint:sqlclosecheck
	if err != nil {
		return nil, fmt.Errorf("%s: %w", label, err)
	}

	defer DeferClose(rows)

	var result []T

	for rows.Next() {
		item, err := scan(rows)
		if err != nil {
			return nil, err
		}

		result = append(result, item)
	}

	if err := rows.Err(); err != nil {
		return result, fmt.Errorf("%s: %w", label, err)
	}

	return result, nil
}

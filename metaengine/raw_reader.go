package metaengine

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// --- RawValueReader ---

// GetRawValue reads the raw JSON bytes for a key without decoding to any.
// ExecuteTyped prefers this path for point lookups, decoding directly to R
// (1 JSON op instead of 3).
func (e *sqliteEngine) GetRawValue(ctx context.Context, col string, key any) ([]byte, bool, error) {
	var valStr string

	var err error

	if plan, ok := e.plans[col]; ok {
		err = e.db.QueryRowContext(ctx,
			fmt.Sprintf("SELECT value FROM %s WHERE key = ?", plan.Table),
			encodeKey(key)).Scan(&valStr)
	} else {
		err = e.db.QueryRowContext(ctx, e.queries.mapGet, col, encodeKey(key)).Scan(&valStr)
	}

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}

		return nil, false, err //nolint:wrapcheck // passthrough
	}

	return unsafeStringToBytes(valStr), true, nil
}

// --- RawScanReader ---

// ScanRawValues scans collection values as raw JSON bytes without decoding.
// ExecuteTyped prefers this path for filtered scans, decoding each row directly
// to the target type (1 JSON op per row instead of 3).
func (e *sqliteEngine) ScanRawValues(
	ctx context.Context,
	col string,
	filters []FilterSpec,
	sort *SortSpec,
	cursor any,
	limit int,
) ([][]byte, error) {
	if plan, ok := e.plans[col]; ok {
		return scanRawPlanned(ctx, e.db, plan, filters, sort, cursor, limit)
	}

	return scanRawStandard(ctx, e.db, col, filters, sort, cursor, limit)
}

func scanRawStandard(
	ctx context.Context,
	db *sql.DB,
	col string,
	filters []FilterSpec,
	sort *SortSpec,
	cursor any,
	limit int,
) ([][]byte, error) {
	var b strings.Builder

	args := []any{col}

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

	if sort != nil {
		path := jsonPath(sort.Column)

		b.WriteString(` ORDER BY json_extract(value, '`)
		b.WriteString(path)
		b.WriteString(`')`)

		if sort.Desc {
			b.WriteString(` DESC`)
		}
	}

	if limit > 0 {
		b.WriteString(` LIMIT ?`)
		args = append(args, limit+1)
	}

	return scanRawRows(ctx, db, b.String(), args...)
}

func scanRawPlanned(
	ctx context.Context,
	db *sql.DB,
	plan LayoutPlan,
	filters []FilterSpec,
	sort *SortSpec,
	cursor any,
	limit int,
) ([][]byte, error) {
	var b strings.Builder

	args := []any{}

	fmt.Fprintf(&b, "SELECT value FROM %s", plan.Table)

	for i, f := range filters {
		if i == 0 {
			b.WriteString(" WHERE ")
		} else {
			b.WriteString(" AND ")
		}

		fmt.Fprintf(&b, "%s %s ?", f.Column, string(f.Op))
		args = append(args, f.Value)
	}

	if sort != nil && cursor != nil {
		if len(filters) == 0 {
			b.WriteString(" WHERE ")
		} else {
			b.WriteString(" AND ")
		}

		op := ">"
		if sort.Desc {
			op = "<"
		}

		fmt.Fprintf(&b, "%s %s ?", sort.Column, op)
		args = append(args, cursor)
	}

	if sort != nil {
		fmt.Fprintf(&b, " ORDER BY %s", sort.Column)

		if sort.Desc {
			b.WriteString(" DESC")
		}
	}

	if limit > 0 {
		b.WriteString(" LIMIT ?")
		args = append(args, limit+1)
	}

	return scanRawRows(ctx, db, b.String(), args...)
}

func scanRawRows(ctx context.Context, db *sql.DB, query string, args ...any) ([][]byte, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err //nolint:wrapcheck // passthrough
	}

	defer func() { _ = rows.Close() }()

	var out [][]byte

	for rows.Next() {
		var valStr string

		if err := rows.Scan(&valStr); err != nil {
			return nil, err //nolint:wrapcheck // passthrough
		}

		out = append(out, unsafeStringToBytes(valStr))
	}

	return out, rows.Err() //nolint:wrapcheck // passthrough
}

// unsafeStringToBytes converts a string to []byte without copying. The result
// shares the string's backing array — safe for read-only use (the caller does
// not mutate the bytes). This avoids allocating a new []byte for every row.
func unsafeStringToBytes(s string) []byte {
	if len(s) == 0 {
		return nil
	}

	return []byte(s) //nolint:gocritic // intentional — json.Unmarshal accepts []byte
}

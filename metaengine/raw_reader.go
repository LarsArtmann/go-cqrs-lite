package metaengine

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// jsonValue carries raw JSON bytes from a SQL engine, deferring decode until
// the typed result is reconstructed. ExecuteTyped and TypedReader recognize
// this type and unmarshal directly into the target type, avoiding the
// intermediate map[string]any + reify round-trip (3 JSON ops → 1).
//
// This is an internal optimization: memory engines return typed Go values
// directly (no jsonValue), and the closure-based MapScan path returns decoded
// any values (filtering needs them). Only pushdown paths (PushdownMapScan,
// GetRawValue, ScanRawValues) return jsonValue.
type jsonValue []byte

// --- RawValueReader ---

// GetRawValue reads the raw JSON bytes for a key without decoding to any.
// ExecuteTyped prefers this path for point lookups, decoding directly to R
// (1 JSON op instead of 3).
func (e *sqliteEngine) GetRawValue(ctx context.Context, col string, key any) ([]byte, bool, error) {
	var valStr string

	var err error

	if plan, ok := e.plans[col]; ok {
		err = e.xc().queryRow(ctx,
			fmt.Sprintf("SELECT value FROM %s WHERE key = ?", quoteIdent(plan.Table)),
			encodeKey(key)).Scan(&valStr)
	} else {
		err = e.xc().queryRow(ctx, e.queries.mapGet, col, encodeKey(key)).Scan(&valStr)
	}

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}

		return nil, false, err //nolint:wrapcheck // passthrough
	}

	return stringToBytes(valStr), true, nil
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
		return scanRawPlanned(ctx, e.xd(), plan, filters, sort, cursor, limit)
	}

	return scanRawStandard(ctx, e.xd(), col, filters, sort, cursor, limit)
}

func scanRawStandard(
	ctx context.Context,
	db dbExec,
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
		appendStandardFilter(&b, &args, f)
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

// buildPlannedSelectQuery constructs a parameterised SELECT for a layout-planned
// table: SELECT value FROM <table> [WHERE filters] [AND cursor] [ORDER BY] [LIMIT].
// The query string and args are shared by scanRawPlanned (raw bytes) and
// pushdownMapScanPlanned (decoded values).
func buildPlannedSelectQuery(
	plan LayoutPlan,
	filters []FilterSpec,
	sort *SortSpec,
	cursor any,
	limit int,
) (string, []any) {
	var b strings.Builder

	args := []any{}

	fmt.Fprintf(&b, "SELECT value FROM %s", quoteIdent(plan.Table))

	whereStarted := false

	for _, f := range filters {
		appendPlannedFilter(&b, &args, f, &whereStarted)
	}

	if sort != nil && cursor != nil {
		if !whereStarted {
			b.WriteString(" WHERE ")

			whereStarted = true
		} else {
			b.WriteString(" AND ")
		}

		op := ">"
		if sort.Desc {
			op = "<"
		}

		fmt.Fprintf(&b, "%s %s ?", quoteIdent(sort.Column), op)

		args = append(args, cursor)
	}

	if sort != nil {
		fmt.Fprintf(&b, " ORDER BY %s", quoteIdent(sort.Column))

		if sort.Desc {
			b.WriteString(" DESC")
		}
	}

	if limit > 0 {
		b.WriteString(" LIMIT ?")

		args = append(args, limit+1)
	}

	return b.String(), args
}

func scanRawPlanned(
	ctx context.Context,
	db dbExec,
	plan LayoutPlan,
	filters []FilterSpec,
	sort *SortSpec,
	cursor any,
	limit int,
) ([][]byte, error) {
	query, args := buildPlannedSelectQuery(plan, filters, sort, cursor, limit)

	return scanRawRows(ctx, db, query, args...)
}

// scanSingleColumn executes query, scans the single string column of each row,
// and applies transform to produce the output slice. Shared by scanRawRows
// (raw bytes) and scanJSONValues (decoded values).
func scanSingleColumn[T any](
	ctx context.Context,
	db dbExec,
	query string,
	transform func(string) T,
	args ...any,
) ([]T, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err //nolint:wrapcheck // passthrough
	}

	defer func() { _ = rows.Close() }()

	var out []T

	for rows.Next() {
		var valStr string

		if err := rows.Scan(&valStr); err != nil {
			return nil, err //nolint:wrapcheck // passthrough
		}

		out = append(out, transform(valStr))
	}

	return out, rows.Err() //nolint:wrapcheck // passthrough
}

func scanRawRows(ctx context.Context, db dbExec, query string, args ...any) ([][]byte, error) {
	return scanSingleColumn(ctx, db, query, stringToBytes, args...)
}

// stringToBytes converts a string to []byte without copying. The result
// shares the string's backing array — safe for read-only use (the caller does
// not mutate the bytes). This avoids allocating a new []byte for every row.
func stringToBytes(s string) []byte {
	if len(s) == 0 {
		return nil
	}

	return []byte(s)
}

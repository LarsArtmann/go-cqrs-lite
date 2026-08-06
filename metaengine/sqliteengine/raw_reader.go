package sqliteengine

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// --- RawValueReader ---

func (e *sqliteEngine) GetRawValue(ctx context.Context, col string, key any) ([]byte, bool, error) {
	var valStr string

	var err error

	if plan, ok := e.plans[col]; ok {
		err = e.xc().queryRow(ctx,
			fmt.Sprintf("SELECT value FROM %s WHERE key = ?", metaengine.QuoteIdent(plan.Table)),
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

func (e *sqliteEngine) ScanRawValues(
	ctx context.Context,
	col string,
	filters []metaengine.FilterSpec,
	sort *metaengine.SortSpec,
	cursor any,
	limit int,
) (metaengine.RawScanResult, error) {
	var rows [][]byte

	var err error

	if plan, ok := e.plans[col]; ok {
		rows, err = scanRawPlanned(ctx, e.xd(), plan, filters, sort, cursor, limit)
	} else {
		rows, err = scanRawStandard(ctx, e.xd(), col, filters, sort, cursor, limit)
	}

	if err != nil {
		return metaengine.RawScanResult{}, err
	}

	hasMore := limit > 0 && len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}

	return metaengine.RawScanResult{Items: rows, HasMore: hasMore}, nil
}

func scanRawStandard(
	ctx context.Context,
	db dbExec,
	col string,
	filters []metaengine.FilterSpec,
	sort *metaengine.SortSpec,
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

func buildPlannedSelectQuery(
	plan metaengine.LayoutPlan,
	filters []metaengine.FilterSpec,
	sort *metaengine.SortSpec,
	cursor any,
	limit int,
) (string, []any) {
	var b strings.Builder

	args := []any{}

	fmt.Fprintf(&b, "SELECT value FROM %s", metaengine.QuoteIdent(plan.Table))

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

		fmt.Fprintf(&b, "%s %s ?", metaengine.QuoteIdent(sort.Column), op)

		args = append(args, cursor)
	}

	if sort != nil {
		fmt.Fprintf(&b, " ORDER BY %s", metaengine.QuoteIdent(sort.Column))

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
	plan metaengine.LayoutPlan,
	filters []metaengine.FilterSpec,
	sort *metaengine.SortSpec,
	cursor any,
	limit int,
) ([][]byte, error) {
	query, args := buildPlannedSelectQuery(plan, filters, sort, cursor, limit)

	return scanRawRows(ctx, db, query, args...)
}

func scanSingleColumn(
	ctx context.Context,
	db dbExec,
	query string,
	decode func(valStr string) any,
	args ...any,
) ([]any, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err //nolint:wrapcheck // passthrough
	}

	defer func() { _ = rows.Close() }()

	var result []any

	for rows.Next() {
		var valStr string

		if err := rows.Scan(&valStr); err != nil {
			return nil, err //nolint:wrapcheck // passthrough
		}

		result = append(result, decode(valStr))
	}

	return result, rows.Err() //nolint:wrapcheck // passthrough
}

func scanRawRows(ctx context.Context, db dbExec, query string, args ...any) ([][]byte, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err //nolint:wrapcheck // passthrough
	}

	defer func() { _ = rows.Close() }()

	var result [][]byte

	for rows.Next() {
		var valStr string

		if err := rows.Scan(&valStr); err != nil {
			return nil, err //nolint:wrapcheck // passthrough
		}

		result = append(result, stringToBytes(valStr))
	}

	return result, rows.Err() //nolint:wrapcheck // passthrough
}

func stringToBytes(s string) []byte {
	return []byte(s)
}

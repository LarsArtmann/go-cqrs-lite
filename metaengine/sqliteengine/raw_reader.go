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
		err = e.xc(ctx).queryRow(ctx,
			fmt.Sprintf("SELECT value FROM %s WHERE key = ?", metaengine.QuoteIdent(plan.Table)),
			encodeKey(key)).Scan(&valStr)
	} else {
		err = e.xc(ctx).queryRow(ctx, e.queries.mapGet, col, encodeKey(key)).Scan(&valStr)
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

// ScanRawValues scans raw JSON rows with keyset pagination. On sorted scans it
// also issues the compound continuation cursor (SortKeyCursor over the last
// returned row's sort value + key) so tie-heavy datasets paginate exactly
// once; the same predicate consumes both legacy scalar cursors and compound
// ones pushed back as the cursor argument.
func (e *sqliteEngine) ScanRawValues(
	ctx context.Context,
	col string,
	filters []metaengine.FilterSpec,
	sort *metaengine.SortSpec,
	cursor any,
	limit int,
) (metaengine.RawScanResult, error) {
	var rows [][]byte

	var keys []string

	var err error

	if plan, ok := e.plans[col]; ok {
		rows, keys, err = scanRawPlanned(ctx, e.xd(ctx), plan, filters, sort, cursor, limit)
	} else {
		rows, keys, err = scanRawStandard(ctx, e.xd(ctx), col, filters, sort, cursor, limit)
	}

	if err != nil {
		return metaengine.RawScanResult{}, err
	}

	hasMore := limit > 0 && len(rows) > limit
	if hasMore {
		rows = rows[:limit]
		keys = keys[:limit]
	}

	var next any
	if sort != nil && len(rows) > 0 {
		next = metaengine.LastJSONRowCursor(rows[len(rows)-1], sort.Column, keys[len(keys)-1])
	}

	return metaengine.RawScanResult{Items: rows, HasMore: hasMore, NextCursor: next}, nil
}

// jsonSortExpr renders the sort expression for standard (meta_map) scans: the
// stored JSON document's field via json_extract.
func jsonSortExpr(sort *metaengine.SortSpec) string {
	return fmt.Sprintf("json_extract(value, '%s')", jsonPath(sort.Column))
}

// metaMapKeyExpr is the keyset tiebreak column of meta_map. It stays unquoted
// to match the historical `WHERE key = ?` statements.
const metaMapKeyExpr = "key"

// buildStandardScanQuery builds the SELECT for standard (meta_map) scans:
// value + key columns, pushed filters, the keyset cursor predicate, and the
// deterministic ORDER BY (sort column, then key ascending — the tiebreak the
// compound cursor predicate mirrors). Shared by the raw and pushdown scan
// surfaces so both consume and issue identical cursors.
func buildStandardScanQuery(
	col string,
	filters []metaengine.FilterSpec,
	sort *metaengine.SortSpec,
	cursor any,
	limit int,
) (string, []any) {
	var b strings.Builder

	args := []any{col}

	b.WriteString(`SELECT value, key FROM meta_map WHERE collection = ?`)

	for _, f := range filters {
		appendStandardFilter(&b, &args, f)
	}

	if sort != nil && cursor != nil {
		started := true // `collection = ?` already opened the WHERE clause
		metaengine.AppendKeysetCursorPredicate(&b, &args, &started,
			jsonSortExpr(sort), metaMapKeyExpr, sort.Desc, cursor,
			metaengine.QuestionPlaceholders)
	}

	if sort != nil {
		metaengine.AppendKeysetOrder(&b, jsonSortExpr(sort), metaMapKeyExpr, sort.Desc)
	}

	if limit > 0 {
		b.WriteString(` LIMIT ?`)

		args = append(args, limit+1)
	}

	return b.String(), args
}

func scanRawStandard(
	ctx context.Context,
	db metaengine.SQLExec,
	col string,
	filters []metaengine.FilterSpec,
	sort *metaengine.SortSpec,
	cursor any,
	limit int,
) ([][]byte, []string, error) {
	query, args := buildStandardScanQuery(col, filters, sort, cursor, limit)

	return scanRawRowsWithKeys(ctx, db, query, args...)
}

// buildPlannedSelectQuery builds the SELECT for planned-table scans: value +
// key columns, direct column filters, the keyset cursor predicate (compound
// SortKeyCursor aware), and the deterministic ORDER BY with key tiebreak.
// art-dupl:accept cross-module SQL builder pattern — separate go.mod
func buildPlannedSelectQuery(
	plan metaengine.LayoutPlan,
	filters []metaengine.FilterSpec,
	sort *metaengine.SortSpec,
	cursor any,
	limit int,
) (string, []any, error) {
	if err := metaengine.ValidateFilterSpecs(filters); err != nil {
		return "", nil, err
	}

	keyExpr := metaengine.QuoteIdent("key")

	var b strings.Builder

	args := []any{}

	fmt.Fprintf(&b, "SELECT value, %s FROM %s", keyExpr, metaengine.QuoteIdent(plan.Table))

	whereStarted := false

	for _, f := range filters {
		appendPlannedFilter(&b, &args, f, &whereStarted)
	}

	if sort != nil && cursor != nil {
		metaengine.AppendKeysetCursorPredicate(&b, &args, &whereStarted,
			metaengine.QuoteIdent(sort.Column), keyExpr, sort.Desc, cursor,
			metaengine.QuestionPlaceholders)
	}

	if sort != nil {
		metaengine.AppendKeysetOrder(&b, metaengine.QuoteIdent(sort.Column), keyExpr, sort.Desc)
	}

	if limit > 0 {
		b.WriteString(` LIMIT ?`)

		args = append(args, limit+1)
	}

	return b.String(), args, nil
}

func scanRawPlanned(
	ctx context.Context,
	db metaengine.SQLExec,
	plan metaengine.LayoutPlan,
	filters []metaengine.FilterSpec,
	sort *metaengine.SortSpec,
	cursor any,
	limit int,
) ([][]byte, []string, error) {
	query, args, err := buildPlannedSelectQuery(plan, filters, sort, cursor, limit)
	if err != nil {
		return nil, nil, err
	}

	return scanRawRowsWithKeys(ctx, db, query, args...)
}

func scanSingleColumn(
	ctx context.Context,
	db metaengine.SQLExec,
	query string,
	decode func(valStr string) any,
	args ...any,
) ([]any, error) {
	rows, err := db.QueryContext(ctx, query, args...) //nolint:sqlclosecheck
	if err != nil {
		return nil, err //nolint:wrapcheck // passthrough
	}

	defer metaengine.DeferClose(rows)

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

// scanRawRowsWithKeys scans two-column (value, key) queries into raw JSON
// bytes plus their key columns — the keyset cursor's Key source.
func scanRawRowsWithKeys(
	ctx context.Context,
	db metaengine.SQLExec,
	query string,
	args ...any,
) ([][]byte, []string, error) {
	rows, err := db.QueryContext(ctx, query, args...) //nolint:sqlclosecheck
	if err != nil {
		return nil, nil, err //nolint:wrapcheck // passthrough
	}

	defer metaengine.DeferClose(rows)

	var values [][]byte

	var keys []string

	for rows.Next() {
		var valStr, keyStr string

		if err := rows.Scan(&valStr, &keyStr); err != nil {
			return nil, nil, err //nolint:wrapcheck // passthrough
		}

		values = append(values, stringToBytes(valStr))
		keys = append(keys, keyStr)
	}

	return values, keys, rows.Err() //nolint:wrapcheck // passthrough
}

func stringToBytes(s string) []byte {
	return []byte(s)
}

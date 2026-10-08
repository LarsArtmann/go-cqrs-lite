//go:build cgo

package duckdbengine

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"strings"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// PushdownMapScan pushes WHERE/ORDER BY/LIMIT into DuckDB using json_extract,
// avoiding the full-table load that MapScan performs. Filters become
// json_extract(value, '$.field') = $N::json, sort becomes
// ORDER BY json_extract(...), and limit becomes LIMIT.
//
// Using json_extract (returns JSON) instead of json_extract_string (text)
// preserves the native JSON type, so numeric comparisons work correctly:
// 5 > 3 (numeric), not "5" > "3" (lexical).
//
// Keyset pagination: the cursor predicate mirrors the shared SQL keyset
// contract — compound [SortKeyCursor] cursors compare (sort op, sort =, key >)
// with the key tiebreak ALWAYS ascending, and ORDER BY appends the key column
// so tie-heavy datasets paginate exactly once. DuckDB's ::json casts apply to
// the sort-value binds only (the key column is VARCHAR), which is why this
// dialect renders the predicate inline instead of the placeholder-generic
// core helper. LIMIT is n+1 for has-more detection; the last included row
// mints the next compound cursor.
func (e *duckdbEngine) PushdownMapScan(
	ctx context.Context,
	collection string,
	filters []metaengine.FilterSpec,
	sort *metaengine.SortSpec,
	cursor any,
	limit int,
) (metaengine.ScanResult, error) {
	if plan, ok := e.lookupPlan(collection); ok {
		return e.pushdownMapScanPlanned(ctx, plan, filters, sort, cursor, limit)
	}

	var b strings.Builder //art-dupl:accept cross-module SQL builder pattern — separate go.mod
	args := []any{collection}

	b.WriteString(`SELECT value, "key" FROM meta_map WHERE collection = $1`)

	for _, f := range filters {
		path := jsonPath(f.Column)

		if f.Op == metaengine.FilterIn {
			values, ok := f.Value.([]any)
			if !ok || len(values) == 0 {
				continue
			}

			placeholders := make([]string, len(values))
			for i, v := range values {
				jb, _ := json.Marshal(v)
				placeholders[i] = fmt.Sprintf("$%d::json", len(args)+1)
				args = append(args, string(jb))
			}

			fmt.Fprintf(&b, ` AND json_extract(value, '%s') IN (%s)`,
				path, strings.Join(placeholders, ", "))
		} else {
			jb, _ := json.Marshal(f.Value)
			fmt.Fprintf(&b, ` AND json_extract(value, '%s') %s $%d::json`,
				path, string(f.Op), len(args)+1)
			args = append(args, string(jb))
		}
	}

	if sort != nil && cursor != nil {
		sortExpr := fmt.Sprintf("json_extract(value, '%s')", jsonPath(sort.Column))

		op := ">"
		if sort.Desc {
			op = "<"
		}

		if skc, ok := cursor.(metaengine.SortKeyCursor); ok {
			jb, _ := json.Marshal(skc.Sort)

			fmt.Fprintf(&b, ` AND (%s %s $%d::json OR %s = $%d::json AND "key" > $%d)`,
				sortExpr, op, len(args)+1,
				sortExpr, len(args)+2,
				len(args)+3)

			args = append(args, string(jb), string(jb), string(skc.Key))
		} else {
			jb, _ := json.Marshal(cursor)

			fmt.Fprintf(&b, ` AND %s %s $%d::json`, sortExpr, op, len(args)+1)
			args = append(args, string(jb))
		}
	}

	if sort != nil { //art-dupl:accept cross-module SQL dialect ORDER BY rendering — separate go.mod
		fmt.Fprintf(&b, ` ORDER BY json_extract(value, '%s')`, jsonPath(sort.Column))

		if sort.Desc {
			b.WriteString(` DESC`)
		}

		b.WriteString(`, "key"`)
	}

	if limit > 0 {
		fmt.Fprintf(&b, ` LIMIT %d`, limit+1)
	}

	rows, keys, err := scanDuckDBJSONValuesWithKeys(ctx, e.conn(ctx), b.String(), args...)
	if err != nil {
		return metaengine.ScanResult{}, err
	}

	hasMore := limit > 0 && len(rows) > limit
	if hasMore {
		rows = rows[:limit]
		keys = keys[:limit]
	}

	var next any
	if sort != nil && len(rows) > 0 {
		next = metaengine.LastDecodedRowCursor(
			rows[len(rows)-1],
			sort.Column,
			[]byte(keys[len(keys)-1]),
		)
	}

	return metaengine.ScanResult{Items: rows, HasMore: hasMore, NextCursor: next}, nil
}

// pushdownMapScanPlanned serves sorted scans from a planned table: the query
// comes from buildPlannedSelectQuery (shared keyset predicate + order), the
// scan yields values with their keys, and the last included row mints the
// compound continuation cursor.
func (e *duckdbEngine) pushdownMapScanPlanned(
	ctx context.Context,
	plan metaengine.LayoutPlan,
	filters []metaengine.FilterSpec,
	sort *metaengine.SortSpec,
	cursor any,
	limit int,
) (metaengine.ScanResult, error) {
	query, args := buildPlannedSelectQuery(plan, filters, sort, cursor, limit)

	rows, keys, err := scanDuckDBJSONValuesWithKeys(ctx, e.conn(ctx), query, args...)
	if err != nil {
		return metaengine.ScanResult{}, err
	}

	hasMore := limit > 0 && len(rows) > limit
	if hasMore {
		rows = rows[:limit]
		keys = keys[:limit]
	}

	var next any
	if sort != nil && len(rows) > 0 {
		next = metaengine.LastDecodedRowCursor(
			rows[len(rows)-1],
			sort.Column,
			[]byte(keys[len(keys)-1]),
		)
	}

	return metaengine.ScanResult{Items: rows, HasMore: hasMore, NextCursor: next}, nil
}

// jsonPath converts a field name to a DuckDB JSON path.
// E.g. "status" → "$.status". Single quotes are escaped.
func jsonPath(field string) string {
	escaped := strings.ReplaceAll(field, "'", "''")
	return "$." + escaped
}

// scanDuckDBJSONValuesWithKeys executes the query and decodes each row's JSON
// value alongside its key column — the pair keyset scans need to tiebreak and
// mint compound cursors.
func scanDuckDBJSONValuesWithKeys(
	ctx context.Context,
	db metaengine.SQLExec,
	query string,
	args ...any,
) ([]any, []string, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("duckdbengine scan: %w", err)
	}

	defer metaengine.DeferClose(rows)
	//art-dupl:accept cross-module SQL engine pattern — separate go.mod

	var (
		result []any
		keys   []string
	)

	for rows.Next() {
		var raw, key string

		if err := rows.Scan(&raw, &key); err != nil {
			return nil, nil, fmt.Errorf("duckdbengine scan: row: %w", err)
		}

		var val any
		if err := json.Unmarshal([]byte(raw), &val); err != nil {
			return nil, nil, fmt.Errorf("duckdbengine scan: unmarshal: %w", err)
		}

		result = append(result, val)
		keys = append(keys, key)
	}

	if err := rows.Err(); err != nil {
		return result, keys, fmt.Errorf("duckdbengine scan: %w", err)
	}

	return result, keys, nil
}

//go:build cgo

package duckdbengine

import (
	"context"
	"database/sql"
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
// Keyset pagination: cursor becomes an additional WHERE clause on the sort
// column. LIMIT is n+1 for has-more detection.
func (e *duckdbEngine) PushdownMapScan(
	ctx context.Context,
	collection string,
	filters []metaengine.FilterSpec,
	sort *metaengine.SortSpec,
	cursor any,
	limit int,
) (metaengine.ScanResult, error) {
	var b strings.Builder
	args := []any{collection}

	b.WriteString(`SELECT value FROM meta_map WHERE collection = $1`)

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
		path := jsonPath(sort.Column)
		op := ">"
		if sort.Desc {
			op = "<"
		}

		jb, _ := json.Marshal(cursor)
		fmt.Fprintf(&b, ` AND json_extract(value, '%s') %s $%d::json`,
			path, op, len(args)+1)
		args = append(args, string(jb))
	}

	if sort != nil {
		path := jsonPath(sort.Column)
		fmt.Fprintf(&b, ` ORDER BY json_extract(value, '%s')`, path)
		if sort.Desc {
			b.WriteString(` DESC`)
		}
	}

	if limit > 0 {
		fmt.Fprintf(&b, ` LIMIT %d`, limit+1)
	}

	rows, err := scanDuckDBJSONValues(ctx, e.db, b.String(), args...)
	if err != nil {
		return metaengine.ScanResult{}, err
	}

	hasMore := limit > 0 && len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}

	return metaengine.ScanResult{Items: rows, HasMore: hasMore}, nil
}

// jsonPath converts a field name to a DuckDB JSON path.
// E.g. "status" → "$.status". Single quotes are escaped.
func jsonPath(field string) string {
	escaped := strings.ReplaceAll(field, "'", "''")
	return "$." + escaped
}

// scanDuckDBJSONValues executes the query and decodes each row's JSON value.
func scanDuckDBJSONValues(
	ctx context.Context,
	db *sql.DB,
	query string,
	args ...any,
) ([]any, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("duckdbengine scan: %w", err)
	}

	defer func() { _ = rows.Close() }()

	var result []any

	for rows.Next() {
		var raw string

		if err := rows.Scan(&raw); err != nil {
			return nil, fmt.Errorf("duckdbengine scan: row: %w", err)
		}

		var val any
		if err := json.Unmarshal([]byte(raw), &val); err != nil {
			return nil, fmt.Errorf("duckdbengine scan: unmarshal: %w", err)
		}

		result = append(result, val)
	}

	if err := rows.Err(); err != nil {
		return result, fmt.Errorf("duckdbengine scan: %w", err)
	}

	return result, nil
}

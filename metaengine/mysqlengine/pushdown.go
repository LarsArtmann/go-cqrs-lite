package mysqlengine

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"strings"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// PushdownMapScan pushes WHERE/ORDER BY/LIMIT into MySQL using JSON
// operators (value->'$.field'), avoiding the full-table load that MapScan
// performs. MySQL 5.7+ supports the JSON path operator -> for extraction.
//
// Using value->'$.field' preserves the native JSON type for numeric
// comparisons. Keyset pagination adds a WHERE clause on the sort column.
// art-dupl:accept cross-module SQL engine pattern — separate go.mod
func (e *mysqlEngine) PushdownMapScan(
	ctx context.Context,
	collection string,
	filters []metaengine.FilterSpec,
	sort *metaengine.SortSpec,
	cursor any,
	limit int,
) (metaengine.ScanResult, error) {
	var b strings.Builder
	args := []any{collection}

	b.WriteString(`SELECT CAST(value AS CHAR) FROM meta_map WHERE collection = ?`)

	for _, f := range filters {
		if f.Op == metaengine.FilterIn {
			values, ok := f.Value.([]any)
			if !ok || len(values) == 0 {
				continue
			}

			placeholders := make([]string, len(values))
			for i, v := range values {
				jb, _ := json.Marshal(v)
				placeholders[i] = "CAST(? AS JSON)"
				args = append(args, string(jb))
			}

			fmt.Fprintf(&b, ` AND value->'$.%s' IN (%s)`,
				escapeJSONPath(f.Column), strings.Join(placeholders, ", "))
		} else {
			jb, _ := json.Marshal(f.Value)
			fmt.Fprintf(&b, ` AND value->'$.%s' %s CAST(? AS JSON)`,
				escapeJSONPath(f.Column), string(f.Op))
			args = append(args, string(jb))
		}
	}

	if sort != nil && cursor != nil {
		op := ">"
		if sort.Desc {
			op = "<"
		}

		jb, _ := json.Marshal(cursor)
		fmt.Fprintf(&b, ` AND value->'$.%s' %s CAST(? AS JSON)`,
			escapeJSONPath(sort.Column), op)
		args = append(args, string(jb))
	}

	if sort != nil {
		fmt.Fprintf(&b, ` ORDER BY value->'$.%s'`, escapeJSONPath(sort.Column))
		if sort.Desc {
			b.WriteString(` DESC`)
		}
	}

	if limit > 0 {
		fmt.Fprintf(&b, ` LIMIT %d`, limit+1)
	}

	rows, err := scanMySQLJSONValues(ctx, e.conn(), b.String(), args...)
	if err != nil {
		return metaengine.ScanResult{}, err
	}

	hasMore := limit > 0 && len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}

	return metaengine.ScanResult{Items: rows, HasMore: hasMore}, nil
}

// ApplyLayout implements metaengine.LayoutPlanner. It creates functional
// indexes on the meta_map table for the declared filter/sort fields.
// MySQL 8.0.13+ supports functional key parts.
// art-dupl:accept cross-module SQL engine pattern — separate go.mod
func (e *mysqlEngine) ApplyLayout(collection string, filterFields, sortFields []string) error {
	e.layoutMu.Lock()
	defer e.layoutMu.Unlock()

	if e.appliedLayouts == nil {
		e.appliedLayouts = make(map[string]bool)
	}

	key := collection
	if e.appliedLayouts[key] {
		return nil
	}

	seen := make(map[string]bool)
	allFields := append(append([]string{}, filterFields...), sortFields...)

	for _, field := range allFields {
		if seen[field] {
			continue
		}

		seen[field] = true

		idxName := sanitizeIndexName("idx_map", collection, field)
		escaped := escapeJSONPath(field)

		ddl := fmt.Sprintf(
			`CREATE INDEX %s ON meta_map ((CAST(value->'$.%s' AS CHAR(255))))`,
			idxName, escaped,
		)

		// MySQL doesn't support CREATE INDEX IF NOT EXISTS; ignore duplicate
		// index errors gracefully.
		if _, err := e.conn().ExecContext(context.Background(), ddl); err != nil {
			// Error code 1061 = Duplicate key name (index already exists).
			if !isDuplicateIndexErr(err) {
				return fmt.Errorf("mysqlengine.ApplyLayout: create index %s: %w", idxName, err)
			}
		}
	}

	e.appliedLayouts[key] = true

	return nil
}

// scanMySQLJSONValues executes the query and decodes each row's JSON value.
// art-dupl:accept cross-module SQL engine pattern — separate go.mod
func scanMySQLJSONValues(
	ctx context.Context,
	db metaengine.SQLExec,
	query string,
	args ...any,
) ([]any, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("mysqlengine scan: %w", err)
	}

	defer metaengine.DeferClose(rows)

	var result []any

	for rows.Next() {
		var raw []byte

		if err := rows.Scan(&raw); err != nil {
			return nil, fmt.Errorf("mysqlengine scan: row: %w", err)
		}

		var val any
		if err := json.Unmarshal(raw, &val); err != nil {
			return nil, fmt.Errorf("mysqlengine scan: unmarshal: %w", err)
		}

		result = append(result, val)
	}

	if err := rows.Err(); err != nil {
		return result, fmt.Errorf("mysqlengine scan: %w", err)
	}

	return result, nil
}

// escapeJSONPath escapes single quotes in a JSON path expression for use
// inside value->'$.field'. MySQL uses ” to escape a single quote.
func escapeJSONPath(key string) string {
	return strings.ReplaceAll(key, "'", "''")
}

// isDuplicateIndexErr checks if the error is a MySQL duplicate index error
// (error code 1061).
func isDuplicateIndexErr(err error) bool {
	return strings.Contains(err.Error(), "1061") || strings.Contains(err.Error(), "Duplicate key")
}

// sanitizeIndexName builds a safe SQL identifier from components.
func sanitizeIndexName(parts ...string) string {
	var b strings.Builder

	for i, p := range parts {
		if i > 0 {
			b.WriteByte('_')
		}

		for _, r := range p {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') ||
				r == '_' {
				b.WriteRune(r)
			} else {
				b.WriteByte('_')
			}
		}
	}

	return b.String()
}

package pgengine

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"fmt"
	"strings"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// PushdownMapScan pushes WHERE/ORDER BY/LIMIT into Postgres using JSONB
// operators (value->'field'), avoiding the full-table load that MapScan
// performs. Filters become value->'field' = $N::jsonb, sort becomes
// ORDER BY value->'field', and limit becomes LIMIT.
//
// Using value->'field' (JSONB) instead of value->>'field' (text) preserves
// the native JSON type, so numeric comparisons work correctly:
// 5 > 3 (numeric), not "5" > "3" (lexical).
//
// Keyset pagination: cursor becomes an additional WHERE clause on the sort
// column. LIMIT is n+1 for has-more detection.
func (e *pgEngine) PushdownMapScan(
	ctx context.Context,
	collection string,
	filters []metaengine.FilterSpec,
	sort *metaengine.SortSpec,
	cursor any,
	limit int,
) ([]any, error) {
	var b strings.Builder
	args := []any{collection}
	ph := 2 // $1 = collection

	b.WriteString(`SELECT value::text FROM meta_map WHERE collection = $1`)

	for _, f := range filters {
		if f.Op == metaengine.FilterIn {
			values, ok := f.Value.([]any)
			if !ok || len(values) == 0 {
				continue
			}

			placeholders := make([]string, len(values))
			for i, v := range values {
				jb, _ := json.Marshal(v)
				placeholders[i] = fmt.Sprintf("$%d::jsonb", ph)
				ph++
				args = append(args, string(jb))
			}

			fmt.Fprintf(&b, ` AND value->'%s' IN (%s)`,
				escapeJSONKey(f.Column), strings.Join(placeholders, ", "))
		} else {
			jb, _ := json.Marshal(f.Value)
			fmt.Fprintf(&b, ` AND value->'%s' %s $%d::jsonb`,
				escapeJSONKey(f.Column), string(f.Op), ph)
			ph++
			args = append(args, string(jb))
		}
	}

	if sort != nil && cursor != nil {
		op := ">"
		if sort.Desc {
			op = "<"
		}

		jb, _ := json.Marshal(cursor)
		fmt.Fprintf(&b, ` AND value->'%s' %s $%d::jsonb`,
			escapeJSONKey(sort.Column), op, ph)
		ph++
		args = append(args, string(jb))
	}

	if sort != nil {
		fmt.Fprintf(&b, ` ORDER BY value->'%s'`, escapeJSONKey(sort.Column))
		if sort.Desc {
			b.WriteString(` DESC`)
		}
	}

	if limit > 0 {
		fmt.Fprintf(&b, ` LIMIT %d`, limit+1)
	}

	return scanPGJSONValues(ctx, e.db, b.String(), args...)
}

// ApplyLayout implements metaengine.LayoutPlanner. It creates partial
// expression indexes on the meta_map table for the declared filter/sort
// fields, scoped to the specific collection. Postgres uses these indexes
// automatically when PushdownMapScan queries reference the same JSONB paths.
//
// Example: ApplyLayout("users", ["status"], ["priority"]) creates:
//
//	CREATE INDEX IF NOT EXISTS idx_map_users_status
//	ON meta_map ((value->'status')) WHERE collection = 'users'
//	CREATE INDEX IF NOT EXISTS idx_map_users_priority
//	ON meta_map ((value->'priority')) WHERE collection = 'users'
func (e *pgEngine) ApplyLayout(collection string, filterFields, sortFields []string) error {
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
		escaped := escapeJSONKey(field)

		ddl := fmt.Sprintf(
			`CREATE INDEX IF NOT EXISTS %s ON meta_map ((value->'%s')) WHERE collection = '%s'`,
			idxName, escaped, escapeSQLString(collection),
		)

		if _, err := e.db.ExecContext(context.Background(), ddl); err != nil {
			return fmt.Errorf("pgengine.ApplyLayout: create index %s: %w", idxName, err)
		}
	}

	e.appliedLayouts[key] = true

	return nil
}

// scanPGJSONValues executes the query and decodes each row's JSONB value.
func scanPGJSONValues(ctx context.Context, db *sql.DB, query string, args ...any) ([]any, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("pgengine scan: %w", err)
	}

	defer func() { _ = rows.Close() }()

	var result []any

	for rows.Next() {
		var raw []byte

		if err := rows.Scan(&raw); err != nil {
			return nil, fmt.Errorf("pgengine scan: row: %w", err)
		}

		var val any
		if err := json.Unmarshal(raw, &val); err != nil {
			return nil, fmt.Errorf("pgengine scan: unmarshal: %w", err)
		}

		result = append(result, val)
	}

	if err := rows.Err(); err != nil {
		return result, fmt.Errorf("pgengine scan: %w", err)
	}

	return result, nil
}

// escapeJSONKey escapes single quotes in a JSONB key literal for use inside
// value->'key'. Postgres uses '' to escape a single quote inside a string.
func escapeJSONKey(key string) string {
	return strings.ReplaceAll(key, "'", "''")
}

// escapeSQLString escapes single quotes in a SQL string literal.
func escapeSQLString(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// sanitizeIndexName builds a safe SQL identifier from components.
// Non-alphanumeric characters are replaced with underscores.
func sanitizeIndexName(parts ...string) string {
	var b strings.Builder
	for i, p := range parts {
		if i > 0 {
			b.WriteByte('_')
		}

		for _, r := range p {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
				b.WriteRune(r)
			} else {
				b.WriteByte('_')
			}
		}
	}

	return b.String()
}

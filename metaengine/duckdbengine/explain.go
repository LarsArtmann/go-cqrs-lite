//go:build cgo

package duckdbengine

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"strings"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// ExplainScanQuery implements metaengine.ExplainableScan. It returns the SQL
// that PushdownMapScan would execute for the given collection and options,
// without running the query.
//
// For planned tables (collections with an applied LayoutPlan), the SQL uses
// direct column references. For standard tables, it uses json_extract with
// $N::json placeholders — the same SQL that PushdownMapScan builds internally.
func (e *duckdbEngine) ExplainScanQuery(
	_ context.Context,
	collection string,
	opts metaengine.ExplainOptions,
) (string, []any) {
	if plan, ok := e.plans[collection]; ok {
		return buildPlannedSelectQuery(plan, opts.Filters, opts.Sort, opts.Cursor, opts.Limit)
	}

	return explainStandardDuckDB(collection, opts.Filters, opts.Sort, opts.Cursor, opts.Limit)
}

// explainStandardDuckDB builds the SQL for a standard (non-planned) scan on
// meta_map using json_extract, mirroring the logic in PushdownMapScan.
func explainStandardDuckDB(
	collection string,
	filters []metaengine.FilterSpec,
	sort *metaengine.SortSpec,
	cursor any,
	limit int,
) (string, []any) {
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

	return b.String(), args
}

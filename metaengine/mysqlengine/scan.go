package mysqlengine

import (
	"context"
	"encoding/json/v2"
	"fmt"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// MapScan implements ScanBackend for MySQL. It SELECTs all rows for the
// collection, decodes JSON values, then applies filter/sort/limit in Go.
// For declarative filter/sort via FilterOnField/SortOnField, the executor
// prefers PushdownMapScan (see pushdown.go) which pushes these into SQL.
// art-dupl:accept cross-module SQL engine pattern — separate go.mod
func (e *mysqlEngine) MapScan(
	ctx context.Context,
	collection string,
	filterFn func(item any) bool,
	sortFunc func(a, b any) int,
	cursor any,
	limit int,
) (metaengine.ScanResult, error) {
	// Planned collections store rows in a dedicated table; MapScan (the
	// closure-based fallback) must read from it, not meta_map (D3 slice 2 —
	// closes the planned/meta_map visibility split).
	query := `SELECT ` + keyCol + `, CAST(value AS CHAR) FROM meta_map WHERE collection = ?`

	var args []any

	if plan, ok := e.planFor(collection); ok {
		query = fmt.Sprintf("SELECT %s, CAST(value AS CHAR) FROM %s",
			keyCol, backtickIdent(plan.Table))
	} else {
		args = append(args, collection)
	}

	//art-dupl:accept cross-module SQL engine pattern — dep-isolated go.mod modules
	rows, err := e.conn(ctx).QueryContext(ctx, query, args...)
	if err != nil {
		return metaengine.ScanResult{}, fmt.Errorf("mysqlengine.MapScan: %w", err)
	}

	defer metaengine.DeferClose(rows)

	type kv struct {
		key   string
		value any
	}

	var pairs []kv

	for rows.Next() {
		var key string

		var raw []byte

		if err := rows.Scan(&key, &raw); err != nil {
			return metaengine.ScanResult{}, fmt.Errorf("mysqlengine.MapScan: scan: %w", err)
		}

		var val any
		if err := json.Unmarshal(raw, &val); err != nil {
			return metaengine.ScanResult{}, fmt.Errorf("mysqlengine.MapScan: unmarshal: %w", err)
		}

		if filterFn != nil && !filterFn(val) {
			continue
		}

		pairs = append(pairs, kv{key: key, value: val})
	}

	if err := rows.Err(); err != nil {
		return metaengine.ScanResult{}, fmt.Errorf("mysqlengine.MapScan: %w", err)
	}

	return metaengine.PairsToScanResult(
		metaengine.SortPaginate(
			pairs,
			func(p kv) []byte { return []byte(p.key) },
			func(p kv) any { return p.value },
			sortFunc,
			cursor,
			limit,
		),
		func(p kv) any { return p.value },
		limit,
	), nil
}

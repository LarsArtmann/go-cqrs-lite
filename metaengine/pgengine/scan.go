package pgengine

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"sort"
	"strings"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// MapScan implements ScanBackend for Postgres. It SELECTs all rows for the
// collection, decodes JSONB values, then applies filter/sort/limit in Go.
// For declarative filter/sort via FilterOnField/SortOnField, the executor
// prefers PushdownMapScan (see pushdown.go) which pushes these into SQL.
func (e *pgEngine) MapScan(
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
	query := `SELECT key, value::text FROM meta_map WHERE collection = $1`

	var args []any

	if plan, ok := e.planFor(collection); ok {
		query = "SELECT key, value::text FROM " + metaengine.QuoteIdent(plan.Table)
	} else {
		args = append(args, collection)
	}

	//art-dupl:accept cross-module SQL engine pattern — dep-isolated go.mod modules
	rows, err := e.conn().QueryContext(ctx, query, args...)
	if err != nil {
		return metaengine.ScanResult{}, fmt.Errorf("pgengine.MapScan: %w", err)
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
			return metaengine.ScanResult{}, fmt.Errorf("pgengine.MapScan: scan: %w", err)
		}

		var val any
		if err := json.Unmarshal(raw, &val); err != nil {
			return metaengine.ScanResult{}, fmt.Errorf("pgengine.MapScan: unmarshal: %w", err)
		}

		if filterFn != nil && !filterFn(val) {
			continue
		}

		pairs = append(pairs, kv{key: key, value: val})
	}

	if err := rows.Err(); err != nil {
		return metaengine.ScanResult{}, fmt.Errorf("pgengine.MapScan: %w", err)
	}

	if sortFunc != nil {
		sort.Slice(pairs, func(i, j int) bool {
			if c := sortFunc(pairs[i].value, pairs[j].value); c != 0 {
				return c < 0
			}

			return strings.Compare(pairs[i].key, pairs[j].key) < 0
		})
	}

	if cursor != nil && sortFunc != nil {
		filtered := pairs[:0]
		for _, p := range pairs {
			if sortFunc(p.value, cursor) <= 0 {
				continue
			}

			filtered = append(filtered, p)
		}

		pairs = filtered
	}

	hasMore := limit > 0 && len(pairs) > limit
	if hasMore {
		pairs = pairs[:limit]
	}

	result := make([]any, len(pairs))
	for i, p := range pairs {
		result[i] = p.value
	}

	return metaengine.ScanResult{Items: result, HasMore: hasMore}, nil
}

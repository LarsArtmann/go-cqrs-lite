package duckdbengine

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"sort"
	"strings"
)

// MapScan implements ScanBackend for DuckDB. It SELECTs all rows for the
// collection, decodes JSON values, then applies filter/sort/limit in Go.
// For declarative filter/sort via FilterOnField/SortOnField, the executor
// prefers PushdownMapScan (see pushdown.go) which pushes these into SQL.
func (e *duckdbEngine) MapScan(
	ctx context.Context,
	collection string,
	filterFn func(item any) bool,
	sortFunc func(a, b any) int,
	cursor any,
	limit int,
) (metaengine.ScanResult, error) {
	rows, err := e.db.QueryContext(
		ctx,
		`SELECT key, value FROM meta_map WHERE collection = $1`,
		collection,
	)
	if err != nil {
		return metaengine.ScanResult{}, fmt.Errorf("duckdbengine.MapScan: %w", err)
	}

	defer func() { _ = rows.Close() }()

	type kv struct {
		key   string
		value any
	}

	var pairs []kv

	for rows.Next() {
		var key string

		var raw string

		if err := rows.Scan(&key, &raw); err != nil {
			return nil, fmt.Errorf("duckdbengine.MapScan: scan: %w", err)
		}

		var val any
		if err := json.Unmarshal([]byte(raw), &val); err != nil {
			return nil, fmt.Errorf("duckdbengine.MapScan: unmarshal: %w", err)
		}

		if filterFn != nil && !filterFn(val) {
			continue
		}

		pairs = append(pairs, kv{key: key, value: val})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("duckdbengine.MapScan: %w", err)
	}

	// Sort with deterministic tiebreaker (same pattern as Memory engine).
	if sortFunc != nil {
		sort.Slice(pairs, func(i, j int) bool {
			if c := sortFunc(pairs[i].value, pairs[j].value); c != 0 {
				return c < 0
			}

			return strings.Compare(pairs[i].key, pairs[j].key) < 0
		})
	}

	// Keyset pagination.
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

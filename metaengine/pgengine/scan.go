package pgengine

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"sort"
	"strings"
)

// MapScan implements ScanBackend for Postgres. It SELECTs all rows for the
// collection, decodes JSONB values, then applies filter/sort/limit in Go.
// Future enhancement: push filter to Postgres WHERE clause via jsonb operators.
func (e *pgEngine) MapScan(
	ctx context.Context,
	collection string,
	filterFn func(item any) bool,
	sortFunc func(a, b any) int,
	cursor any,
	limit int,
) ([]any, error) {
	rows, err := e.db.QueryContext(
		ctx,
		`SELECT key, value::text FROM meta_map WHERE collection = $1`,
		collection,
	)
	if err != nil {
		return nil, fmt.Errorf("pgengine.MapScan: %w", err)
	}

	defer func() { _ = rows.Close() }()

	type kv struct {
		key   string
		value any
	}

	var pairs []kv

	for rows.Next() {
		var key string

		var raw []byte

		if err := rows.Scan(&key, &raw); err != nil {
			return nil, fmt.Errorf("pgengine.MapScan: scan: %w", err)
		}

		var val any
		if err := json.Unmarshal(raw, &val); err != nil {
			return nil, fmt.Errorf("pgengine.MapScan: unmarshal: %w", err)
		}

		if filterFn != nil && !filterFn(val) {
			continue
		}

		pairs = append(pairs, kv{key: key, value: val})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("pgengine.MapScan: %w", err)
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

	if limit > 0 && limit < len(pairs) {
		pairs = pairs[:limit]
	}

	result := make([]any, len(pairs))
	for i, p := range pairs {
		result[i] = p.value
	}

	return result, nil
}

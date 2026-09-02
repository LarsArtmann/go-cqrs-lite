package dgraphengine

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"sort"
	"strings"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// MapScan implements ScanBackend for Dgraph. It queries all map entries for
// the collection, decodes JSON values, then applies filter/sort/limit in Go.
func (e *dgraphEngine) MapScan(
	ctx context.Context,
	collection string,
	filterFn func(item any) bool,
	sortFunc func(a, b any) int,
	cursor any,
	limit int,
) (metaengine.ScanResult, error) {
	q := `query entry($col: string) {
		entry(func: eq(cqrs.map_collection, $col)) {
			cqrs.map_key
			cqrs.map_value
		}
	}`

	resp, err := e.readTx().
		QueryWithVars(ctx, q, map[string]string{"$col": collection})
	if err != nil {
		return metaengine.ScanResult{}, fmt.Errorf("dgraphengine.MapScan: %w", err)
	}

	var result struct {
		Entry []struct {
			MapKey   string `json:"cqrs.map_key"`
			MapValue string `json:"cqrs.map_value"`
		} `json:"entry"`
	}

	if err := json.Unmarshal(resp.GetJson(), &result); err != nil {
		return metaengine.ScanResult{}, fmt.Errorf("dgraphengine.MapScan: unmarshal: %w", err)
	}

	type kv struct {
		key   string
		value any
	}

	var pairs []kv

	for _, entry := range result.Entry {
		var val any

		if err := json.Unmarshal([]byte(entry.MapValue), &val); err != nil {
			return metaengine.ScanResult{}, fmt.Errorf("dgraphengine.MapScan: decode: %w", err)
		}

		if filterFn != nil && !filterFn(val) {
			continue
		}

		pairs = append(pairs, kv{key: entry.MapKey, value: val})
	}

	if sortFunc != nil {
		sort.Slice(pairs, func(i, j int) bool {
			if c := sortFunc(pairs[i].value, pairs[j].value); c != 0 {
				return c < 0
			}

			return strings.Compare(pairs[i].key, pairs[j].key) < 0
		})
	}

	//art-dupl:accept cross-module engine pattern — dep-isolated go.mod modules
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

	items := make([]any, len(pairs))
	for i, p := range pairs {
		items[i] = p.value
	}

	return metaengine.ScanResult{Items: items, HasMore: hasMore}, nil
}

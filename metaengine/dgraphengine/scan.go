package dgraphengine

import (
	"context"
	"encoding/json/v2"
	"fmt"

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

package pebbleengine

import (
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// kvPair pairs a Pebble key with its decoded value for sorting. The raw field
// holds the original JSON bytes (used by ScanRawValues to return raw bytes
// without re-encoding; nil for MapScan which returns decoded values directly).
type kvPair struct {
	key   []byte
	value any    // decoded value for filter/sort/cursor comparisons
	raw   []byte // original JSON bytes (ScanRawValues return path; nil for MapScan)
}

// kvPairKey/kvPairValue are the accessors handed to metaengine.SortPaginate.
func kvPairKey(p kvPair) []byte { return p.key }

func kvPairValue(p kvPair) any { return p.value }

// sortAndPaginate delegates to the shared metaengine.SortPaginate core; the
// algorithm (sort by value, byte-key tiebreak, keyset cursor, limit+1) lives
// in one place for all KV engines.
func sortAndPaginate(pairs []kvPair, sortFn func(a, b any) int, cursor any, limit int) []kvPair {
	return metaengine.SortPaginate(pairs, kvPairKey, kvPairValue, sortFn, cursor, limit)
}

func trimRaw(results [][]byte, limit int) metaengine.RawScanResult {
	hasMore := limit > 0 && len(results) > limit
	if hasMore {
		results = results[:limit]
	}

	return metaengine.RawScanResult{Items: results, HasMore: hasMore}
}

// extractOrDirect returns the named column from a map item, or the value itself
// if it is not a map. This lets a single comparator function handle both item
// comparisons (where both args are decoded JSON maps) and cursor comparisons
// (where the cursor is a bare field value like float64(2)).
func extractOrDirect(v any, col string) any {
	if m, ok := v.(map[string]any); ok {
		return m[col]
	}

	return v
}

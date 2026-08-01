package pebbleengine

import (
	"bytes"
	"sort"

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

// sortAndPaginate sorts pairs by value (with byte-key tiebreak for determinism),
// applies keyset pagination (skipping items at or before cursor), and truncates
// to limit+1 (the +1 lets callers detect has-more).
//
// sortFn is a tri-state comparator (negative = a before b). When nil, no
// sorting or cursor pagination is applied — only the limit truncation runs.
// cursor is the keyset pagination cursor: items where sortFn(item, cursor) <= 0
// are skipped (already seen). The caller must ensure sortFn handles the cursor
// value correctly (it may be a raw field value rather than a full item — see
// extractOrDirect).
func sortAndPaginate(pairs []kvPair, sortFn func(a, b any) int, cursor any, limit int) []kvPair {
	if sortFn != nil {
		sort.Slice(pairs, func(i, j int) bool {
			if c := sortFn(pairs[i].value, pairs[j].value); c != 0 {
				return c < 0
			}

			return bytes.Compare(pairs[i].key, pairs[j].key) < 0
		})
	}

	if cursor != nil && sortFn != nil {
		filtered := pairs[:0]

		for _, p := range pairs {
			if sortFn(p.value, cursor) <= 0 {
				continue
			}

			filtered = append(filtered, p)
		}

		pairs = filtered
	}

	truncLimit := 0
	if limit > 0 {
		truncLimit = limit + 1
	}

	if truncLimit > 0 && len(pairs) > truncLimit {
		pairs = pairs[:truncLimit]
	}

	return pairs
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

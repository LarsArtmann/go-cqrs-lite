package metaengine

import (
	"bytes"
	"sort"
)

// SortPaginate sorts pairs by value (with byte-key tiebreak for determinism),
// applies keyset pagination (skipping items at or before cursor), and truncates
// to limit+1 (the +1 lets callers detect has-more). It is the shared core of
// the KV engines' in-memory scan paths (badger, pebble, bbolt); each engine
// maps its own pair type in via keyOf/valueOf, so the extraction removes the
// duplicated algorithm without forcing a common pair struct.
//
// sortFn is a tri-state comparator (negative = a before b). When nil, no
// sorting or cursor pagination is applied — only the limit truncation runs.
// cursor is the keyset pagination cursor: items where sortFn(item, cursor) <= 0
// are skipped (already seen). The caller must ensure sortFn handles the cursor
// value correctly (it may be a raw field value rather than a full item).
// The slice is sorted/filtered in place and returned for convenience.
func SortPaginate[T any](
	pairs []T,
	keyOf func(T) []byte,
	valueOf func(T) any,
	sortFn func(a, b any) int,
	cursor any,
	limit int,
) []T {
	if sortFn != nil {
		sort.Slice(pairs, func(i, j int) bool {
			if c := sortFn(valueOf(pairs[i]), valueOf(pairs[j])); c != 0 {
				return c < 0
			}

			return bytes.Compare(keyOf(pairs[i]), keyOf(pairs[j])) < 0
		})
	}

	if cursor != nil && sortFn != nil {
		filtered := pairs[:0]

		for _, p := range pairs {
			if sortFn(valueOf(p), cursor) <= 0 {
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

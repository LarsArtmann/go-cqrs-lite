package metaengine

import (
	"bytes"
	"sort"
)

// SortKeyCursor is a compound keyset cursor: the sort-field value plus the
// byte key of the item that produced it. A value-only cursor cannot
// distinguish items that TIE the cursor's sort value — skipping the tie
// block silently drops unseen rows, keeping it re-serves seen ones. With
// both components the cursor filter applies the same (sortValue, key)
// ordering the sort itself uses, so tie-heavy datasets paginate with
// neither drops nor duplicates.
type SortKeyCursor struct {
	Sort any
	Key  []byte
}

// SortPaginate sorts pairs by value (with byte-key tiebreak for determinism),
// applies keyset pagination (skipping items at or before cursor), and truncates
// to limit+1 (the +1 lets callers detect has-more). It is the shared core of
// the KV engines' in-memory scan paths (badger, pebble, bbolt); each engine
// maps its own pair type in via keyOf/valueOf, so the extraction removes the
// duplicated algorithm without forcing a common pair struct.
//
// sortFn is a tri-state comparator (negative = a before b). When nil, no
// sorting or cursor pagination is applied — only the limit truncation runs.
// cursor is the keyset pagination cursor: a [SortKeyCursor] is compared with
// the full (sortValue, key) ordering, so ties paginate exactly once. A raw
// value cursor keeps the legacy behavior (items where sortFn(item, cursor)
// <= 0 are skipped) — tie-lossy, retained for compatibility.
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
		cursorValue := cursor

		var compound SortKeyCursor

		hasCompound := false

		if c, ok := cursor.(SortKeyCursor); ok {
			cursorValue = c.Sort
			compound = c
			hasCompound = true
		}

		filtered := pairs[:0]

		for _, p := range pairs {
			c := sortFn(valueOf(p), cursorValue)

			if c < 0 {
				continue
			}

			if c == 0 {
				if hasCompound {
					if bytes.Compare(keyOf(p), compound.Key) <= 0 {
						continue
					}
				} else {
					continue
				}
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

// PairsToScanResult truncates pairs to limit (reporting hasMore when more data
// existed beyond it) and collects each pair's value into the ScanResult items.
// It is the tail half of the SortPaginate contract: SortPaginate returns up to
// limit+1 pairs precisely so this helper can detect has-more. Each KV engine
// maps its own pair type in via valueOf.
func PairsToScanResult[T any](pairs []T, valueOf func(T) any, limit int) ScanResult {
	hasMore := limit > 0 && len(pairs) > limit
	if hasMore {
		pairs = pairs[:limit]
	}

	results := make([]any, len(pairs))
	for i, p := range pairs {
		results[i] = valueOf(p)
	}

	return ScanResult{Items: results, HasMore: hasMore}
}

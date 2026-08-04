package pebbleengine

import (
	"context"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// ScanCounter is an optional capability for counting scan results without
// materializing all values. Engines that implement this can answer "how many
// rows match this filter?" in O(N) time with O(1) memory (no JSON decode for
// the unfiltered case).
type ScanCounter interface {
	ScanCount(
		ctx context.Context,
		collection string,
		filters []metaengine.FilterSpec,
	) (int64, error)
}

// ScanCount returns the number of items in a collection that match the given
// filters. When no filters are provided, it counts keys only (no JSON decode).
// When filters are provided, each value is decoded to evaluate the filter.
func (e *pebbleEngine) ScanCount(
	_ context.Context,
	col string,
	filters []metaengine.FilterSpec,
) (int64, error) {
	iter, err := e.newPrefixIter(collectionPrefix(col))
	if err != nil {
		return 0, err
	}

	defer func() { _ = iter.Close() }()

	var count int64

	// Fast path: no filters → count keys without decoding values.
	if len(filters) == 0 {
		for iter.First(); iter.Valid(); iter.Next() {
			count++
		}

		return count, iter.Error()
	}

	// Filtered path: decode each value to evaluate the filter.
	for iter.First(); iter.Valid(); iter.Next() {
		val := decodeJSON(iter.Value())

		if metaengine.PassesFilterSpecs(val, filters) {
			count++
		}
	}

	return count, iter.Error()
}

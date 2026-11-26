package pebbleengine

import (
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/keycodec"
)

// filterIndexPairs turns filter-index hits into the pair form the shared
// sort-and-paginate core consumes.
func filterIndexPairs(hits []sortIndexHit) []kvPair {
	if len(hits) == 0 {
		return nil
	}

	pairs := make([]kvPair, len(hits))

	for i, hit := range hits {
		pairs[i] = kvPair{
			key:   keycodec.UserKeyBytes([]byte(hit.primaryKey), nil),
			value: decodeJSON(hit.value),
			raw:   hit.value,
		}
	}

	return pairs
}

// rawPairsResult sorts, paginates and limits a candidate set that already
// passed every filter. Both the unindexed scan and the filter-index scan end
// here, so they share one cursor contract: a compound (sort value, key)
// cursor that the caller can round-trip even when the sort column is not
// the indexed column.
func rawPairsResult(
	pairs []kvPair,
	sortSpec *metaengine.SortSpec,
	cursor any,
	limit int,
) metaengine.RawScanResult {
	var sortFn func(a, b any) int

	if sortSpec != nil {
		sortFn = func(a, b any) int {
			c := metaengine.CompareValues(
				extractOrDirect(a, sortSpec.Column),
				extractOrDirect(b, sortSpec.Column),
			)

			if sortSpec.Desc {
				return -c
			}

			return c
		}
	}

	pairs = sortAndPaginate(pairs, sortFn, cursor, limit)

	hasMore := limit > 0 && len(pairs) > limit
	if hasMore {
		pairs = pairs[:limit]
	}

	results := make([][]byte, len(pairs))

	for i, p := range pairs {
		results[i] = p.raw
	}

	var next any

	if sortSpec != nil && len(pairs) > 0 {
		last := pairs[len(pairs)-1]
		next = metaengine.LastDecodedRowCursor(last.value, sortSpec.Column, last.key)
	}

	return metaengine.RawScanResult{Items: results, HasMore: hasMore, NextCursor: next}
}

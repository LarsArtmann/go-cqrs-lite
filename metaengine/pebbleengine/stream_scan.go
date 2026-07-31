package pebbleengine

import (
	"context"
	"iter"
	"slices"

	"github.com/cockroachdb/pebble"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// StreamScan returns a lazy iterator over collection rows, applying filter
// predicates in Go. This implements the metaengine.StreamingScan interface
// for OOM-safe iteration of large collections.
//
// WITHOUT sort: true lazy iteration (O(1) memory per row).
// WITH sort: materializes all matching rows, sorts them, then streams the
// sorted result. This is an O(N) memory tradeoff — the caller still gets
// one-at-a-time delivery, but the engine must buffer internally.
func (e *pebbleEngine) StreamScan(
	_ context.Context,
	col string,
	filters []metaengine.FilterSpec,
	sort *metaengine.SortSpec,
) iter.Seq2[any, error] {
	return func(yield func(any, error) bool) {
		prefix := collectionPrefix(col)
		upperBound := nextKey(prefix)

		dbIter, err := e.db.NewIter(&pebble.IterOptions{
			LowerBound: prefix,
			UpperBound: upperBound,
		})
		if err != nil {
			yield(nil, err)

			return
		}

		defer func() { _ = dbIter.Close() }()

		// Unsorted path: stream directly, no materialization.
		if sort == nil {
			for dbIter.First(); dbIter.Valid(); dbIter.Next() {
				val := decodeJSON(dbIter.Value())

				if len(filters) > 0 && !metaengine.PassesFilterSpecs(val, filters) {
					continue
				}

				if !yield(val, nil) {
					return
				}
			}

			if err := dbIter.Error(); err != nil {
				yield(nil, err)
			}

			return
		}

		// Sorted path: collect, sort, then stream.
		var matched []any

		for dbIter.First(); dbIter.Valid(); dbIter.Next() {
			val := decodeJSON(dbIter.Value())

			if len(filters) > 0 && !metaengine.PassesFilterSpecs(val, filters) {
				continue
			}

			matched = append(matched, val)
		}

		if err := dbIter.Error(); err != nil {
			yield(nil, err)

			return
		}

		slices.SortFunc(matched, func(a, b any) int {
			av := metaengine.ItemFieldByName(a, sort.Column)
			bv := metaengine.ItemFieldByName(b, sort.Column)

			cmp := metaengine.CompareValues(av, bv)
			if sort.Desc {
				return -cmp
			}

			return cmp
		})

		for _, val := range matched {
			if !yield(val, nil) {
				return
			}
		}
	}
}

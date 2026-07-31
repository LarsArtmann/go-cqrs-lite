package pebbleengine

import (
	"context"
	"slices"

	"github.com/cockroachdb/pebble"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// hasSortField returns true if field is declared as a sort field in this plan.
func (p layoutPlan) hasSortField(field string) bool {
	return slices.Contains(p.sortFields, field)
}

// sortIndexFieldPrefix builds the sort index key prefix for a field (all values).
// Format: "o{sep}{col}{sep}{field}{sep}".
func sortIndexFieldPrefix(col, field string) []byte {
	return []byte("o" + sep + col + sep + field + sep)
}

// sortIndexKey builds the full sort index key for a field value + primary key.
// Format: "o{sep}{col}{sep}{field}{sep}{encodedValue}{sep}{primaryKey}".
func sortIndexKey(col, field, encodedValue, primaryKey string) []byte {
	return []byte("o" + sep + col + sep + field + sep + encodedValue + sep + primaryKey)
}

// scanWithSortIndex uses the sort index for ordered iteration. Keys in the
// sort index are laid out as o{sep}{col}{sep}{field}{sep}{encodedValue}{sep}{pk},
// so lexicographic forward iteration yields ascending order and backward
// iteration yields descending order — no Go-level sort needed. Filters are
// applied in Go; early termination stops once limit+1 matching rows are found.
// Cursor pagination uses the encoded cursor value to set an exclusive bound.
func (e *pebbleEngine) scanWithSortIndex(
	_ context.Context,
	col string,
	filters []metaengine.FilterSpec,
	sortSpec *metaengine.SortSpec,
	cursor any,
	limit int,
) ([][]byte, error) {
	prefix := sortIndexFieldPrefix(col, sortSpec.Column)
	lowerBound := prefix
	upperBound := nextKey(prefix)

	if cursor != nil {
		cursorGroup := append(append(prefix, encodeIndexValue(cursor)...), sep...)

		if sortSpec.Desc {
			upperBound = cursorGroup
		} else {
			lowerBound = nextKey(cursorGroup)
		}
	}

	iter, err := e.db.NewIter(&pebble.IterOptions{
		LowerBound: lowerBound,
		UpperBound: upperBound,
	})
	if err != nil {
		return nil, err
	}

	defer func() { _ = iter.Close() }()

	targetCount := 0
	if limit > 0 {
		targetCount = limit + 1
	}

	var results [][]byte

	if sortSpec.Desc {
		for iter.Last(); iter.Valid(); iter.Prev() {
			if e.collectSortIndexEntry(iter, col, filters, &results) &&
				targetCount > 0 && len(results) >= targetCount {
				break
			}
		}
	} else {
		for iter.First(); iter.Valid(); iter.Next() {
			if e.collectSortIndexEntry(iter, col, filters, &results) &&
				targetCount > 0 && len(results) >= targetCount {
				break
			}
		}
	}

	if err := iter.Error(); err != nil {
		return nil, err
	}

	return results, nil
}

// collectSortIndexEntry reads the value for the current iterator position,
// applies filters in Go, and appends to results when the row passes.
// Returns true if the row was appended.
func (e *pebbleEngine) collectSortIndexEntry(
	iter *pebble.Iterator,
	col string,
	filters []metaengine.FilterSpec,
	results *[][]byte,
) bool {
	fullKey := append([]byte(nil), iter.Key()...)
	primaryKey := extractPrimaryKeyFromIndex(fullKey)

	val, closer, err := e.db.Get(mapKey(col, primaryKey))
	if err != nil {
		return false
	}

	valCopy := append([]byte(nil), val...)
	_ = closer.Close()

	if len(filters) > 0 {
		decoded := decodeJSON(valCopy)

		if !metaengine.PassesFilterSpecs(decoded, filters) {
			return false
		}
	}

	*results = append(*results, valCopy)

	return true
}

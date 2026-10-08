package pebbleengine

import (
	"bytes"
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

// applyIndexEntries walks a value's layout-plan fields and hands every
// produced secondary-index key to fn: filter-field keys first (prefix + key
// suffix), then sort-field keys. kind carries the error-message fragment
// ("index entry" / "sort index entry") so callers keep their exact strings.
func applyIndexEntries(
	fields map[string]any,
	plan layoutPlan,
	col, key string,
	fn func(kind string, idxKey []byte) error,
) error {
	for _, field := range plan.filterFields {
		valStr, ok := indexFieldValue(fields, field)
		if !ok {
			continue
		}

		idxKey := append(layoutKeyPrefix(col, field, valStr), []byte(key)...)

		if err := fn("index entry", idxKey); err != nil {
			return err
		}
	}

	for _, field := range plan.sortFields {
		valStr, ok := indexFieldValue(fields, field)
		if !ok {
			continue
		}

		idxKey := sortIndexKey(col, field, valStr, key)

		if err := fn("sort index entry", idxKey); err != nil {
			return err
		}
	}

	return nil
}

// indexFieldValue encodes a field's value for a secondary-index key; ok is
// false when the event carries no value for the field.
func indexFieldValue(fields map[string]any, field string) (string, bool) {
	fieldVal, ok := fields[field]
	if !ok {
		return "", false
	}

	return encodeIndexValue(fieldVal), true
}

// sortIndexHit is one row produced by sort-index iteration: the row's raw
// JSON bytes plus the bare primary key, so callers can emit compound
// (sort value, key) cursors.
type sortIndexHit struct {
	primaryKey string
	value      []byte
}

// scanWithSortIndex uses the sort index for ordered iteration. Keys in the
// sort index are laid out as o{sep}{col}{sep}{field}{sep}{encodedValue}{sep}{pk},
// so lexicographic forward iteration yields (value ASC, key ASC). Filters are
// applied in Go; early termination stops once limit+1 matching rows are found.
//
// Cursor pagination: a metaengine.SortKeyCursor sets a composite bound on the
// full (encodedValue, primaryKey) index entry — ascending resumes strictly
// after the cursor entry, descending ranges up to (excluding) it so later keys
// of the same value group stay in play. A raw scalar cursor keeps the legacy
// whole-group-skip bound. Backward iteration alone would order equal-value
// ties key-DESCENDING, violating the wire contract's key-ascending tiebreak,
// so DESC walks buffer each equal-value run and flush it reversed; the buffer
// is bounded by targetCount because dropped-oldest entries would sort beyond
// the page anyway.
func (e *pebbleEngine) scanWithSortIndex(
	_ context.Context,
	col string,
	filters []metaengine.FilterSpec,
	sortSpec *metaengine.SortSpec,
	cursor any,
	limit int,
) ([]sortIndexHit, error) {
	prefix := sortIndexFieldPrefix(col, sortSpec.Column)
	lowerBound := prefix
	upperBound := nextKey(prefix)

	if cursor != nil {
		if compound, ok := cursor.(metaengine.SortKeyCursor); ok {
			// cursor.Key is the bare user key (wire form); index entries carry
			// the JSON-encoded key form, so re-encode before building the entry.
			entry := sortIndexKey(
				col, sortSpec.Column,
				encodeIndexValue(compound.Sort), encodeKeyStr(string(compound.Key)),
			)

			if sortSpec.Desc {
				upperBound = entry
			} else {
				lowerBound = nextKey(entry)
			}
		} else {
			cursorGroup := append(append(prefix, encodeIndexValue(cursor)...), sep...)

			if sortSpec.Desc {
				upperBound = cursorGroup
			} else {
				lowerBound = nextKey(cursorGroup)
			}
		}
	}

	iter, err := e.db.NewIter(&pebble.IterOptions{
		LowerBound: lowerBound,
		UpperBound: upperBound,
	})
	if err != nil {
		return nil, err
	}

	defer metaengine.DeferClose(iter)

	targetCount := 0
	if limit > 0 {
		targetCount = limit + 1
	}

	var results []sortIndexHit

	if sortSpec.Desc {
		var run []sortIndexHit

		var runGroup []byte

		flushRun := func() {
			for i := len(run) - 1; i >= 0; i-- {
				results = append(results, run[i])
			}

			run = run[:0]
		}

		for iter.Last(); iter.Valid(); iter.Prev() {
			fullKey := append([]byte(nil), iter.Key()...)
			group := fullKey[:len(fullKey)-len(extractPrimaryKeyFromIndex(fullKey))]

			if runGroup == nil || !bytes.Equal(group, runGroup) {
				flushRun()
				runGroup = append(runGroup[:0], group...)
			}

			if hit, ok := e.collectSortIndexHit(fullKey, col, filters); ok {
				run = append(run, hit)

				if targetCount > 0 && len(run) > targetCount {
					run = run[1:]
				}
			}

			if targetCount > 0 && len(results) >= targetCount {
				break
			}
		}

		flushRun()
	} else {
		for iter.First(); iter.Valid(); iter.Next() {
			fullKey := append([]byte(nil), iter.Key()...)

			if hit, ok := e.collectSortIndexHit(fullKey, col, filters); ok {
				results = append(results, hit)
			}

			if targetCount > 0 && len(results) >= targetCount {
				break
			}
		}
	}

	if err := iter.Error(); err != nil {
		return nil, err
	}

	return results, nil
}

// collectSortIndexHit reads the value for an index entry key, applies filters
// in Go, and returns the hit when the row passes.
func (e *pebbleEngine) collectSortIndexHit(
	fullKey []byte,
	col string,
	filters []metaengine.FilterSpec,
) (sortIndexHit, bool) {
	primaryKey := extractPrimaryKeyFromIndex(fullKey)

	val, closer, err := e.db.Get(mapKey(col, primaryKey))
	if err != nil {
		return sortIndexHit{}, false
	}

	valCopy := append([]byte(nil), val...)
	_ = closer.Close()

	if len(filters) > 0 {
		decoded := decodeJSON(valCopy)

		if !metaengine.PassesFilterSpecs(decoded, filters) {
			return sortIndexHit{}, false
		}
	}

	return sortIndexHit{primaryKey: primaryKey, value: valCopy}, true
}

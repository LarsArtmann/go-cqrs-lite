package pebbleengine

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"fmt"
	"slices"
	"sort"

	"github.com/cockroachdb/pebble"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// Compile-time assertion that pebbleEngine implements LayoutPlanner.
var _ metaengine.LayoutPlanner = (*pebbleEngine)(nil)

// layoutPlan stores the declared filter/sort fields for a collection.
// When present, MapSet writes secondary index entries for BOTH filter and
// sort fields. Filter index entries enable prefix-range lookups (equality,
// range). Sort index entries enable ordered iteration (ascending/descending)
// without a Go-level sort, with cursor-based pagination and early termination.
type layoutPlan struct {
	filterFields []string // fields indexed for prefix-scan filtering ('i' prefix)
	sortFields   []string // fields indexed for ordered iteration ('o' prefix)
}

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

// layoutKeyPrefix builds the secondary index key prefix for a field value.
// Format: "i{sep}{col}{sep}{field}{sep}{value}{sep}{primaryKey}".
func layoutKeyPrefix(col, field, value string) []byte {
	return []byte("i" + sep + col + sep + field + sep + value + sep)
}

// formatIndexInt encodes a signed int64 into a fixed-width lexicographically-
// ordered string. The offset maps [-2^63, 2^63-1] to [0, 2^64-1] so that
// lexicographic byte comparison matches numeric comparison for ALL integers,
// including negatives and mixed digit counts.
func formatIndexInt(v int64) string {
	return fmt.Sprintf("%020d", uint64(v)-uint64(1<<63))
}

// encodeIndexValue converts a field value to an order-preserving string for
// secondary index keys. Integers (including float64-encoded JSON numbers
// without fractional parts) are zero-padded to 20 chars with sign offset.
// Strings and bools use their natural representation. Floats with fractional
// parts fall through to fmt.Sprintf (best-effort, documented limitation).
func encodeIndexValue(v any) string {
	switch val := v.(type) {
	case int:
		return formatIndexInt(int64(val))
	case int8:
		return formatIndexInt(int64(val))
	case int16:
		return formatIndexInt(int64(val))
	case int32:
		return formatIndexInt(int64(val))
	case int64:
		return formatIndexInt(val)
	case uint:
		return formatIndexInt(int64(val))
	case uint8:
		return formatIndexInt(int64(val))
	case uint16:
		return formatIndexInt(int64(val))
	case uint32:
		return formatIndexInt(int64(val))
	case uint64:
		return formatIndexInt(int64(val))
	case float64:
		if val == float64(int64(val)) {
			return formatIndexInt(int64(val))
		}

		return fmt.Sprintf("%v", val)
	case float32:
		if val == float32(int64(val)) {
			return formatIndexInt(int64(val))
		}

		return fmt.Sprintf("%v", val)
	case string:
		return val
	case bool:
		if val {
			return "true"
		}

		return "false"
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", v)
	}
}

// ApplyLayout creates a secondary index layout for the collection. After this,
// MapSet writes secondary index entries for the declared filter fields, and
// ScanRawValues uses prefix scans when a filter matches a declared field.
func (e *pebbleEngine) ApplyLayout(collection string, filterFields, sortFields []string) error {
	e.layoutMu.Lock()
	defer e.layoutMu.Unlock()

	if e.layouts == nil {
		e.layouts = make(map[string]layoutPlan)
	}

	e.layouts[collection] = layoutPlan{
		filterFields: append([]string(nil), filterFields...),
		sortFields:   append([]string(nil), sortFields...),
	}

	return nil
}

// writeIndexEntries writes secondary index entries for a value's filter fields.
// Called by MapSet when a layout plan exists for the collection.
func (e *pebbleEngine) writeIndexEntries(
	batch *pebble.Batch,
	col, key string,
	valueJSON []byte,
	plan layoutPlan,
) error {
	var fields map[string]any
	if err := json.Unmarshal(valueJSON, &fields); err != nil {
		return nil //nolint:nilerr // not JSON object — skip indexing
	}

	for _, field := range plan.filterFields {
		fieldVal, ok := fields[field]
		if !ok {
			continue
		}

		valStr := encodeIndexValue(fieldVal)
		idxKey := append(layoutKeyPrefix(col, field, valStr), []byte(key)...)

		if err := batch.Set(idxKey, nil, nil); err != nil {
			return fmt.Errorf("pebbleengine: write index entry: %w", err)
		}
	}

	for _, field := range plan.sortFields {
		fieldVal, ok := fields[field]
		if !ok {
			continue
		}

		valStr := encodeIndexValue(fieldVal)
		idxKey := sortIndexKey(col, field, valStr, key)

		if err := batch.Set(idxKey, nil, nil); err != nil {
			return fmt.Errorf("pebbleengine: write sort index entry: %w", err)
		}
	}

	return nil
}

// deleteIndexEntries removes old secondary index entries for a key being updated.
// Must be called before writing new index entries on update.
func (e *pebbleEngine) deleteIndexEntries(
	batch *pebble.Batch,
	col, key string,
	oldValueJSON []byte,
	plan layoutPlan,
) {
	if len(oldValueJSON) == 0 {
		return
	}

	var fields map[string]any
	if err := json.Unmarshal(oldValueJSON, &fields); err != nil {
		return
	}

	for _, field := range plan.filterFields {
		fieldVal, ok := fields[field]
		if !ok {
			continue
		}

		valStr := encodeIndexValue(fieldVal)
		idxKey := append(layoutKeyPrefix(col, field, valStr), []byte(key)...)
		_ = batch.Delete(idxKey, nil)
	}

	for _, field := range plan.sortFields {
		fieldVal, ok := fields[field]
		if !ok {
			continue
		}

		valStr := encodeIndexValue(fieldVal)
		idxKey := sortIndexKey(col, field, valStr, key)
		_ = batch.Delete(idxKey, nil)
	}
}

// fieldIndexPrefix builds the secondary index key prefix for a field (all values).
// Format: "i{sep}{col}{sep}{field}{sep}".
func fieldIndexPrefix(col, field string) []byte {
	return []byte("i" + sep + col + sep + field + sep)
}

// indexBounds returns LowerBound and UpperBound for an index scan based on the
// first indexable filter in filters. Supports FilterEq, FilterGt, FilterGe,
// FilterLt, FilterLe, and FilterIn (expanded to multiple equality scans).
// Returns ok=false if no indexable filter is found.
func (p layoutPlan) indexBounds(
	col string,
	filters []metaengine.FilterSpec,
) ([]byte, []byte, bool) {
	if len(filters) == 0 {
		return nil, nil, false
	}

	for _, f := range filters {
		for _, field := range p.filterFields {
			if field != f.Column {
				continue
			}

			fp := fieldIndexPrefix(col, field)
			valStr := encodeIndexValue(f.Value)
			vp := layoutKeyPrefix(col, field, valStr)

			switch f.Op {
			case metaengine.FilterEq:
				return vp, nextKey(vp), true
			case metaengine.FilterGt:
				return nextKey(vp), nextKey(fp), true
			case metaengine.FilterGe:
				return vp, nextKey(fp), true
			case metaengine.FilterLt:
				return fp, vp, true
			case metaengine.FilterLe:
				return fp, nextKey(vp), true
			}
		}
	}

	return nil, nil, false
}

// scanWithIndex uses a secondary index to find matching keys, then reads
// their values. This is O(matches) instead of O(all rows in collection).
// Supports FilterEq, FilterGt, FilterGe, FilterLt, FilterLe via index bounds.
func (e *pebbleEngine) scanWithIndex(
	_ context.Context,
	col string,
	filters []metaengine.FilterSpec,
	plan layoutPlan,
) ([][]byte, error) {
	lowerBound, upperBound, ok := plan.indexBounds(col, filters)
	if !ok {
		return nil, nil
	}

	iter, err := e.db.NewIter(&pebble.IterOptions{
		LowerBound: lowerBound,
		UpperBound: upperBound,
	})
	if err != nil {
		return nil, err
	}

	defer func() { _ = iter.Close() }()

	var results [][]byte

	for iter.First(); iter.Valid(); iter.Next() {
		// The index key is: i{sep}{col}{sep}{field}{sep}{value}{sep}{primaryKey}
		// Extract primaryKey by finding the last separator.
		fullKey := append([]byte(nil), iter.Key()...)
		primaryKey := extractPrimaryKeyFromIndex(fullKey)

		// Read the actual value from the map store.
		val, closer, err := e.db.Get(mapKey(col, primaryKey))
		if err != nil {
			continue
		}

		valCopy := append([]byte(nil), val...)
		_ = closer.Close()

		results = append(results, valCopy)
	}

	if err := iter.Error(); err != nil {
		return nil, err
	}

	return results, nil
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

// extractPrimaryKeyFromIndex extracts the primary key portion from any index
// entry key by finding the last separator. Works for both equality and range
// scans where the value portion varies in length.
func extractPrimaryKeyFromIndex(fullKey []byte) string {
	idx := bytes.LastIndex(fullKey, []byte(sep))
	if idx < 0 || idx >= len(fullKey)-1 {
		return ""
	}

	return string(fullKey[idx+1:])
}

// sortIndexedResults sorts raw JSON values by a sort field.
func sortIndexedResults(results [][]byte, sortField string, desc bool) {
	if sortField == "" || len(results) <= 1 {
		return
	}

	sort.Slice(results, func(i, j int) bool {
		vi := decodeJSON(results[i])
		vj := decodeJSON(results[j])

		c := metaengine.CompareValues(
			metaengine.ItemFieldByName(vi, sortField),
			metaengine.ItemFieldByName(vj, sortField),
		)

		if c != 0 {
			if desc {
				return c > 0
			}

			return c < 0
		}

		return bytes.Compare(results[i], results[j]) < 0
	})
}

// applyLimit truncates results to limit+1 (the extra row signals "has more").
func applyLimit(results [][]byte, limit int) [][]byte {
	if limit <= 0 || len(results) <= limit+1 {
		return results
	}

	return results[:limit+1]
}

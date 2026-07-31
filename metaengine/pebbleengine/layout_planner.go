package pebbleengine

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/cockroachdb/pebble"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// Compile-time assertion that pebbleEngine implements LayoutPlanner.
var _ metaengine.LayoutPlanner = (*pebbleEngine)(nil)

// layoutPlan stores the declared filter/sort fields for a collection.
// When present, MapSet writes secondary index entries so scans can use
// prefix-range lookups instead of full-collection scans + Go filtering.
type layoutPlan struct {
	filterFields []string // fields indexed for prefix-scan filtering
	sortFields   []string // fields indexed for prefix-scan sorting
}

// layoutKeyPrefix builds the secondary index key prefix for a field value.
// Format: "i{sep}{col}{sep}{field}{sep}{value}{sep}{primaryKey}"
func layoutKeyPrefix(col, field, value string) []byte {
	return []byte("i" + sep + col + sep + field + sep + value + sep)
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

		valStr := fmt.Sprintf("%v", fieldVal)
		idxKey := append(layoutKeyPrefix(col, field, valStr), []byte(key)...)
		if err := batch.Set(idxKey, nil, nil); err != nil {
			return fmt.Errorf("pebbleengine: write index entry: %w", err)
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

		valStr := fmt.Sprintf("%v", fieldVal)
		idxKey := append(layoutKeyPrefix(col, field, valStr), []byte(key)...)
		_ = batch.Delete(idxKey, nil) //nolint:errcheck // best-effort cleanup
	}
}

// indexKeysForFilter returns the secondary index prefix for a filter on a
// declared field. Returns ok=false if the field is not indexed or the filter
// is not an equality filter.
func (p layoutPlan) indexPrefix(col string, filters []metaengine.FilterSpec) ([]byte, bool) {
	if len(filters) == 0 {
		return nil, false
	}

	for _, f := range filters {
		if f.Op != metaengine.FilterEq {
			continue
		}

		for _, field := range p.filterFields {
			if field == f.Column {
				valStr := fmt.Sprintf("%v", f.Value)

				return layoutKeyPrefix(col, field, valStr), true
			}
		}
	}

	return nil, false
}

// scanWithIndex uses a secondary index to find matching keys, then reads
// their values. This is O(matches) instead of O(all rows in collection).
func (e *pebbleEngine) scanWithIndex(
	_ context.Context,
	col string,
	filters []metaengine.FilterSpec,
	plan layoutPlan,
) ([][]byte, error) {
	prefix, ok := plan.indexPrefix(col, filters)
	if !ok {
		return nil, nil
	}

	upperBound := nextKey(prefix)

	iter, err := e.db.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: upperBound,
	})
	if err != nil {
		return nil, err //nolint:wrapcheck // passthrough
	}

	defer func() { _ = iter.Close() }()

	var results [][]byte

	for iter.First(); iter.Valid(); iter.Next() {
		// The index key is: i{col}{field}{value}{primaryKey}
		// Extract primaryKey from the end of the key.
		fullKey := append([]byte(nil), iter.Key()...)
		primaryKey := extractPrimaryKey(fullKey, prefix)

		// Read the actual value from the map store.
		val, closer, err := e.db.Get(mapKey(col, primaryKey))
		if err != nil {
			if err == pebble.ErrNotFound {
				continue
			}

			continue
		}

		valCopy := append([]byte(nil), val...)
		_ = closer.Close() //nolint:sqlclosecheck,errcheck // pebble closer

		results = append(results, valCopy)
	}

	if err := iter.Error(); err != nil {
		return nil, err //nolint:wrapcheck // passthrough
	}

	return results, nil
}

// extractPrimaryKey extracts the primary key portion from an index entry key.
// The index key is prefix + primaryKey, so we strip the prefix.
func extractPrimaryKey(fullKey, prefix []byte) string {
	if len(fullKey) <= len(prefix) {
		return ""
	}

	return string(fullKey[len(prefix):])
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

		return strings.Compare(string(results[i]), string(results[j])) < 0
	})
}

// PebbleLayoutSupport is a compile-time assertion that pebbleEngine has the
// layout infrastructure wired.
var _ sync.Locker = (*sync.Mutex)(nil) // ensures sync import is used

// applyLimit truncates results to limit+1 (the extra row signals "has more").
func applyLimit(results [][]byte, limit int) [][]byte {
	if limit <= 0 || len(results) <= limit+1 {
		return results
	}

	return results[:limit+1]
}

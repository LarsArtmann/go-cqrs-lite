package pebbleengine

import (
	"bytes"
	"encoding/json/v2"
	"fmt"

	"github.com/cockroachdb/pebble"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// indexIntOffset biases int64 to its uint64 rank so negative values
// sort before positives in the zero-padded key form.
const indexIntOffset = uint64(1 << 63)

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
	return fmt.Sprintf("%020d", uint64(v)-indexIntOffset)
}

// encodeIndexValue converts a field value to an order-preserving string for
// secondary index keys. Integers (including float64-encoded JSON numbers
// without fractional parts) are zero-padded to 20 chars with sign offset.
// Strings and bools use their natural representation.
//
// LIMITATION: Floats with fractional parts (e.g. 3.14) fall through to
// fmt.Sprintf("%v"), which does NOT preserve lexicographic ordering
// ("10.5" < "3.14" lexicographically). Filter/sort pushdown on float
// columns may produce wrong results. Use integer-scaled values (cents,
// milliseconds) instead of floats for indexed columns.
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
// MapSet writes secondary index entries for the declared filter and sort
// fields, and ScanRawValues uses prefix scans when a filter matches a
// declared field. Rows that already exist are indexed in one pass, so they
// stay visible to indexed scans; the call therefore costs one scan of the
// collection and fails without publishing the plan if that scan fails.
func (e *pebbleEngine) ApplyLayout(collection string, filterFields, sortFields []string) error {
	plan := layoutPlan{
		filterFields: append([]string(nil), filterFields...),
		sortFields:   append([]string(nil), sortFields...),
	}

	// layoutMu is held across the backfill so a concurrent MapSet either
	// lands before the scan (and is indexed here) or sees the published plan
	// (and indexes itself); no row can slip between the two.
	e.layoutMu.Lock()
	defer e.layoutMu.Unlock()

	if err := e.backfillIndex(collection, plan); err != nil {
		return err
	}

	if e.layouts == nil {
		e.layouts = make(map[string]layoutPlan)
	}

	e.layouts[collection] = plan

	return nil
}

// writeIndexEntries writes secondary index entries for a value's filter and
// sort fields. Called by MapSet when a layout plan exists for the collection.
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

	return applyIndexEntries(fields, plan, col, key, func(kind string, idxKey []byte) error {
		if err := batch.Set(idxKey, nil, nil); err != nil {
			return fmt.Errorf("pebbleengine: write %s: %w", kind, err)
		}

		return nil
	})
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

	_ = applyIndexEntries(fields, plan, col, key, func(_ string, idxKey []byte) error {
		return batch.Delete(idxKey, nil)
	})
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
			case metaengine.FilterNe, metaengine.FilterIn:
				// Not index-narrowable: these ops span the whole field
				// range — the caller falls back to the unindexed scan.
				return nil, nil, false
			}
		}
	}

	return nil, nil, false
}

// scanWithIndex uses a secondary index to find matching keys, then reads
// their values. This is O(matches) instead of O(all rows in collection).
// Supports FilterEq, FilterGt, FilterGe, FilterLt, FilterLe via index bounds.
// The index narrows to ONE filter; every filter (including that one) is then
// re-applied in Go, because rows can carry stale or missing index entries.
// indexed is false when no filter can narrow the range; the caller then
// falls back to the unindexed scan.
func (e *pebbleEngine) scanWithIndex(
	col string,
	filters []metaengine.FilterSpec,
	plan layoutPlan,
) ([]sortIndexHit, bool, error) {
	lowerBound, upperBound, ok := plan.indexBounds(col, filters)
	if !ok {
		return nil, false, nil
	}

	var hits []sortIndexHit

	iter, err := e.db.NewIter(&pebble.IterOptions{
		LowerBound: lowerBound,
		UpperBound: upperBound,
	})
	if err != nil {
		return nil, true, err
	}

	defer metaengine.DeferClose(iter)

	for iter.First(); iter.Valid(); iter.Next() {
		if hit, ok := e.collectSortIndexHit(iter.Key(), col, filters); ok {
			hits = append(hits, hit)
		}
	}

	if err := iter.Error(); err != nil {
		return nil, true, err
	}

	return hits, true, nil
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

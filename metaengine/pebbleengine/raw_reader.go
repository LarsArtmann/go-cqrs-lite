package pebbleengine

import (
	"context"
	"errors"

	"github.com/cockroachdb/pebble"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// Compile-time assertions for the raw reader interfaces.
var (
	_ metaengine.RawValueReader = (*pebbleEngine)(nil)
	_ metaengine.RawScanReader  = (*pebbleEngine)(nil)
)

// --- RawValueReader ---

// getPebbleRaw reads the raw bytes for a collection key from Pebble, handling
// ErrNotFound and closer lifecycle. The returned slice is a copy ( Pebble
// values are only valid until closer.Close). Shared by MapGet and GetRawValue.
func (e *pebbleEngine) getPebbleRaw(col string, key any) ([]byte, bool, error) {
	val, closer, err := e.db.Get(mapKey(col, encodeKeyStr(key)))
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil, false, nil
		}

		return nil, false, err
	}

	defer func() { _ = closer.Close() }() //cqrs-lint:ignore(C015) pebble closer, error is always nil

	return append([]byte(nil), val...), true, nil
}

// GetRawValue reads the raw JSON bytes for a key without decoding to any.
// The TypedReader and ExecuteTyped paths prefer this over MapGet for point
// lookups, decoding directly to the target type V (1 JSON op instead of 2:
// decode-to-any + reify-from-map).
func (e *pebbleEngine) GetRawValue(_ context.Context, col string, key any) ([]byte, bool, error) {
	return e.getPebbleRaw(col, key)
}

// --- RawScanReader ---

// ScanRawValues scans collection values as raw JSON bytes without decoding
// each row to any. Filters and sorting are applied in Go (Pebble has no SQL
// engine), but the returned bytes allow the caller to decode directly to the
// target type V, avoiding the reify-from-map reflection step.
func (e *pebbleEngine) ScanRawValues(
	ctx context.Context,
	col string,
	filters []metaengine.FilterSpec,
	sortSpec *metaengine.SortSpec,
	cursor any,
	limit int,
) ([][]byte, error) {
	// Fast path: use secondary index if a layout plan exists and filters match.
	e.layoutMu.Lock()
	plan, hasLayout := e.layouts[col]
	e.layoutMu.Unlock()

	if hasLayout {
		// Sort index path: ordered iteration via the sort-prefix index when the
		// sort column matches a declared sort field. Avoids the Go-level sort and
		// enables early termination at limit+1 matching rows.
		if sortSpec != nil && plan.hasSortField(sortSpec.Column) {
			return e.scanWithSortIndex(ctx, col, filters, sortSpec, cursor, limit)
		}

		// Filter index path: prefix-range lookup when a filter matches a declared
		// filter field. Results are then Go-sorted, cursor-paginated, and limited.
		if indexed, err := e.scanWithIndex(ctx, col, filters, plan); err == nil && indexed != nil {
			if sortSpec != nil {
				sortIndexedResults(indexed, sortSpec.Column, sortSpec.Desc)

				if cursor != nil {
					indexed = paginateIndexedResults(indexed, sortSpec, cursor)
				}
			}

			return applyLimit(indexed, limit), nil
		}
	}

	prefix := collectionPrefix(col)
	upperBound := nextKey(prefix)

	iter, err := e.db.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: upperBound,
	})
	if err != nil {
		return nil, err
	}

	defer func() { _ = iter.Close() }()

	var pairs []kvPair

	for iter.First(); iter.Valid(); iter.Next() {
		raw := append([]byte(nil), iter.Value()...)
		decoded := decodeJSON(raw)

		if len(filters) > 0 && !metaengine.PassesFilterSpecs(decoded, filters) {
			continue
		}

		pairs = append(pairs, kvPair{
			key:   append([]byte(nil), iter.Key()...),
			value: decoded,
			raw:   raw,
		})
	}

	if err := iter.Error(); err != nil {
		return nil, err
	}

	// Build sort comparator from SortSpec. Direction (Desc) is encoded into
	// the comparator so the keyset pagination in sortAndPaginate handles both
	// ascending and descending correctly. extractOrDirect lets the same
	// comparator handle item-vs-item sort and item-vs-cursor pagination.
	var sortFn func(a, b any) int

	if sortSpec != nil {
		sortFn = func(a, b any) int {
			va := extractOrDirect(a, sortSpec.Column)
			vb := extractOrDirect(b, sortSpec.Column)
			c := metaengine.CompareValues(va, vb)

			if sortSpec.Desc {
				return -c
			}

			return c
		}
	}

	pairs = sortAndPaginate(pairs, sortFn, cursor, limit)

	results := make([][]byte, len(pairs))
	for i, p := range pairs {
		results[i] = p.raw
	}

	return results, nil
}

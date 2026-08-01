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
) (metaengine.RawScanResult, error) {
	e.layoutMu.Lock()
	plan, hasLayout := e.layouts[col]
	e.layoutMu.Unlock()

	if hasLayout {
		if sortSpec != nil && plan.hasSortField(sortSpec.Column) {
			rows, err := e.scanWithSortIndex(ctx, col, filters, sortSpec, cursor, limit)
			if err != nil {
				return metaengine.RawScanResult{}, err
			}

			return trimRaw(rows, limit), nil
		}

		if indexed, err := e.scanWithIndex(ctx, col, filters, plan); err == nil && indexed != nil {
			return trimRaw(processFilterIndex(indexed, sortSpec, cursor, limit), limit), nil
		}
	}

	prefix := collectionPrefix(col)
	upperBound := nextKey(prefix)

	iter, err := e.db.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: upperBound,
	})
	if err != nil {
		return metaengine.RawScanResult{}, err
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
		return metaengine.RawScanResult{}, err
	}

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

	hasMore := limit > 0 && len(pairs) > limit
	if hasMore {
		pairs = pairs[:limit]
	}

	results := make([][]byte, len(pairs))
	for i, p := range pairs {
		results[i] = p.raw
	}

	return metaengine.RawScanResult{Items: results, HasMore: hasMore}, nil
}

package pebbleengine

import (
	"bytes"
	"context"
	"errors"
	"sort"

	"github.com/cockroachdb/pebble"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// Compile-time assertions for the raw reader interfaces.
var (
	_ metaengine.RawValueReader = (*pebbleEngine)(nil)
	_ metaengine.RawScanReader  = (*pebbleEngine)(nil)
)

// --- RawValueReader ---

// GetRawValue reads the raw JSON bytes for a key without decoding to any.
// The TypedReader and ExecuteTyped paths prefer this over MapGet for point
// lookups, decoding directly to the target type V (1 JSON op instead of 2:
// decode-to-any + reify-from-map).
func (e *pebbleEngine) GetRawValue(_ context.Context, col string, key any) ([]byte, bool, error) {
	val, closer, err := e.db.Get(mapKey(col, encodeKeyStr(key)))
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil, false, nil
		}

		return nil, false, err //nolint:wrapcheck // passthrough
	}

	defer func() { _ = closer.Close() }() //cqrs-lint:ignore(C015) pebble closer, error is always nil

	// Pebble values are only valid until closer.Close — copy.
	return append([]byte(nil), val...), true, nil
}

// --- RawScanReader ---

// ScanRawValues scans collection values as raw JSON bytes without decoding
// each row to any. Filters and sorting are applied in Go (Pebble has no SQL
// engine), but the returned bytes allow the caller to decode directly to the
// target type V, avoiding the reify-from-map reflection step.
func (e *pebbleEngine) ScanRawValues(
	_ context.Context,
	col string,
	filters []metaengine.FilterSpec,
	sortSpec *metaengine.SortSpec,
	cursor any,
	limit int,
) ([][]byte, error) {
	prefix := collectionPrefix(col)
	upperBound := nextKey(prefix)

	iter, err := e.db.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: upperBound,
	})
	if err != nil {
		return nil, err //nolint:wrapcheck // passthrough
	}

	defer func() { _ = iter.Close() }()

	type kv struct {
		key   []byte
		value []byte // raw JSON bytes (owned copy)
	}

	var pairs []kv

	for iter.First(); iter.Valid(); iter.Next() {
		raw := append([]byte(nil), iter.Value()...)

		if len(filters) > 0 {
			decoded := decodeJSON(raw)

			if !metaengine.PassesFilterSpecs(decoded, filters) {
				continue
			}
		}

		pairs = append(pairs, kv{
			key:   append([]byte(nil), iter.Key()...),
			value: raw,
		})
	}

	if err := iter.Error(); err != nil {
		return nil, err //nolint:wrapcheck // passthrough
	}

	// Sort in Go (Pebble has no secondary index).
	if sortSpec != nil {
		sort.Slice(pairs, func(i, j int) bool {
			vi := decodeJSON(pairs[i].value)
			vj := decodeJSON(pairs[j].value)

			c := metaengine.CompareValues(
				metaengine.ItemFieldByName(vi, sortSpec.Column),
				metaengine.ItemFieldByName(vj, sortSpec.Column),
			)

			if c != 0 {
				if sortSpec.Desc {
					return c > 0
				}

				return c < 0
			}

			return bytes.Compare(pairs[i].key, pairs[j].key) < 0
		})

		// Keyset pagination: skip items at or before the cursor.
		if cursor != nil {
			filtered := pairs[:0]

			for _, p := range pairs {
				v := decodeJSON(p.value)
				fieldVal := metaengine.ItemFieldByName(v, sortSpec.Column)
				c := metaengine.CompareValues(fieldVal, cursor)

				if sortSpec.Desc {
					if c >= 0 {
						continue
					}
				} else {
					if c <= 0 {
						continue
					}
				}

				filtered = append(filtered, p)
			}

			pairs = filtered
		}
	}

	truncLimit := 0
	if limit > 0 {
		truncLimit = limit + 1
	}

	if truncLimit > 0 && len(pairs) > truncLimit {
		pairs = pairs[:truncLimit]
	}

	results := make([][]byte, len(pairs))
	for i, p := range pairs {
		results[i] = p.value
	}

	return results, nil
}


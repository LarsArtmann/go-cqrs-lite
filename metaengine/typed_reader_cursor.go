package metaengine

import (
	"context"
	"fmt"
)

func (r *TypedReader[V]) ScanPage(ctx context.Context, opts ...ScanOption) ([]V, *Cursor, error) {
	result, err := r.Scan(ctx, opts...)
	if err != nil {
		return nil, nil, err
	}

	cfg := scanConfig{limit: 100}
	for _, opt := range opts {
		opt(&cfg)
	}

	if len(result) == 0 || cfg.limit <= 0 || len(result) < cfg.limit {
		return result, nil, nil
	}

	nextVal := extractCursorValue(result[cfg.limit-1], cfg)

	return result, &Cursor{Value: nextVal}, nil
}

// trimAndCache trims results to the limit and, when a PrefetchCache is attached,
// caches the extra rows beyond the limit for the next page. The next page's
// cursor key is derived from the last returned item's sort field (or the item
// itself when no sort is specified).
func (r *TypedReader[V]) trimAndCache(result []V, cfg scanConfig) []V {
	if r.prefetch != nil && cfg.limit > 0 && len(result) > cfg.limit {
		cursorVal := extractCursorValue(result[cfg.limit-1], cfg)
		nextKey := prefetchCursorKey(r.collection, cursorVal)
		extra := make([]any, len(result)-cfg.limit)

		for i, v := range result[cfg.limit:] {
			extra[i] = v
		}

		r.prefetch.Put(nextKey, extra)
	}

	return trimToLimit(result, cfg.limit)
}

// extractCursorValue derives the raw cursor value from an item using the scan
// config's sort specification. When a sort spec is set, the cursor is the sort
// column value; otherwise the whole item is used.
func extractCursorValue[V any](item V, cfg scanConfig) any {
	if cfg.sort != nil {
		return itemFieldByName(item, cfg.sort.Column)
	}

	if len(cfg.sortCols) > 0 {
		return itemFieldByName(item, cfg.sortCols[0].Column)
	}

	return item
}

// rawCursorValue extracts the raw cursor value from cfg.cursor, unwrapping
// a *Cursor (from WithCursorString) to its Value field for engine consumption.
func rawCursorValue(cursor any) any {
	if c, ok := cursor.(*Cursor); ok {
		return c.Value
	}

	return cursor
}

// prefetchCursorKey builds the PrefetchCache lookup key from a collection name
// and cursor value. The cursor is normalized to a *Cursor and encoded via
// [Cursor.Encode], producing an opaque base64 string that is HTTP-safe and
// deterministic regardless of whether the caller passed a raw value
// (WithCursor) or an encoded string (WithCursorString). Both trimAndCache
// (cache write) and the prefetch check in Scan (cache read) use this function,
// ensuring the key formats always match.
func prefetchCursorKey(collection string, cursorVal any) string {
	var c *Cursor

	switch v := cursorVal.(type) {
	case *Cursor:
		c = v
	case Cursor:
		c = &v
	default:
		c = &Cursor{Value: cursorVal}
	}

	encoded, err := c.Encode()
	if err != nil || encoded == "" {
		return fmt.Sprintf("%s:%v", collection, cursorVal)
	}

	return collection + ":" + encoded
}

// trimToLimit trims the result to the limit (engines return limit+1 rows for
// has-more detection, which TypedReader.Scan discards).
func trimToLimit[V any](result []V, limit int) []V {
	if limit > 0 && len(result) > limit {
		return result[:limit]
	}

	return result
}

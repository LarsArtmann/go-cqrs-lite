package metaengine

import (
	"context"
	"fmt"
)

// pageMeta carries engine pagination metadata out of the scan paths so
// ScanPage can prefer the engine-issued compound cursor over reflecting the
// sort field out of the last item. hasMore is the engine's own probe result;
// nextCursor is [ScanResult.NextCursor] (nil on paths that cannot derive one).
type pageMeta struct {
	nextCursor any
	hasMore    bool
}

// ScanPage performs a scan and returns both the results and the next-page
// cursor for keyset pagination. When the engine issued a compound cursor
// ([ScanResult.NextCursor], a [SortKeyCursor] over the sort value + byte key
// of the last row) it is returned as-is, so tie-heavy datasets paginate with
// neither drops nor duplicates; ScanPage also honors the engine's has-more
// probe and returns a nil cursor at exact end-of-stream (a full final page
// no longer mints a cursor into an empty next page). Otherwise the cursor is
// derived from the sort field of the last returned item (the legacy value
// cursor — tie-lossy, retained for paths without engine cursor support and
// for PrefetchCache-attached readers, whose 2x fetch makes the engine's
// fetch-boundary cursor unrepresentative of the page boundary). Pass the
// cursor back via WithCursor(cursor.Value) on the next call, or use
// WithCursorString(cursor.Encode()) for an HTTP-safe opaque cursor string
// that round-trips through the PrefetchCache.
//
// When a PrefetchCache is attached, ScanPage auto-populates it: extra rows
// beyond the limit are cached so the next page request is served from cache
// instead of hitting the engine.
//
// Returns (items, nil cursor, nil) when there are no more pages.
func (r *TypedReader[V]) ScanPage(ctx context.Context, opts ...ScanOption) ([]V, *Cursor, error) {
	result, meta, err := r.scanItems(ctx, opts...)
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

	if r.prefetch == nil && meta.nextCursor != nil {
		if meta.hasMore {
			return result, &Cursor{Value: meta.nextCursor}, nil
		}

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

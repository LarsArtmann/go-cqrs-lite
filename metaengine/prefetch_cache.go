package metaengine

import "sync"

// --- Cursor pre-fetch cache ---

// PrefetchCache caches scan results beyond the requested limit for the next
// page request. Eliminates the limit+1 round-trip pattern.
// Thread-safe: safe for concurrent use from multiple goroutines.
//
// Attach it one of two ways — either is enough, the cache is what carries the
// state, not the reader:
//
//	// (a) reader-level default, for a long-lived reader
//	reader := metaengine.NewReader[Item](store, "items").
//	    WithPrefetch(metaengine.NewPrefetchCache())
//
//	// (b) per scan call, when the reader is created per request
//	reader.ScanPage(ctx, metaengine.WithPrefetch(cache),
//	    metaengine.WithLimit(50), metaengine.WithCursor(cur.Value))
//
// Row caching is exact-boundary: the scan fetches 2×limit and the overflow is
// keyed by the NEXT page's cursor, so only a call that passes the same cache,
// collection and cursor is served from it (the engine never serves a cached
// page with different filters or sort — key the cache per query shape if you
// paginate more than one shape over the same collection). There is no eviction
// beyond [PrefetchCache.Clear]: one cache per pagination walk, cleared when the
// walk ends, is the intended lifecycle.
type PrefetchCache struct {
	mu    sync.RWMutex
	pages map[string][]any // cursor key → cached rows
}

// NewPrefetchCache creates a new pre-fetch cache.
func NewPrefetchCache() *PrefetchCache {
	return &PrefetchCache{pages: make(map[string][]any)}
}

// Get returns cached rows for a cursor key, or nil if not cached.
func (c *PrefetchCache) Get(key string) []any {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.pages[key]
}

// Put stores rows for a cursor key.
func (c *PrefetchCache) Put(key string, rows []any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.pages[key] = rows
}

// Len reports how many pages are currently cached — the observability handle for
// "is this cache actually being used", and the cache has no other eviction
// signal (it only grows until Clear).
func (c *PrefetchCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.pages)
}

// Clear removes all cached pages.
func (c *PrefetchCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.pages = make(map[string][]any)
}

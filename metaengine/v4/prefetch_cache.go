package metaengine

import "sync"

// --- Cursor pre-fetch cache ---

// PrefetchCache caches scan results beyond the requested limit for the next
// page request. Eliminates the limit+1 round-trip pattern.
// Thread-safe: safe for concurrent use from multiple goroutines.
//
// Attach it one of two ways — either is enough, the cache is what carries the
// state, not the TypedReader:
//
//	// (a) as the reader's default, for a long-lived TypedReader
//	r := metaengine.NewReader[Item](store, "items").
//	    WithPrefetch(metaengine.NewPrefetchCache())
//
//	// (b) per scan call, when a TypedReader is created per request
//	r.ScanPage(ctx, metaengine.WithPrefetch(cache),
//	    metaengine.WithLimit(50), metaengine.WithCursor(cur.Value))
//
// Keys are SHAPE-qualified (collection + filters + sort + cursor), so a page
// cached for one query shape is never served to another — the same cursor value
// under a different filter or sort is a different key. The scan fetches 2×limit
// and caches the overflow under the next page's cursor. There is no eviction
// beyond [PrefetchCache.Clear] (watch [PrefetchCache.Len]): one cache per
// pagination walk, cleared when the walk ends, is the intended lifecycle.
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

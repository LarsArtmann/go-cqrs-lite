// Package cache provides an Otter-backed caching decorator for
// [readmodel.Store].
//
// CachedStore transparently caches Get results in a high-performance
// in-memory cache (maypok86/otter/v2). Writes (Set, Delete) write-through to
// the underlying store and update the cache atomically. Scans bypass the cache
// (they are bulk reads that don't benefit from single-key caching).
//
//	cached, _ := cache.New(store,
//	    cache.WithCapacity[Todo, TodoID](10_000),
//	    cache.WithTTL[Todo, TodoID](5*time.Minute),
//	)
//
// Use CachedStore when read latency matters more than memory: projection read
// models that are read frequently but updated rarely are ideal candidates.
package cache

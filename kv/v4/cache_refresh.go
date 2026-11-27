package kv

import "context"

// loadThrough reads a cache miss from the store and caches it. Get is
// read-then-insert, and the two steps are not atomic: a Set or Delete that
// lands between them would cache a value the store no longer holds, and with
// the default TTL of 0 that entry would never expire. The write counter
// closes the window: a load that was overtaken is dropped again.
func (cs *Cache[T, K]) loadThrough(ctx context.Context, key string, keyID K) (*T, error) {
	before := cs.writes.Load()

	val, err := cs.store.Get(ctx, keyID) //nolint:wrapcheck // transparent cache proxy
	if err != nil {
		return nil, err
	}

	cached, err := cs.copy(val)
	if err != nil {
		return nil, err
	}

	cs.cache.Set(key, cached)

	// Checked after the insert: dropping the entry also covers a write whose
	// own cache update landed in between (the next Get then re-reads it).
	if cs.writes.Load() != before {
		cs.cache.Invalidate(key)
	}

	return val, nil
}

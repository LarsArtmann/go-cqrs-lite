package decider

import (
	"container/list"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// DefaultStateCacheCapacity is used when capacity <= 0.
const DefaultStateCacheCapacity = 128

// StateCache caches folded stream state in memory to avoid replaying the
// full event history on every load.
//
// The cache is best-effort: a miss falls back to the normal load path
// (snapshot store or full replay). The cache is process-local and does not
// participate in distributed consistency — each process maintains its own
// cache.
//
// When a cached entry exists at version V, the Repository loads only events
// after V (via store.LoadFromVersion) and folds them onto the cached state,
// producing the current state in O(new events) instead of O(total events).
//
// States stored in the cache must be treated as immutable by the consumer.
type StateCache[State any] interface {
	// Get retrieves the cached state and version for the given stream.
	// Returns ok=false if the stream is not in the cache.
	Get(ref id.StreamRef) (state State, version event.Version, ok bool)

	// Put stores the state and version for the given stream.
	Put(ref id.StreamRef, state State, version event.Version)

	// Invalidate removes the stream from the cache.
	Invalidate(ref id.StreamRef)
}

// NewStateCache creates an LRU-bounded StateCache with the given capacity.
// If capacity <= 0, DefaultStateCacheCapacity is used.
//
// The cache evicts the least recently used entry when capacity is exceeded.
func NewStateCache[State any](capacity int) StateCache[State] {
	if capacity <= 0 {
		capacity = DefaultStateCacheCapacity
	}

	return &lruCache[State]{ //nolint:exhaustruct // mu is zero-valued
		cap:   capacity,
		items: make(map[string]*list.Element, capacity),
		order: list.New(),
	}
}

type cacheEntry[State any] struct {
	ref     id.StreamRef
	state   State
	version event.Version
}

type lruCache[State any] struct {
	mu    sync.Mutex
	cap   int
	items map[string]*list.Element
	order *list.List
}

func (c *lruCache[State]) Get(ref id.StreamRef) (State, event.Version, bool) {
	var state State
	var version event.Version
	var ok bool

	c.locked(ref, func(key string) {
		elem, found := c.items[key]
		if !found {
			return
		}

		c.order.MoveToFront(elem)
		entry := elem.Value.(*cacheEntry[State]) //nolint:forcetypeassert // list only stores *cacheEntry

		state = entry.state
		version = entry.version
		ok = true
	})

	return state, version, ok
}

func (c *lruCache[State]) Put(ref id.StreamRef, state State, version event.Version) {
	c.locked(ref, func(key string) {
		if elem, found := c.items[key]; found {
			c.order.MoveToFront(elem)
			entry := elem.Value.(*cacheEntry[State]) //nolint:forcetypeassert // list only stores *cacheEntry
			entry.state = state
			entry.version = version

			return
		}

		entry := &cacheEntry[State]{ref: ref, state: state, version: version}
		elem := c.order.PushFront(entry)
		c.items[key] = elem

		if c.order.Len() > c.cap {
			oldest := c.order.Back()
			if oldest != nil {
				c.order.Remove(oldest)
				oldEntry := oldest.Value.(*cacheEntry[State]) //nolint:forcetypeassert // list only stores *cacheEntry
				delete(c.items, oldEntry.ref.String())
			}
		}
	})
}

func (c *lruCache[State]) Invalidate(ref id.StreamRef) {
	c.locked(ref, func(key string) {
		if elem, found := c.items[key]; found {
			c.order.Remove(elem)
			delete(c.items, key)
		}
	})
}

// locked executes fn while holding the cache mutex. It provides the key derived
// from ref so every mutating method follows the same lock + key-computation
// pattern.
func (c *lruCache[State]) locked(ref id.StreamRef, fn func(key string)) {
	c.mu.Lock()
	defer c.mu.Unlock()

	fn(ref.String())
}

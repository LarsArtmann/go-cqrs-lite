package decider

import (
	"container/list"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// DefaultStateCacheCapacity is used when capacity <= 0.
const DefaultStateCacheCapacity = 128

// StateCache caches folded aggregate state in memory to avoid replaying the
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
	// Get retrieves the cached state and version for the given aggregate.
	// Returns ok=false if the aggregate is not in the cache.
	Get(ref id.AggregateRef) (state State, version event.Version, ok bool)

	// Put stores the state and version for the given aggregate.
	Put(ref id.AggregateRef, state State, version event.Version)

	// Invalidate removes the aggregate from the cache.
	Invalidate(ref id.AggregateRef)
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
	ref     id.AggregateRef
	state   State
	version event.Version
}

type lruCache[State any] struct {
	mu    sync.Mutex
	cap   int
	items map[string]*list.Element
	order *list.List
}

func (c *lruCache[State]) Get(ref id.AggregateRef) (State, event.Version, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := ref.String()

	elem, ok := c.items[key]
	if !ok {
		var zero State

		return zero, 0, false
	}

	c.order.MoveToFront(elem)
	entry := elem.Value.(*cacheEntry[State]) //nolint:forcetypeassert // list only stores *cacheEntry

	return entry.state, entry.version, true
}

func (c *lruCache[State]) Put(ref id.AggregateRef, state State, version event.Version) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := ref.String()

	if elem, ok := c.items[key]; ok {
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
}

func (c *lruCache[State]) Invalidate(ref id.AggregateRef) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := ref.String()

	if elem, ok := c.items[key]; ok {
		c.order.Remove(elem)
		delete(c.items, key)
	}
}

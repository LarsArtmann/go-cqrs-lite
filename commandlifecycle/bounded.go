package commandlifecycle

// boundedMap is a FIFO-evicted string-keyed map for in-process caches whose
// re-seed cost (a store load, or a reset attempt counter) is a performance
// heuristic, not a correctness concern. Capacity <= 0 means unbounded.
//
// Not safe for concurrent use; callers must synchronize (Recorder and
// attemptTracker hold their own mutexes around these calls).
type boundedMap[T any] struct {
	entries map[string]T
	order   []string
	cap     int
}

const defaultCacheCapacity = 1024

func newBoundedMap[T any](capacity int) *boundedMap[T] {
	return &boundedMap[T]{
		entries: make(map[string]T),
		cap:     capacity,
	}
}

func (m *boundedMap[T]) get(key string) (T, bool) {
	v, ok := m.entries[key]

	return v, ok
}

// put inserts or updates key, evicting the oldest-inserted live entry when a
// new key arrives at capacity. Deletes leave stale order entries that are
// reclaimed lazily by future evictions.
func (m *boundedMap[T]) put(key string, val T) {
	if _, ok := m.entries[key]; !ok {
		if m.cap > 0 && len(m.entries) >= m.cap {
			m.evictOldest()
		}

		m.order = append(m.order, key)
	}

	m.entries[key] = val
}

func (m *boundedMap[T]) delete(key string) {
	delete(m.entries, key)
}

func (m *boundedMap[T]) len() int {
	return len(m.entries)
}

func (m *boundedMap[T]) evictOldest() {
	for len(m.order) > 0 {
		oldest := m.order[0]
		m.order = m.order[1:]

		if _, live := m.entries[oldest]; live {
			delete(m.entries, oldest)

			return
		}
	}
}

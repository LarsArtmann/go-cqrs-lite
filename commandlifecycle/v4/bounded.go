package commandlifecycle

// boundedMap is a FIFO-evicted string-keyed map for in-process caches whose
// re-seed cost (a store load, or a reset attempt counter) is a performance
// heuristic, not a correctness concern. Capacity <= 0 means unbounded.
//
// Deletes leave stale entries in the order slice; a stale counter triggers
// amortized compaction once stale entries dominate the live set, so a
// delete-heavy workload cannot grow order without bound. Not safe for
// concurrent use; callers must synchronize (Recorder and attemptTracker hold
// their own mutexes around these calls).
type boundedMap[T any] struct {
	entries map[string]T
	order   []string
	stale   int
	cap     int
}

const defaultCacheCapacity = 1024

// compactStaleThreshold is the minimum stale count before compaction is
// considered; below it the bookkeeping cost outweighs the reclaimed memory.
const compactStaleThreshold = 64

func newBoundedMap[T any](capacity int) *boundedMap[T] {
	return &boundedMap[T]{
		entries: make(map[string]T),
		order:   nil,
		stale:   0,
		cap:     capacity,
	}
}

func (m *boundedMap[T]) get(key string) (T, bool) {
	v, ok := m.entries[key]

	return v, ok
}

// put inserts or updates key, evicting the oldest-inserted live entry when a
// new key arrives at capacity.
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
	if _, ok := m.entries[key]; !ok {
		return
	}

	delete(m.entries, key)
	m.stale++
	m.maybeCompact()
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

		// A stale key was evicted from the order slice: the stale count
		// temporarily DIPS NEGATIVE here. That is correct, not a bug — the
		// key was already deleted from entries (counting it stale) but
		// never compacted out of order, and evictOldest is now doing the
		// compaction. Len(entries) is the authoritative size; stale is a
		// compaction heuristic only and recovers on the next delete.
		m.stale--
	}
}

// maybeCompact rebuilds the order slice from the live entries once stale
// entries dominate, releasing both the stale keys and the old backing array.
func (m *boundedMap[T]) maybeCompact() {
	if m.stale <= compactStaleThreshold || m.stale <= len(m.entries) {
		return
	}

	compacted := make([]string, 0, len(m.entries))

	for _, key := range m.order {
		if _, live := m.entries[key]; live {
			compacted = append(compacted, key)
		}
	}

	m.order = compacted
	m.stale = 0
}

// Package dedup provides a fixed-capacity ring buffer for deduplicating
// short-lived IDs at stream boundaries (e.g. the replay→live boundary in
// projection catch-up or SSE reconnection).
//
// The ring retains only the most recently added IDs, evicting the oldest entry
// when full. Both [Ring.Add] and [Ring.Has] are O(1). Memory is bounded at
// capacity regardless of how many IDs are added over the lifetime of the ring.
//
// [Ring] is NOT safe for concurrent use. Callers that share a ring across
// goroutines must synchronize access externally.
package dedup

// DefaultCapacity is a sensible ring capacity for replay→live deduplication.
//
// Overlapping events — those present in both a journal replay and the live
// stream — always appear at the tail of the replay sequence: they were
// published during the replay window, so they are the newest events in the
// journal. The live channel buffer is bounded (typically 100–256), so at most
// that many events can overlap. A ring of 1024 entries gives a 4–10x safety
// margin while bounding memory to ~90KB regardless of journal size.
const DefaultCapacity = 1024

// Ring is a fixed-capacity set of string IDs used to deduplicate events at a
// stream boundary. Only the most recently added IDs are retained.
type Ring struct {
	buf   []string
	idx   map[string]int // id → position in buf
	head  int            // next write position (oldest entry when full)
	count int            // entries currently in the ring
}

// NewRing creates a Ring with the given capacity. Falls back to DefaultCapacity
// if capacity <= 0 (defensive — callers may pass user-configured values).
func NewRing(capacity int) *Ring {
	if capacity <= 0 {
		capacity = DefaultCapacity
	}

	return &Ring{
		buf:   make([]string, capacity),
		idx:   make(map[string]int, capacity),
		head:  0,
		count: 0,
	}
}

// Add inserts an ID into the ring. If the ring is full, the oldest ID is
// evicted. Adding an ID that is already present is a no-op.
func (r *Ring) Add(id string) {
	if _, ok := r.idx[id]; ok {
		return
	}

	if r.count == len(r.buf) {
		delete(r.idx, r.buf[r.head])
	} else {
		r.count++
	}

	r.buf[r.head] = id
	r.idx[id] = r.head
	r.head = (r.head + 1) % len(r.buf)
}

// Has reports whether the ID is currently in the ring. A nil receiver always
// returns false, so callers can use a nil *Ring when no replay occurred.
func (r *Ring) Has(id string) bool {
	if r == nil {
		return false
	}

	_, ok := r.idx[id]

	return ok
}

// Len returns the number of IDs currently in the ring. A nil receiver returns 0.
func (r *Ring) Len() int {
	if r == nil {
		return 0
	}

	return r.count
}

// Capacity returns the maximum number of IDs the ring can hold.
// A nil receiver returns 0.
func (r *Ring) Capacity() int {
	if r == nil {
		return 0
	}

	return len(r.buf)
}

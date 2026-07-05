package http

// dedupRing is a fixed-capacity set of event IDs used to deduplicate events at
// the replay→live boundary.
//
// Only the most recently added IDs are retained. This is correct because
// overlapping events — those present in both the journal replay and the live
// stream — always appear at the tail of the replay sequence: they were
// published during the replay window (after AddClient, before the live loop
// starts), so they are the newest events in the journal. The live channel
// buffer is bounded (sseChannelBufSize), so at most that many events can
// overlap. A ring of dedupRingCapacity entries gives a generous safety margin.
//
// Both Add and Has are O(1). Memory is bounded at capacity regardless of
// journal size.
type dedupRing struct {
	buf   []string
	idx   map[string]int // id → position in buf
	head  int            // next write position (oldest entry when full)
	count int            // entries currently in the ring
}

// newDedupRing creates a dedupRing with the given capacity. Falls back to
// sseDedupRingCapacity if capacity <= 0 (defensive — callers pass constants).
func newDedupRing(capacity int) *dedupRing {
	if capacity <= 0 {
		capacity = sseDedupRingCapacity
	}

	return &dedupRing{
		buf: make([]string, capacity),
		idx: make(map[string]int, capacity),
	}
}

// Add inserts an ID into the ring. If the ring is full, the oldest ID is
// evicted. Adding an ID that is already present is a no-op.
func (r *dedupRing) Add(id string) {
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
// returns false, so callers can use a nil *dedupRing when no replay occurred.
func (r *dedupRing) Has(id string) bool {
	if r == nil {
		return false
	}

	_, ok := r.idx[id]

	return ok
}

// Len returns the number of IDs currently in the ring.
func (r *dedupRing) Len() int {
	return r.count
}

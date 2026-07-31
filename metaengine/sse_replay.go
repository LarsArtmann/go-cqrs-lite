package metaengine

import (
	"sync"
	"sync/atomic"
)

// SeqValue pairs a value with its monotonic sequence number from the replay
// journal. ServeSSE writes the Seq as the SSE event id field so clients can
// reconnect with Last-Event-ID.
type SeqValue[V any] struct {
	Seq   uint64
	Value V
}

// SSEReplay is a bounded ring buffer of recent value changes with monotonic
// sequence IDs. It enables Last-Event-ID reconnection for ServeSSE: when a
// client reconnects with the Last-Event-ID header, ServeSSE replays missed
// values from the journal before switching to live streaming.
//
// Create one via [Watcher.WithReplay] and let ServeSSE detect it automatically:
//
//	watcher := metaengine.NewWatcher[UserView](store, "users")
//	replay := watcher.WithReplay(1000) // buffer 1000 recent changes
//	defer watcher.Close()
//
// ServeSSE will write id: <seq> on every event. On reconnect, clients send
// Last-Event-ID: <seq> and receive all changes after that sequence.
type SSEReplay[V any] struct {
	mu      sync.Mutex
	entries []seqEntry[V]
	cap     int
	head    int // next write position
	count   int // current number of entries
	seq     atomic.Uint64
}

type seqEntry[V any] struct {
	seq   uint64
	value V
}

// NewSSEReplay creates a replay journal with the given capacity. A capacity
// of 0 defaults to 64. The journal assigns monotonically increasing sequence
// numbers starting at 1.
func NewSSEReplay[V any](capacity int) *SSEReplay[V] {
	if capacity <= 0 {
		capacity = 64
	}

	return &SSEReplay[V]{
		entries: make([]seqEntry[V], capacity),
		cap:     capacity,
	}
}

// record stores a value, assigns the next sequence number, and returns it.
// Called by Store.notifyWatchers when a replay recorder is registered.
func (r *SSEReplay[V]) record(value V) uint64 {
	seq := r.seq.Add(1)

	r.mu.Lock()
	r.entries[r.head] = seqEntry[V]{seq: seq, value: value}
	r.head = (r.head + 1) % r.cap //nolint:wsl_v5
	if r.count < r.cap {
		r.count++
	}
	r.mu.Unlock()

	return seq
}

// Replay returns all entries with sequence numbers greater than afterSeq,
// in ascending order. Returns nil if no entries match (client is up to date).
func (r *SSEReplay[V]) Replay(afterSeq uint64) []SeqValue[V] {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.count == 0 {
		return nil
	}

	var result []SeqValue[V]

	// Walk entries in insertion order. The ring buffer's oldest entry is
	// at (head - count + cap) % cap when the buffer is full, or at index 0
	// when partially filled.
	start := (r.head - r.count + r.cap) % r.cap

	for i := range r.count {
		idx := (start + i) % r.cap
		entry := r.entries[idx] //nolint:wsl_v5
		if entry.seq > afterSeq {
			result = append(result, SeqValue[V]{Seq: entry.seq, Value: entry.value})
		}
	}

	return result
}

// LatestSeq returns the highest sequence number assigned, or 0 if no values
// have been recorded.
func (r *SSEReplay[V]) LatestSeq() uint64 {
	return r.seq.Load()
}

// watcherNotification wraps a value with its sequence number for delivery
// through watcherEntry.ch when a replay recorder is attached. The Watch
// adapter goroutines check for this type and unwrap it.
type watcherNotification struct {
	seq   uint64
	value any
}

// replayRecorder is the non-generic interface that Store.notifyWatchers uses
// to record value changes. SSEReplay[V] implements this via a shim.
type replayRecorder interface {
	recordValue(value any) uint64
}

// replayShim adapts the generic SSEReplay[V] to the non-generic replayRecorder
// interface. One shim is created per Watcher.WithReplay call.
type replayShim[V any] struct {
	replay *SSEReplay[V]
}

func (s *replayShim[V]) recordValue(value any) uint64 {
	v, ok := value.(V)
	if !ok {
		return 0
	}

	return s.replay.record(v)
}

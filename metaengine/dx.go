package metaengine

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// --- Watch / Reactive reads (scaffold) ---

// watcherEntry is a registered watcher subscription stored on the Store.
// A nil key means "all keys" (collection-level); a non-nil key means
// notifications are filtered to that specific key only.
type watcherEntry struct {
	ch     chan any
	closed bool
	key    any // optional: filter to this key only (nil = all keys)
}

// Watcher provides reactive read notifications. When a value changes, all
// subscribers are notified.
type Watcher[V any] struct {
	mu      sync.Mutex
	store   *Store
	coll    string
	entries []*watcherEntry
	replay  *SSEReplay[V] // optional replay journal (nil = no reconnection)
}

// NewWatcher creates a watcher for a collection.
func NewWatcher[V any](store *Store, collection string) *Watcher[V] {
	return &Watcher[V]{store: store, coll: collection}
}

// WithReplay attaches an SSEReplay journal to the watcher, enabling
// Last-Event-ID reconnection for ServeSSE. The journal records recent value
// changes with monotonic sequence numbers. When a client reconnects with the
// Last-Event-ID header, ServeSSE replays missed values from the journal before
// switching to live streaming.
//
// Returns the replay journal so the caller can inspect it (e.g., LatestSeq).
// The journal is cleaned up when Close is called.
func (w *Watcher[V]) WithReplay(capacity int) *SSEReplay[V] {
	r := NewSSEReplay[V](capacity)

	w.mu.Lock()
	w.replay = r
	w.mu.Unlock()

	w.store.registerReplay(w.coll, &replayShim[V]{replay: r})

	return r
}

// Replay returns the replay journal, or nil if WithReplay was not called.
func (w *Watcher[V]) Replay() *SSEReplay[V] {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.replay
}

// addWatcherEntry creates a watcherEntry, registers it under the watcher's
// mutex, and subscribes it to the store for change notifications.
func (w *Watcher[V]) addWatcherEntry(key any) *watcherEntry {
	entry := &watcherEntry{ch: make(chan any, 1), key: key}

	w.mu.Lock()
	w.entries = append(w.entries, entry)
	w.mu.Unlock()

	w.store.registerWatcher(w.coll, entry)

	return entry
}

// Watch returns a channel that receives updated values. The optional key
// parameter filters notifications to that specific key only; pass nil to
// receive all changes in the collection. The channel is buffered (1) and
// drops notifications if the consumer is slow.
// Callers must close the watcher when done (via Close).
func (w *Watcher[V]) Watch(ctx context.Context, key any) <-chan V {
	ch := make(chan V, 1)

	entry := w.addWatcherEntry(key)

	// Adapter goroutine: convert any→V, unwrap watcherNotification if present.
	go func() {
		defer close(ch)

		for {
			select {
			case <-ctx.Done():
				return
			case val, ok := <-entry.ch:
				if !ok {
					return
				}

				v, ok := unwrapWatcherValue[V](val)
				if ok {
					select {
					case ch <- v:
					default: // drop if consumer is slow
					}
				}
			}
		}
	}()

	return ch
}

// WatchWithSeq is like Watch but returns SeqValue pairs (sequence number +
// value). The sequence number comes from the replay journal attached via
// WithReplay. Use this when you need the event ID for SSE Last-Event-ID
// support. If no replay journal is attached, Seq is always 0.
func (w *Watcher[V]) WatchWithSeq(ctx context.Context, key any) <-chan SeqValue[V] {
	ch := make(chan SeqValue[V], 1)

	entry := w.addWatcherEntry(key)

	// Adapter goroutine: convert any→SeqValue[V], unwrap watcherNotification.
	go func() {
		defer close(ch)

		for {
			select {
			case <-ctx.Done():
				return
			case val, ok := <-entry.ch:
				if !ok {
					return
				}

				sv, ok := unwrapWatcherSeqValue[V](val)
				if ok {
					select {
					case ch <- sv:
					default: // drop if consumer is slow
					}
				}
			}
		}
	}()

	return ch
}

// reifyWatcherValue converts a notification channel value to type V.
// It handles three cases:
//
//  1. Fast path: the value is already V (MemoryEngine, fold-produced structs).
//  2. Nil: the value is nil (delete/remove operations) — returns the zero
//     value of V so consumers receive deletion notifications instead of
//     having them silently dropped.
//  3. Reify fallback: the value is map[string]any or another engine-specific
//     type (SQLite JSON decode) — JSON round-trips to V via reify[V].
//
// Returns (zero, false) only when the value is non-nil and reify fails
// (genuinely incompatible types — should not happen in normal operation).
func reifyWatcherValue[V any](val any) (V, bool) {
	var zero V

	if val == nil {
		return zero, true
	}

	// Fast path: already the right type (common case — fold returns typed V).
	if v, ok := val.(V); ok {
		return v, true
	}

	// Fallback: reify via JSON round-trip (handles map[string]any from SQLite).
	r, err := reify[V](val)
	if err != nil {
		return zero, false
	}

	return r, true
}

// unwrapWatcherValue extracts V from a notification (raw value or
// watcherNotification wrapper). Returns (zeroV, false) on type mismatch.
func unwrapWatcherValue[V any](val any) (V, bool) {
	if notif, ok := val.(watcherNotification); ok {
		return reifyWatcherValue[V](notif.value)
	}

	return reifyWatcherValue[V](val)
}

// unwrapWatcherSeqValue extracts SeqValue[V] from a notification (raw value
// or watcherNotification wrapper). When the value arrives without a seq
// (no replay recorder), Seq is 0.
func unwrapWatcherSeqValue[V any](val any) (SeqValue[V], bool) {
	var zero SeqValue[V]

	if notif, ok := val.(watcherNotification); ok {
		v, ok := reifyWatcherValue[V](notif.value)
		if !ok {
			return zero, false
		}

		return SeqValue[V]{Seq: notif.seq, Value: v}, true
	}

	v, ok := reifyWatcherValue[V](val)
	if !ok {
		return zero, false
	}

	return SeqValue[V]{Seq: 0, Value: v}, true
}

// Close stops all subscriptions and unregisters the replay journal if attached.
func (w *Watcher[V]) Close() {
	w.mu.Lock()
	defer w.mu.Unlock()

	for _, entry := range w.entries {
		close(entry.ch)
		entry.closed = true
	}

	w.entries = nil

	if w.replay != nil {
		w.store.unregisterReplay(w.coll)
	}
}

// --- Cursor pre-fetch cache ---

// PrefetchCache caches scan results beyond the requested limit for the next
// page request. Eliminates the limit+1 round-trip pattern.
// Thread-safe: safe for concurrent use from multiple goroutines.
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

// Clear removes all cached pages.
func (c *PrefetchCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.pages = make(map[string][]any)
}

// --- TTL support ---

// TTLConfig configures time-to-live for collection entries.
type TTLConfig struct {
	DefaultTTL time.Duration
}

// WithTTL sets a TTL hint on a query configuration. The TTL value is stored
// in the query config and surfaced in the plan, but no engine currently
// enforces automatic expiration. This is advisory-only — callers that need
// TTL semantics must implement their own eviction (e.g. a background sweeper
// or lazy expiry on read). Engine-level TTL support may be added in a future
// release.
func WithTTL(d time.Duration) QueryOption {
	return func(c *QueryConfig) {
		c.TTL = d.Nanoseconds()
	}
}

// --- Typed map update ---

// MapUpdateTyped performs a typed read-modify-write on a collection entry.
// The prev value is automatically reified to type V before being passed to
// the update function, eliminating the engine-dependent `any` type footgun:
// MemoryEngine preserves Go struct types, but SQLite returns map[string]any
// from JSON. Without MapUpdateTyped, direct users of the MapUpdater interface
// must call reify[V] themselves.
//
// When the key does not exist, found is false and prev is the zero value of V.
//
// Example:
//
//	err := metaengine.MapUpdateTyped[UserView](store, ctx, "users", userID,
//	    func(prev UserView, found bool) UserView {
//	        if !found { return UserView{ID: userID, Count: 1} }
//	        prev.Count++
//	        return prev
//	    })
func MapUpdateTyped[V any](
	store *Store,
	ctx context.Context,
	collection string,
	key any,
	update func(prev V, found bool) V,
) error {
	eng, ok := store.collectionEngine(collection)
	if !ok {
		return fmt.Errorf("%w: %q", errNoQueryForInputType, collection)
	}

	if mu, ok := eng.(MapUpdater); ok {
		return mu.MapUpdate(ctx, collection, key, func(prev any) any { //nolint:wrapcheck
			var prevVal V

			found := prev != nil
			if found {
				reified, err := reify[V](prev)
				if err != nil {
					return prev
				}

				prevVal = reified
			}

			return update(prevVal, found)
		})
	}

	if mb, ok := eng.(MapBackend); ok {
		existing, found, err := mb.MapGet(ctx, collection, key)
		if err != nil {
			return fmt.Errorf("typed map get %s: %w", collection, err)
		}

		var prevVal V

		if found {
			reified, err := reify[V](existing)
			if err != nil {
				return fmt.Errorf("typed map reify %s: %w", collection, err)
			}

			prevVal = reified
		}

		updated := update(prevVal, found)

		if err := mb.MapSet(ctx, collection, key, updated); err != nil {
			return fmt.Errorf("typed map set %s: %w", collection, err)
		}

		return nil
	}

	return unsupportedEngine(errUnsupportedMapOps, eng.Profile().Name)
}

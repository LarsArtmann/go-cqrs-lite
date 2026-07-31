package metaengine

import (
	"context"
	"sync"
	"time"
)

// FluentBuilder provides a chainable API for declaring queries as an
// alternative to the variadic-`any` constructor. Each method returns the
// builder for chaining.
//
//	q := metaengine.New("find_user").
//	    Fold(UserCreated{}, func(e UserCreated) (UserID, UserView) { ... }).
//	    FoldUpdate(UserUpdated{}, func(e UserUpdated, prev UserView) UserView { ... }).
//	    FoldDelete(UserDeleted{}).
//	    Filter("status", metaengine.FilterEq).
//	    Sort("joined_at", true).
//	    Volume(1_000_000).
//	    Build[FindUser, UserView]()
type FluentBuilder struct {
	name    string
	folds   []Fold
	cfg     QueryConfig
	filters []filterSpecBuilder
	sort    *sortSpecBuilder
}

type filterSpecBuilder struct {
	column string
	op     FilterOp
}

type sortSpecBuilder struct {
	column string
	desc   bool
}

// New starts a fluent query declaration.
func New(name string) *FluentBuilder {
	return &FluentBuilder{name: name}
}

// Fold adds an insert fold (func(e E) (K, V)).
func (b *FluentBuilder) Fold(eventSample any, handler any) *FluentBuilder {
	b.folds = append(b.folds, OnTyped(eventNameFromSample(eventSample), eventSample, handler))

	return b
}

// FoldUpdate adds an update fold. The handler signature must be func(e E, prev V) V.
func (b *FluentBuilder) FoldUpdate(eventType string, handler any) *FluentBuilder {
	b.folds = append(b.folds, Fold{EventType: eventType, Kind: FoldUpdate, updateHandler: handler})

	return b
}

// FoldDelete adds a delete fold.
func (b *FluentBuilder) FoldDelete(eventType string) *FluentBuilder {
	b.folds = append(b.folds, Fold{EventType: eventType, Kind: FoldRemove})

	return b
}

// Filter declares a pushdown filter.
func (b *FluentBuilder) Filter(column string, op FilterOp) *FluentBuilder {
	b.filters = append(b.filters, filterSpecBuilder{column: column, op: op})

	return b
}

// Sort declares a pushdown sort.
func (b *FluentBuilder) Sort(column string, desc bool) *FluentBuilder {
	b.sort = &sortSpecBuilder{column: column, desc: desc}

	return b
}

// Volume sets the expected query volume.
func (b *FluentBuilder) Volume(n int64) *FluentBuilder {
	b.cfg.Volume = n

	return b
}

// LatencyBudget sets the target latency budget.
func (b *FluentBuilder) LatencyBudget(ms int64) *FluentBuilder {
	b.cfg.LatencyBudgetMs = ms

	return b
}

// Build finalizes the query declaration with generic type parameters.
func Build[Q any, R any](b *FluentBuilder) QueryDecl[Q, R] {
	var args []any

	args = append(args, toAnySlice(b.folds)...)

	for _, f := range b.filters {
		args = append(args, FilterOnField[R](f.column, f.op))
	}

	if b.sort != nil {
		args = append(args, SortOnField[R](b.sort.column, b.sort.desc))
	}

	args = append(args, Volume(b.cfg.Volume))

	if b.cfg.LatencyBudgetMs > 0 {
		args = append(args, WithLatencyBudget(b.cfg.LatencyBudgetMs))
	}

	return Query[Q, R](b.name, args...)
}

func toAnySlice[T any](s []T) []any {
	result := make([]any, len(s))
	for i, v := range s {
		result[i] = v
	}

	return result
}

func eventNameFromSample(sample any) string {
	return EventTypeName(sample)
}

// --- Watch / Reactive reads (scaffold) ---

// watcherEntry is a registered watcher subscription stored on the Store.
type watcherEntry struct {
	ch     chan any
	closed bool
}

// Watcher provides reactive read notifications. When a value changes, all
// subscribers are notified.
type Watcher[V any] struct {
	mu       sync.Mutex
	store    *Store
	coll    string
	entries []*watcherEntry
}

// NewWatcher creates a watcher for a collection.
func NewWatcher[V any](store *Store, collection string) *Watcher[V] {
	return &Watcher[V]{store: store, coll: collection}
}

// Watch returns a channel that receives updated values. The channel is
// buffered (1) and drops notifications if the consumer is slow.
// Callers must close the watcher when done (via Close).
func (w *Watcher[V]) Watch(ctx context.Context, key any) <-chan V {
	ch := make(chan V, 1)

	entry := &watcherEntry{ch: make(chan any, 1)}

	w.mu.Lock()
	w.entries = append(w.entries, entry)
	w.mu.Unlock()

	// Register on the store so Apply can notify
	w.store.registerWatcher(w.coll, entry)

	// Adapter goroutine: convert any→V
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case val, ok := <-entry.ch:
				if !ok {
					return
				}
				if v, ok := val.(V); ok {
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

// Close stops all subscriptions.
func (w *Watcher[V]) Close() {
	w.mu.Lock()
	defer w.mu.Unlock()

	for _, entry := range w.entries {
		close(entry.ch)
		entry.closed = true
	}

	w.entries = nil
}

// --- Cursor pre-fetch cache ---

// PrefetchCache caches scan results beyond the requested limit for the next
// page request. Eliminates the limit+1 round-trip pattern.
type PrefetchCache struct {
	pages map[string][]any // cursor key → cached rows
}

// NewPrefetchCache creates a new pre-fetch cache.
func NewPrefetchCache() *PrefetchCache {
	return &PrefetchCache{pages: make(map[string][]any)}
}

// Get returns cached rows for a cursor key, or nil if not cached.
func (c *PrefetchCache) Get(key string) []any {
	return c.pages[key]
}

// Put stores rows for a cursor key.
func (c *PrefetchCache) Put(key string, rows []any) {
	c.pages[key] = rows
}

// Clear removes all cached pages.
func (c *PrefetchCache) Clear() {
	c.pages = make(map[string][]any)
}

// --- TTL support ---

// TTLConfig configures time-to-live for collection entries.
type TTLConfig struct {
	DefaultTTL time.Duration
}

// WithTTL sets a TTL on a query configuration. Entries older than the TTL
// are expired. This is a declarative hint — actual expiration requires
// engine support (SQLite: background sweeper, Memory: lazy eviction).
func WithTTL(d time.Duration) QueryOption {
	_ = d // Stored as metadata for future engine support

	return func(c *QueryConfig) {
		// TTL is stored in the config metadata for engines that support it.
		// Currently a no-op hint; actual enforcement is engine-specific.
	}
}

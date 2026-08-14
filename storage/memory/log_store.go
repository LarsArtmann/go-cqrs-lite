package memory

import (
	"fmt"
	"slices"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/dispatcher/v4"
)

// LogStoreConfig parameterizes a [LogStore] with the entity-specific
// behaviors that differ between the event, command, and query stores.
//
// All fields except GetID, IsZeroID, and ClosedErr are optional;
// a nil function disables the corresponding behavior.
type LogStoreConfig[T any, ID comparable] struct {
	// GetID returns the entity's unique ID (event ID, command ID, request ID).
	GetID func(T) ID

	// IsZeroID reports whether an ID is the zero value (seek start position).
	IsZeroID func(ID) bool

	// ClosedErr is the sentinel wrapped into errors after Close.
	ClosedErr error

	// NewDupErr builds a duplicate-ID conflict error. nil disables
	// duplicate detection (the event store relies on version checks instead).
	NewDupErr func(itemID ID, suffix string) error

	// NewNotFound builds a stream-not-found rejection for scoped reads.
	// nil disables scoped reads (LoadStreamLocked returns an error).
	NewNotFound func(op, streamKey string) error

	// TrackStreams maintains the stream→indices map. Set false for
	// global-only logs (the query store has no per-stream scoping).
	TrackStreams bool
}

// LogStore is the generic append-only log core shared by the in-memory
// event, command, and query stores (WAL unification). All three are the
// same shape — append to a log, read from a log — differing only in the
// policies supplied via [LogStoreConfig].
//
// It owns the lock discipline (RWMutex + closed check), the global log,
// the per-stream index, and the ID index for position-based reads.
// Domain stores embed it and expose their typed interfaces on top.
//
// It is safe for concurrent use.
type LogStore[T any, ID comparable] struct {
	dispatcher.Lifecycle

	mu          sync.RWMutex
	log         []T
	streamIndex map[string][]int // streamKey → indices into log
	idIndex     map[ID]int       // entity ID → index into log
	cfg         LogStoreConfig[T, ID]
}

// NewLogStore creates an empty log store with the given configuration.
func NewLogStore[T any, ID comparable](cfg LogStoreConfig[T, ID]) *LogStore[T, ID] {
	return &LogStore[T, ID]{
		streamIndex: make(map[string][]int),
		idIndex:     make(map[ID]int),
		cfg:         cfg,
	}
}

// Close marks the store closed. Subsequent operations return cfg.ClosedErr.
func (s *LogStore[T, ID]) Close() error {
	return s.Lifecycle.Close()
}

// WithWrite checks the store is open, acquires the write lock, and runs fn
// under the lock.
func (s *LogStore[T, ID]) WithWrite(code, msg string, fn func() error) error {
	if err := wrapClosed(s.CheckClosed(s.cfg.ClosedErr), code, msg); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return fn()
}

// WithReadLock is the read-side companion to WithWrite. Top-level generic
// function because Go does not permit generic methods; R carries the read
// method's return type through the closure.
func WithReadLock[T any, ID comparable, R any](
	s *LogStore[T, ID],
	code, msg string,
	fn func() (R, error),
) (R, error) {
	if err := wrapClosed(s.CheckClosed(s.cfg.ClosedErr), code, msg); err != nil {
		var zero R

		return zero, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return fn()
}

// AppendLocked appends items to the log under the stream key, maintaining
// both indexes. Caller must hold the write lock (run inside WithWrite).
func (s *LogStore[T, ID]) AppendLocked(streamKey string, items []T) {
	for _, item := range items {
		idx := len(s.log)
		s.idIndex[s.cfg.GetID(item)] = idx
		s.log = append(s.log, item)

		if s.cfg.TrackStreams {
			s.streamIndex[streamKey] = append(s.streamIndex[streamKey], idx)
		}
	}
}

// StreamLenLocked returns the number of items in a stream.
// Caller must hold at least the read lock.
func (s *LogStore[T, ID]) StreamLenLocked(streamKey string) int {
	return len(s.streamIndex[streamKey])
}

// CheckDuplicateLocked returns a conflict error when itemID already exists.
// No-op when the config has no duplicate policy. Caller must hold the write
// lock so check+append is atomic.
func (s *LogStore[T, ID]) CheckDuplicateLocked(itemID ID, suffix string) error {
	if s.cfg.NewDupErr == nil {
		return nil
	}

	if _, exists := s.idIndex[itemID]; exists {
		return s.cfg.NewDupErr(itemID, suffix)
	}

	return nil
}

// LoadStreamLocked returns the items of a stream, optionally filtered.
// Returns a fresh slice. Caller must hold the read lock.
func (s *LogStore[T, ID]) LoadStreamLocked(
	op, streamKey string,
	filter func([]T) []T,
) ([]T, error) {
	if s.cfg.NewNotFound == nil {
		return nil, fmt.Errorf("memory: store has no stream scoping: %s", op)
	}

	indices, exists := s.streamIndex[streamKey]
	if !exists {
		return nil, s.cfg.NewNotFound(op, streamKey)
	}

	items := make([]T, len(indices))
	for i, idx := range indices {
		items[i] = s.log[idx]
	}

	if filter != nil {
		items = filter(items)
	}

	return items, nil
}

// ReadAllLocked returns a clone of the whole log. Caller must hold the read
// lock.
func (s *LogStore[T, ID]) ReadAllLocked() []T {
	return slices.Clone(s.log)
}

// FilterAllLocked returns a fresh slice of every log item matching pred.
// Caller must hold the read lock.
func (s *LogStore[T, ID]) FilterAllLocked(pred func(T) bool) []T {
	result := make([]T, 0, len(s.log))

	for _, item := range s.log {
		if pred(item) {
			result = append(result, item)
		}
	}

	return result
}

// ReadFromLocked returns up to limit items ordered by insertion, starting
// after afterID. When afterID is not found in the log, fromStartWhenMissing
// selects between replaying from the beginning (event-store semantics: safe
// re-replay for projections) and returning nothing (command/query journal
// semantics: an unknown position means nothing new). Caller must hold the
// read lock.
func (s *LogStore[T, ID]) ReadFromLocked(
	afterID ID,
	limit int,
	fromStartWhenMissing bool,
) []T {
	startIdx := 0

	if !s.cfg.IsZeroID(afterID) {
		idx, exists := s.idIndex[afterID]
		if !exists {
			if !fromStartWhenMissing {
				return nil
			}
		} else {
			startIdx = idx + 1
		}
	}

	if startIdx >= len(s.log) {
		return nil
	}

	filtered := s.log[startIdx:]
	if limit > 0 && len(filtered) > limit {
		filtered = filtered[:limit]
	}

	return slices.Clone(filtered)
}

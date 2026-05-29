package testhelpers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// FakeStore implements event.Store for testing.
// All methods are safe for concurrent use.
type FakeStore struct {
	mu     sync.RWMutex
	events map[string][]event.Event
	saveFn func(
		ctx context.Context,
		aggregateType event.AggregateType,
		aggregateID id.AggregateID,
		events []event.Event,
		expectedVersion event.Version,
	) error
	loadFn            func(aggregateType event.AggregateType, aggregateID id.AggregateID) ([]event.Event, error)
	loadFromVersionFn func(aggregateType event.AggregateType, aggregateID id.AggregateID, version event.Version) ([]event.Event, error)
	loadToVersionFn   func(aggregateType event.AggregateType, aggregateID id.AggregateID, maxVersion event.Version) ([]event.Event, error)
	loadToTimestampFn func(aggregateType event.AggregateType, aggregateID id.AggregateID, maxTime time.Time) ([]event.Event, error)
	appendBatchFn     func(aggregateType event.AggregateType, aggregateID id.AggregateID, events []event.Event) error
	closeFn           func() error
	readAllFn         func() ([]event.Event, error)
	readFromFn        func(afterEventID id.EventID, limit int) ([]event.Event, error)
}

// NewFakeStore creates a FakeStore with empty state.
func NewFakeStore() *FakeStore {
	return &FakeStore{events: make(map[string][]event.Event)}
}

func getOverride[T any](s *FakeStore, fn *T) T {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return *fn
}

// Save appends events to the aggregate's stream.
// SaveFn sets an optional override for Save calls.
func (s *FakeStore) Save(
	ctx context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	events []event.Event,
	expectedVersion event.Version,
) error {
	if fn := getOverride(s, &s.saveFn); fn != nil {
		return fn(ctx, aggregateType, aggregateID, events, expectedVersion)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := event.StreamKey(aggregateType, aggregateID)
	s.events[key] = append(s.events[key], events...)

	return nil
}

// AppendBatch appends events without concurrency checks.
func (s *FakeStore) AppendBatch(
	_ context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	events []event.Event,
) error {
	if fn := getOverride(s, &s.appendBatchFn); fn != nil {
		return fn(aggregateType, aggregateID, events)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := event.StreamKey(aggregateType, aggregateID)
	s.events[key] = append(s.events[key], events...)

	return nil
}

// Load returns all events for an aggregate.
func (s *FakeStore) Load(
	_ context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
) ([]event.Event, error) {
	if fn := getOverride(s, &s.loadFn); fn != nil {
		return fn(aggregateType, aggregateID)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	key := event.StreamKey(aggregateType, aggregateID)

	return append([]event.Event{}, s.events[key]...), nil
}

// LoadFromVersion returns events starting after the given version.
func (s *FakeStore) LoadFromVersion(
	_ context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	version event.Version,
) ([]event.Event, error) {
	if fn := getOverride(s, &s.loadFromVersionFn); fn != nil {
		return fn(aggregateType, aggregateID, version)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	key := event.StreamKey(aggregateType, aggregateID)
	all := s.events[key]

	for i, evt := range all {
		if evt.Version() > version {
			return append([]event.Event{}, all[i:]...), nil
		}
	}

	return nil, nil
}

// LoadToVersion returns events up to and including maxVersion.
func (s *FakeStore) LoadToVersion(
	_ context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	maxVersion event.Version,
) ([]event.Event, error) {
	if fn := getOverride(s, &s.loadToVersionFn); fn != nil {
		return fn(aggregateType, aggregateID, maxVersion)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	key := event.StreamKey(aggregateType, aggregateID)
	all := s.events[key]

	end := min(maxVersion.Int(), len(all))

	return all[:end], nil
}

// LoadToTimestamp returns events where OccurredAt <= maxTime.
func (s *FakeStore) LoadToTimestamp(
	_ context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	maxTime time.Time,
) ([]event.Event, error) {
	if fn := getOverride(s, &s.loadToTimestampFn); fn != nil {
		return fn(aggregateType, aggregateID, maxTime)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	key := event.StreamKey(aggregateType, aggregateID)
	all := s.events[key]

	var filtered []event.Event

	for _, e := range all {
		if !e.OccurredAt().After(maxTime) {
			filtered = append(filtered, e)
		}
	}

	return filtered, nil
}

// ReadAll returns all events across all aggregates.
func (s *FakeStore) ReadAll(_ context.Context) ([]event.Event, error) {
	if fn := getOverride(s, &s.readAllFn); fn != nil {
		return fn()
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var all []event.Event

	for _, evts := range s.events {
		all = append(all, evts...)
	}

	return all, nil
}

// ReadFrom returns events starting after the given event ID.
func (s *FakeStore) ReadFrom(
	ctx context.Context,
	afterEventID id.EventID,
	limit int,
) ([]event.Event, error) {
	if fn := getOverride(s, &s.readFromFn); fn != nil {
		return fn(afterEventID, limit)
	}

	all, err := s.ReadAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("read all for ReadFrom (limit=%d, after=%s): %w", limit, afterEventID, err)
	}

	startIdx := 0

	if !afterEventID.IsZero() {
		for i, e := range all {
			if e.ID() == afterEventID {
				startIdx = i + 1

				break
			}
		}
	}

	filtered := all[startIdx:]
	if limit > 0 && len(filtered) > limit {
		filtered = filtered[:limit]
	}

	return filtered, nil
}

// Close is a no-op for testing.
func (s *FakeStore) Close() error {
	if fn := getOverride(s, &s.closeFn); fn != nil {
		return fn()
	}

	return nil
}

// LoadFn sets an optional override for Load calls.
// Return an error to simulate load failures.

// LoadFromVersionFn sets an optional override for LoadFromVersion calls.
// Return an error to simulate load-from-version failures.

var (
	_ event.Store           = (*FakeStore)(nil)
	_ event.Journal         = (*FakeStore)(nil)
	_ event.SeekableJournal = (*FakeStore)(nil)
)

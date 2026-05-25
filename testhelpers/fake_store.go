package testhelpers

import (
	"context"
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
	loadFromVersionFn  func(aggregateType event.AggregateType, aggregateID id.AggregateID, version event.Version) ([]event.Event, error)
	loadToVersionFn   func(aggregateType event.AggregateType, aggregateID id.AggregateID, maxVersion event.Version) ([]event.Event, error)
	loadToTimestampFn func(aggregateType event.AggregateType, aggregateID id.AggregateID, maxTime time.Time) ([]event.Event, error)
	appendBatchFn     func(aggregateType event.AggregateType, aggregateID id.AggregateID, events []event.Event) error
	deleteFn          func(aggregateType event.AggregateType, aggregateID id.AggregateID) error
	closeFn           func() error
}

// NewFakeStore creates a FakeStore with empty state.
func NewFakeStore() *FakeStore {
	return &FakeStore{events: make(map[string][]event.Event)}
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
	s.mu.RLock()
	fn := s.saveFn
	s.mu.RUnlock()

	if fn != nil {
		return fn(ctx, aggregateType, aggregateID, events, expectedVersion)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := fakeStreamKey(aggregateType, aggregateID)
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
	s.mu.RLock()
	fn := s.appendBatchFn
	s.mu.RUnlock()

	if fn != nil {
		return fn(aggregateType, aggregateID, events)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := fakeStreamKey(aggregateType, aggregateID)
	s.events[key] = append(s.events[key], events...)

	return nil
}

// Load returns all events for an aggregate.
func (s *FakeStore) Load(
	_ context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
) ([]event.Event, error) {
	s.mu.RLock()
	fn := s.loadFn
	s.mu.RUnlock()

	if fn != nil {
		return fn(aggregateType, aggregateID)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	key := fakeStreamKey(aggregateType, aggregateID)

	return append([]event.Event{}, s.events[key]...), nil
}

// LoadFromVersion returns events starting after the given version.
func (s *FakeStore) LoadFromVersion(
	_ context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	version event.Version,
) ([]event.Event, error) {
	s.mu.RLock()
	fn := s.loadFromVersionFn
	s.mu.RUnlock()

	if fn != nil {
		return fn(aggregateType, aggregateID, version)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	key := fakeStreamKey(aggregateType, aggregateID)
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
	s.mu.RLock()
	fn := s.loadToVersionFn
	s.mu.RUnlock()

	if fn != nil {
		return fn(aggregateType, aggregateID, maxVersion)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	key := fakeStreamKey(aggregateType, aggregateID)
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
	s.mu.RLock()
	fn := s.loadToTimestampFn
	s.mu.RUnlock()

	if fn != nil {
		return fn(aggregateType, aggregateID, maxTime)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	key := fakeStreamKey(aggregateType, aggregateID)
	all := s.events[key]

	var filtered []event.Event

	for _, e := range all {
		if !e.OccurredAt().After(maxTime) {
			filtered = append(filtered, e)
		}
	}

	return filtered, nil
}

// Delete removes all events for an aggregate.
func (s *FakeStore) Delete(
	_ context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
) error {
	s.mu.RLock()
	fn := s.deleteFn
	s.mu.RUnlock()

	if fn != nil {
		return fn(aggregateType, aggregateID)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := fakeStreamKey(aggregateType, aggregateID)
	delete(s.events, key)

	return nil
}

// Close is a no-op for testing.
func (s *FakeStore) Close() error {
	s.mu.RLock()
	fn := s.closeFn
	s.mu.RUnlock()

	if fn != nil {
		return fn()
	}

	return nil
}

// LoadFn sets an optional override for Load calls.
// Return an error to simulate load failures.

// LoadFromVersionFn sets an optional override for LoadFromVersion calls.
// Return an error to simulate load-from-version failures.

// DeleteFn sets an optional override for Delete calls.
// Return an error to simulate delete failures.

var _ event.Store = (*FakeStore)(nil)

func fakeStreamKey(aggregateType event.AggregateType, aggregateID id.AggregateID) string {
	return string(aggregateType) + ":" + aggregateID.String()
}

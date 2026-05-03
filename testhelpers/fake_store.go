package testhelpers

import (
	"context"
	"sync"

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
}

// NewFakeStore creates a FakeStore with empty state.
func NewFakeStore() *FakeStore {
	return &FakeStore{events: make(map[string][]event.Event)}
}

// SaveFn sets an optional override for Save calls.
// Return an error to simulate store failures.
func (s *FakeStore) SaveFn(
	fn func(
		ctx context.Context,
		aggregateType event.AggregateType,
		aggregateID id.AggregateID,
		events []event.Event,
		expectedVersion event.Version,
	) error,
) *FakeStore {
	s.saveFn = fn

	return s
}

// Save appends events to the aggregate's stream.
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
	defer s.mu.RUnlock()

	key := fakeStreamKey(aggregateType, aggregateID)

	return s.events[key], nil
}

// LoadFromVersion returns events starting after the given version.
func (s *FakeStore) LoadFromVersion(
	_ context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	version event.Version,
) ([]event.Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := fakeStreamKey(aggregateType, aggregateID)
	all := s.events[key]

	for i, evt := range all {
		if evt.Version() > version {
			return all[i:], nil
		}
	}

	return nil, nil
}

// Delete removes all events for an aggregate.
func (s *FakeStore) Delete(
	_ context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := fakeStreamKey(aggregateType, aggregateID)
	delete(s.events, key)

	return nil
}

// Close is a no-op for testing.
func (s *FakeStore) Close() error { return nil }

var _ event.Store = (*FakeStore)(nil)

func fakeStreamKey(aggregateType event.AggregateType, aggregateID id.AggregateID) string {
	return string(aggregateType) + ":" + aggregateID.String()
}

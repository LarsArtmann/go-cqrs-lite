package testhelpers

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

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

func (s *FakeStore) LoadFn(
	fn func(aggregateType event.AggregateType, aggregateID id.AggregateID) ([]event.Event, error),
) *FakeStore {
	s.loadFn = fn

	return s
}

func (s *FakeStore) LoadFromVersionFn(
	fn func(aggregateType event.AggregateType, aggregateID id.AggregateID, version event.Version) ([]event.Event, error),
) *FakeStore {
	s.loadFromVersionFn = fn

	return s
}

func (s *FakeStore) DeleteFn(
	fn func(aggregateType event.AggregateType, aggregateID id.AggregateID) error,
) *FakeStore {
	s.deleteFn = fn

	return s
}

// CloseFn sets an optional override for Close calls.
func (s *FakeStore) CloseFn(fn func() error) *FakeStore {
	s.closeFn = fn

	return s
}

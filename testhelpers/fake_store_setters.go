package testhelpers

import (
	"time"

	"github.com/larsartmann/go-cqrs-lite/event"
)

// SaveFn sets an optional override for Save calls.
// Return an error to simulate store failures.
func (s *FakeStore) SaveFn(fn event.SaveFunc) *FakeStore {
	s.saveFn = fn

	return s
}

func (s *FakeStore) LoadFn(
	fn func(ref event.AggregateRef) ([]event.Event, error),
) *FakeStore {
	s.loadFn = fn

	return s
}

func (s *FakeStore) LoadFromVersionFn(
	fn func(ref event.AggregateRef, version event.Version) ([]event.Event, error),
) *FakeStore {
	s.loadFromVersionFn = fn

	return s
}

// CloseFn sets an optional override for Close calls.
func (s *FakeStore) CloseFn(fn func() error) *FakeStore {
	s.closeFn = fn

	return s
}

func (s *FakeStore) AppendBatchFn(
	fn func(ref event.AggregateRef, events []event.Event) error,
) *FakeStore {
	s.appendBatchFn = fn

	return s
}

func (s *FakeStore) LoadToVersionFn(
	fn func(ref event.AggregateRef, maxVersion event.Version) ([]event.Event, error),
) *FakeStore {
	s.loadToVersionFn = fn

	return s
}

func (s *FakeStore) LoadToTimestampFn(
	fn func(ref event.AggregateRef, maxTime time.Time) ([]event.Event, error),
) *FakeStore {
	s.loadToTimestampFn = fn

	return s
}

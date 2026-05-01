package testhelpers

import (
	"context"
	"fmt"
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
	if s.saveFn != nil {
		return s.saveFn(ctx, aggregateType, aggregateID, events, expectedVersion)
	}

	s.mu.Lock()

	key := string(aggregateType) + ":" + aggregateID.String()
	s.events[key] = append(s.events[key], events...)

	s.mu.Unlock()

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

	key := string(aggregateType) + ":" + aggregateID.String()
	s.events[key] = append(s.events[key], events...)

	s.mu.Unlock()

	return nil
}

// Load returns all events for an aggregate.
func (s *FakeStore) Load(
	_ context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
) ([]event.Event, error) {
	s.mu.RLock()

	key := string(aggregateType) + ":" + aggregateID.String()
	evts := s.events[key]

	s.mu.RUnlock()

	return evts, nil
}

// LoadFromVersion returns events starting after the given version.
func (s *FakeStore) LoadFromVersion(
	_ context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	version event.Version,
) ([]event.Event, error) {
	s.mu.RLock()

	key := string(aggregateType) + ":" + aggregateID.String()
	all := s.events[key]

	s.mu.RUnlock()

	for i, evt := range all {
		if evt.Version() > version.Int() {
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

	key := string(aggregateType) + ":" + aggregateID.String()
	delete(s.events, key)

	s.mu.Unlock()

	return nil
}

var _ event.Store = (*FakeStore)(nil)

// FakeBus implements event.Bus for testing.
type FakeBus struct {
	mu         sync.Mutex
	Published  []event.Event
	PublishErr error
}

// NewFakeBus creates a FakeBus with no published events.
func NewFakeBus() *FakeBus {
	return &FakeBus{}
}

// Publish appends events or returns PublishErr if set.
func (b *FakeBus) Publish(_ context.Context, events ...event.Event) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.PublishErr != nil {
		return b.PublishErr
	}

	b.Published = append(b.Published, events...)

	return nil
}

// Subscribe is a no-op for testing.
func (b *FakeBus) Subscribe(_ event.Type, _ event.Handler) error { return nil }

// SubscribeAll is a no-op for testing.
func (b *FakeBus) SubscribeAll(_ event.Handler) error { return nil }

var _ event.Bus = (*FakeBus)(nil)

// FakeSnapshotStore implements event.SnapshotStore for testing.
type FakeSnapshotStore struct {
	mu       sync.RWMutex
	snapshot *event.Snapshot
	saved    []event.Snapshot
	loadErr  error
	saveErr  error
}

// NewFakeStore creates a FakeSnapshotStore with no snapshot.
func NewFakeSnapshotStore() *FakeSnapshotStore {
	return &FakeSnapshotStore{}
}

// SetSnapshot configures the snapshot returned by Load.
func (s *FakeSnapshotStore) SetSnapshot(snap *event.Snapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.snapshot = snap
}

// SetLoadError configures an error returned by Load.
func (s *FakeSnapshotStore) SetLoadError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.loadErr = err
}

// SetSaveError configures an error returned by Save.
func (s *FakeSnapshotStore) SetSaveError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.saveErr = err
}

// Saved returns a copy of all snapshots saved via Save.
func (s *FakeSnapshotStore) Saved() []event.Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return append([]event.Snapshot{}, s.saved...)
}

// Save records the snapshot for later verification.
func (s *FakeSnapshotStore) Save(_ context.Context, snap event.Snapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.saveErr != nil {
		return s.saveErr
	}

	s.saved = append(s.saved, snap)

	return nil
}

// Load returns the configured snapshot or error.
func (s *FakeSnapshotStore) Load(
	_ context.Context,
	_ event.AggregateType,
	_ id.AggregateID,
) (*event.Snapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.snapshot, s.loadErr
}

// LoadAtVersion returns the configured snapshot or error.
func (s *FakeSnapshotStore) LoadAtVersion(
	_ context.Context,
	_ event.AggregateType,
	_ id.AggregateID,
	_ event.Version,
) (*event.Snapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.snapshot, s.loadErr
}

// Delete is a no-op for testing.
func (s *FakeSnapshotStore) Delete(
	_ context.Context,
	_ event.AggregateType,
	_ id.AggregateID,
) error {
	return nil
}

var _ event.SnapshotStore = (*FakeSnapshotStore)(nil)

// FakeOutbox implements event.Outbox for testing.
type FakeOutbox struct {
	mu       sync.Mutex
	Entries  []event.OutboxEntry
	appendFn func(events []event.Event) error
}

// NewFakeOutbox creates a FakeOutbox with no entries.
func NewFakeOutbox() *FakeOutbox {
	return &FakeOutbox{}
}

// AppendFn sets an optional override for Append calls.
func (o *FakeOutbox) AppendFn(fn func(events []event.Event) error) *FakeOutbox {
	o.appendFn = fn

	return o
}

// Append writes events to the outbox.
func (o *FakeOutbox) Append(_ context.Context, events []event.Event) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.appendFn != nil {
		return o.appendFn(events)
	}

	o.Entries = append(o.Entries, event.OutboxEntry{
		ID:     event.OutboxID(fmt.Sprintf("outbox-%d", len(o.Entries))),
		Events: events,
	})

	return nil
}

// PollPending returns all entries.
func (o *FakeOutbox) PollPending(_ context.Context, _ int) ([]event.OutboxEntry, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	return o.Entries, nil
}

// Ack clears all entries.
func (o *FakeOutbox) Ack(_ context.Context, _ []event.OutboxID) error {
	o.mu.Lock()
	o.Entries = nil
	o.mu.Unlock()

	return nil
}

var _ event.Outbox = (*FakeOutbox)(nil)

// FakeCheckpointStore implements event.CheckpointStore for testing.
// All operations are no-ops.
type FakeCheckpointStore struct{}

// Load returns a zero EventID (no-op).
func (FakeCheckpointStore) Load(_ context.Context, _ string) (id.EventID, error) {
	return id.EventID{}, nil
}

// Save does nothing (no-op).
func (FakeCheckpointStore) Save(_ context.Context, _ string, _ id.EventID) error {
	return nil
}

var _ event.CheckpointStore = FakeCheckpointStore{}

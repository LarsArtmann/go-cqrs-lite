package aggregate

import (
	"context"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// Repository loads and saves aggregate roots.
type Repository interface {
	// Save persists uncommitted events from the aggregate.
	Save(ctx context.Context, root Root) error

	// Load replays event history into the provided aggregate.
	// The aggregate must have its ID and Type set (via its constructor).
	Load(ctx context.Context, root Root) error
}

// SnapshotStrategy decides when to create a snapshot after saving events.
type SnapshotStrategy interface {
	// ShouldSnapshot returns true if a snapshot should be created
	// for the given aggregate after it has reached the given version.
	ShouldSnapshot(aggregateType event.AggregateType, version int) bool
}

// EveryNEvents creates a SnapshotStrategy that snapshots every N events.
func EveryNEvents(n int) SnapshotStrategy {
	return &everyN{interval: n}
}

type everyN struct{ interval int }

func (s *everyN) ShouldSnapshot(_ event.AggregateType, version int) bool {
	return version > 0 && version%s.interval == 0
}

// EventSourcedRepository persists and loads aggregates using event sourcing.
type EventSourcedRepository struct {
	store            event.Store
	bus              event.Bus
	snapshotStore    event.SnapshotStore
	outbox           event.Outbox
	codec            event.Codec
	snapshotStrategy SnapshotStrategy
}

var _ Repository = (*EventSourcedRepository)(nil)

// RepositoryOption configures an EventSourcedRepository.
type RepositoryOption func(*EventSourcedRepository)

// WithSnapshotStore enables snapshot support for the repository.
func WithSnapshotStore(store event.SnapshotStore) RepositoryOption {
	return func(r *EventSourcedRepository) {
		r.snapshotStore = store
	}
}

// WithOutbox enables outbox support for reliable event publishing.
// When configured, Save appends events to the outbox instead of
// publishing directly to the bus. The caller must run an OutboxPublisher
// background process to drain the outbox.
func WithOutbox(outbox event.Outbox) RepositoryOption {
	return func(r *EventSourcedRepository) {
		r.outbox = outbox
	}
}

// WithCodec sets the codec for snapshot serialization.
// When set, Save encodes snapshot state via the codec instead of
// relying on the aggregate to serialize itself. Load decodes via
// the codec before calling ApplySnapshot.
func WithCodec(codec event.Codec) RepositoryOption {
	return func(r *EventSourcedRepository) {
		r.codec = codec
	}
}

// WithSnapshotStrategy sets the strategy for automatic snapshotting.
// When set, Save checks the strategy after persisting events and
// creates a snapshot if the strategy triggers.
func WithSnapshotStrategy(strategy SnapshotStrategy) RepositoryOption {
	return func(r *EventSourcedRepository) {
		r.snapshotStrategy = strategy
	}
}

// NewRepository creates a new event-sourced repository.
func NewRepository(
	store event.Store,
	bus event.Bus,
	opts ...RepositoryOption,
) *EventSourcedRepository {
	r := &EventSourcedRepository{ //nolint:exhaustruct // options fill remaining fields
		store: store,
		bus:   bus,
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// opError formats an error for aggregate operations.
func opError(
	op string,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	err error,
) error {
	return fmt.Errorf("%s for %s %s: %w", op, aggregateType, aggregateID, err)
}

// Save persists uncommitted events. If an outbox is configured, events are
// appended to the outbox for reliable eventual publishing. Otherwise, they
// are published directly to the bus.
func (r *EventSourcedRepository) Save(ctx context.Context, root Root) error {
	changes := root.UncommittedChanges()
	if len(changes) == 0 {
		return nil
	}

	aggregateID := root.ID()
	aggregateType := root.Type()

	expectedVersion := event.Version(root.Version() - len(changes))

	err := r.store.Save(ctx, aggregateType, aggregateID, changes, expectedVersion)
	if err != nil {
		return opError("save", aggregateType, aggregateID, err)
	}

	if r.outbox != nil {
		err = r.outbox.Append(ctx, changes)
		if err != nil {
			return opError("stage events in outbox", aggregateType, aggregateID, err)
		}
	} else {
		err = r.bus.Publish(ctx, changes...)
		if err != nil {
			return opError("publish events", aggregateType, aggregateID, err)
		}
	}

	root.MarkChangesAsCommitted()

	if r.shouldSnapshot(root) {
		err = r.saveSnapshot(ctx, root)
		if err != nil {
			return opError("save snapshot", aggregateType, aggregateID, err)
		}
	}

	return nil
}

// Load replays event history into the aggregate.
// If a snapshot store is configured, it loads the latest snapshot first,
// sets the aggregate version, then replays events from the snapshot version onward.
func (r *EventSourcedRepository) Load(ctx context.Context, root Root) error {
	aggregateID := root.ID()
	aggregateType := root.Type()

	events, err := r.loadEvents(ctx, root, aggregateType, aggregateID)
	if err != nil {
		return err
	}

	err = root.LoadEvents(events)
	if err != nil {
		return fmt.Errorf(
			"replay %d events for %s %s: %w",
			len(events),
			aggregateType,
			aggregateID,
			err,
		)
	}

	return nil
}

// loadEvents returns events for the aggregate, using a snapshot if available.
func (r *EventSourcedRepository) loadEvents(
	ctx context.Context,
	root Root,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
) ([]event.Event, error) {
	if r.snapshotStore == nil {
		return r.loadFromStore(ctx, aggregateType, aggregateID)
	}

	snapshot, snapErr := r.snapshotStore.Load(ctx, aggregateType, aggregateID)
	if snapErr != nil || snapshot == nil {
		return r.loadFromStore(ctx, aggregateType, aggregateID)
	}

	root.SetVersion(snapshot.Version)

	err := root.ApplySnapshot(snapshot.State)
	if err != nil {
		return nil, opError("apply snapshot", aggregateType, aggregateID, err)
	}

	events, err := r.store.LoadFromVersion(ctx, aggregateType, aggregateID, snapshot.Version)
	if err != nil {
		return nil, fmt.Errorf(
			"load events from version %d for %s %s: %w",
			snapshot.Version,
			aggregateType,
			aggregateID,
			err,
		)
	}

	return events, nil
}

func (r *EventSourcedRepository) loadFromStore(
	ctx context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
) ([]event.Event, error) {
	events, err := r.store.Load(ctx, aggregateType, aggregateID)
	if err != nil {
		return nil, opError("load events", aggregateType, aggregateID, err)
	}

	return events, nil
}

func (r *EventSourcedRepository) shouldSnapshot(root Root) bool {
	return r.snapshotStrategy != nil &&
		r.snapshotStore != nil &&
		r.snapshotStrategy.ShouldSnapshot(root.Type(), root.Version())
}

func (r *EventSourcedRepository) saveSnapshot(ctx context.Context, root Root) error {
	var state []byte

	if r.codec != nil {
		encoded, err := r.codec.Encode(root)
		if err != nil {
			return fmt.Errorf("encode snapshot state: %w", err)
		}

		state = encoded
	}

	err := r.snapshotStore.Save(ctx, event.Snapshot{
		AggregateID:   root.ID(),
		AggregateType: root.Type(),
		Version:       event.Version(root.Version()),
		State:         state,
		CreatedAt:     time.Now().UTC(),
	})
	if err != nil {
		return fmt.Errorf("snapshot store save: %w", err)
	}

	return nil
}

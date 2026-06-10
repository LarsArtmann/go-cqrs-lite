package pebble

import (
	"context"
	"fmt"
	"hash/fnv"
	"log/slog"
	"sync"

	"github.com/cockroachdb/pebble"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

const lockShardCount = 256

// EventStore implements go-cqrs-lite/event.Store using Pebble.
//
// Save uses per-aggregate locking to prevent concurrent writes from silently
// overwriting each other (Pebble batch commits are atomic, but two goroutines
// can both pass checkVersion before either commits). A fixed-size sharded
// mutex pool avoids unbounded memory growth from a sync.Map.
type EventStore struct {
	db            *pebble.DB
	logger        *slog.Logger
	prefix        string
	journalPrefix string
	lockShards    [lockShardCount]sync.Mutex
	syncWrites    bool
}

// StoreOption configures a EventStore.
type StoreOption func(*EventStore)

// WithAsyncWrites disables sync writes for higher throughput at the cost of
// durability guarantees. Use only when data loss on crash is acceptable
// (e.g., caches, replay-able projections).
func WithAsyncWrites() StoreOption {
	return func(s *EventStore) { s.syncWrites = false }
}

// NewStore creates a new store using an existing Pebble DB.
// Panics if db is nil.
func NewStore(database *pebble.DB, logger *slog.Logger, opts ...StoreOption) *EventStore {
	if database == nil {
		panic("pebble: NewStore called with nil db")
	}

	s := &EventStore{ //nolint:exhaustruct // locks initialized lazily
		db:            database,
		logger:        logger,
		prefix:        "cqrs_event:",
		journalPrefix: "cqrs_journal:",
		syncWrites:    true,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// eventKey generates a storage key for an event.
// Pattern: cqrs_event:{ref.Type}:{ref.ID}:{version}.
func (a *EventStore) eventKey(
	ref event.AggregateRef,
	version event.Version,
) []byte {
	return fmt.Appendf(nil, "%s%s:%s:%010d", a.prefix, ref.Type, ref.ID, version.Int())
}

// aggregatePrefix returns the prefix for all events of an aggregate.
func (a *EventStore) aggregatePrefix(
	ref event.AggregateRef,
) []byte {
	return fmt.Appendf(nil, "%s%s:%s:", a.prefix, ref.Type, ref.ID)
}

// aggregateUpperBound returns the exclusive upper bound for all events of an aggregate.
// Pairs with aggregatePrefix to form a complete key range for NewIter:
// iter := db.NewIter(&pebble.IterOptions{LowerBound: prefix, UpperBound: upperBound}).
// The trailing 0xff byte sorts after any event version (eventKey uses %010d, max 10 digits).
func (a *EventStore) aggregateUpperBound(
	ref event.AggregateRef,
) []byte {
	return fmt.Appendf(nil, "%s%s:%s:\xff", a.prefix, ref.Type, ref.ID)
}

// Save implements event.Store.Save with per-aggregate locking for concurrency safety.
func (a *EventStore) Save(
	_ context.Context,
	ref event.AggregateRef,
	events []event.Event,
	expectedVersion event.Version,
) error {
	if len(events) == 0 {
		return nil
	}

	a.lockAggregate(ref)
	defer a.unlockAggregate(ref)

	err := a.checkVersion(ref, expectedVersion)
	if err != nil {
		return event.WrapInfrastructure(err, "pebble.check_version",
			fmt.Sprintf("pebble check version for %s %s", ref.Type, ref.ID))
	}

	batch := a.db.NewBatch()

	defer func() { _ = batch.Close() }()

	err = a.writeEventsToBatch(
		batch, ref, events, expectedVersion,
	)
	if err != nil {
		return event.WrapInfrastructure(
			err,
			"pebble.write_events",
			fmt.Sprintf(
				"pebble write %d events for %s %s",
				len(events),
				ref.Type,
				ref.ID,
			),
		)
	}

	return a.commitAndLog(batch, "events saved", ref, len(events))
}

func (a *EventStore) lockShard(ref event.AggregateRef) *sync.Mutex {
	h := fnv.New32a()
	_, _ = h.Write([]byte(ref.Type))
	_, _ = h.Write([]byte(ref.ID.String()))

	return &a.lockShards[h.Sum32()%lockShardCount]
}

func (a *EventStore) lockAggregate(ref event.AggregateRef) { a.lockShard(ref).Lock() }

func (a *EventStore) unlockAggregate(ref event.AggregateRef) { a.lockShard(ref).Unlock() }

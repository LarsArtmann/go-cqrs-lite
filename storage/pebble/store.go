package pebble

import (
	"context"
	"fmt"
	"hash/fnv"
	"log/slog"
	"sync"

	"github.com/cockroachdb/pebble"
	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
)

const lockShardCount = 256

// EventStore implements go-cqrs-lite/event.Store using Pebble.
//
// Save uses per-aggregate locking to prevent concurrent writes from silently
// overwriting each other (Pebble batch commits are atomic, but two goroutines
// can both pass checkVersion before either commits). A fixed-size sharded
// mutex pool avoids unbounded memory growth from a sync.Map.
type EventStore struct {
	storeBase

	journalPrefix string
	lockShards    [lockShardCount]sync.Mutex
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
// Returns ErrNilDatabase if db is nil.
func NewStore(database *pebble.DB, logger *slog.Logger, opts ...StoreOption) (*EventStore, error) {
	if database == nil {
		return nil, ErrNilDatabase
	}

	s := &EventStore{
		storeBase: storeBase{
			db:         database,
			logger:     logger,
			prefix:     "cqrs_event:",
			syncWrites: true,
		},
		journalPrefix: "cqrs_journal:",
	}

	for _, opt := range opts {
		opt(s)
	}

	return s, nil
}

// eventKey generates a storage key for an event.
// Pattern: cqrs_event:{ref.Type}:{ref.ID}:{version}.
func (a *EventStore) eventKey(
	ref id.AggregateRef,
	version event.Version,
) []byte {
	return fmt.Appendf(nil, "%s%s:%s:%010d", a.prefix, ref.Type, ref.ID, version.Int())
}

// aggregatePrefix returns the prefix for all events of an aggregate.
func (a *EventStore) aggregatePrefix(
	ref id.AggregateRef,
) []byte {
	return fmt.Appendf(nil, "%s%s:%s:", a.prefix, ref.Type, ref.ID)
}

// aggregateUpperBound returns the exclusive upper bound for all events of an aggregate.
// Pairs with aggregatePrefix to form a complete key range for NewIter:
// iter := db.NewIter(&pebble.IterOptions{LowerBound: prefix, UpperBound: upperBound}).
// The trailing 0xff byte sorts after any event version (eventKey uses %010d, max 10 digits).
func (a *EventStore) aggregateUpperBound(
	ref id.AggregateRef,
) []byte {
	return fmt.Appendf(nil, "%s%s:%s:\xff", a.prefix, ref.Type, ref.ID)
}

// startLoadSpan creates an aggregate span and returns the key range for a
// full aggregate scan. The caller must defer span.End().
func (a *EventStore) startLoadSpan(
	ctx context.Context,
	spanName string,
	ref id.AggregateRef,
) (cqrsotel.Span, []byte, []byte) {
	_, span := startAggregateSpan(ctx, spanName, ref)
	return span, a.aggregatePrefix(ref), a.aggregateUpperBound(ref)
}

// startLoadFromVersionSpan creates an aggregate span with a version attribute
// and returns the key range starting from version+1. The caller must defer
// span.End().
func (a *EventStore) startLoadFromVersionSpan(
	ctx context.Context,
	spanName string,
	ref id.AggregateRef,
	version event.Version,
) (cqrsotel.Span, []byte, []byte) {
	_, span := startAggregateSpan(ctx, spanName, ref,
		cqrsotel.AttrInt(cqrsotel.AttrAggregateVersion, version.Int()))
	return span, a.eventKey(ref, version+1), a.aggregateUpperBound(ref)
}

// journalBounds returns the lower/upper key range for scanning the entire
// global journal (all events ordered by occurrence time).
func (a *EventStore) journalBounds() ([]byte, []byte) {
	return []byte(a.journalPrefix), []byte(a.journalPrefix + "\xff")
}

// Save implements event.Store.Save with per-aggregate locking for concurrency safety.
func (a *EventStore) Save(
	ctx context.Context,
	ref id.AggregateRef,
	events []event.Event,
	expectedVersion event.Version,
) error {
	if len(events) == 0 {
		return nil
	}

	_, span := startAggregateSpan(ctx, "pebble.event.save", ref,
		cqrsotel.AttrInt("event.count", len(events)),
		cqrsotel.AttrInt(cqrsotel.AttrAggregateVersion, expectedVersion.Int()))
	defer span.End()

	a.lockAggregate(ref)
	defer a.unlockAggregate(ref)

	err := a.checkVersion(ref, expectedVersion)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return errorfamily.WrapInfrastructure(err, "pebble.check_version",
			fmt.Sprintf("pebble check version for %s %s", ref.Type, ref.ID))
	}

	batch := a.db.NewBatch()

	defer func() { _ = batch.Close() }()

	err = a.writeEventsToBatch(
		batch, ref, events, expectedVersion,
	)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return errorfamily.WrapInfrastructure(
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

	err = a.commitAndLog(batch, "events saved", ref, len(events))
	if err != nil {
		cqrsotel.RecordError(span, err)

		return err
	}

	return nil
}

func (a *EventStore) lockShard(ref id.AggregateRef) *sync.Mutex {
	h := fnv.New32a()
	_, _ = h.Write([]byte(ref.Type))
	_, _ = h.Write([]byte(ref.ID.String()))

	return &a.lockShards[h.Sum32()%lockShardCount]
}

func (a *EventStore) lockAggregate(ref id.AggregateRef) { a.lockShard(ref).Lock() }

func (a *EventStore) unlockAggregate(ref id.AggregateRef) { a.lockShard(ref).Unlock() }

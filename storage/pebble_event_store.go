package storage

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/cockroachdb/pebble"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// PebbleEventStore implements go-cqrs-lite/event.Store using Pebble.
//
// Save uses per-aggregate locking to prevent concurrent writes from silently
// overwriting each other (Pebble batch commits are atomic, but two goroutines
// can both pass checkVersion before either commits). The lock map grows with
// the number of unique aggregates — bounded by actual data volume for an
// embedded single-process store.
type PebbleEventStore struct {
	db     *pebble.DB
	logger *slog.Logger
	prefix string
	locks  sync.Map // map[string]*sync.Mutex — one per aggregate
}

// NewPebbleStore creates a new store using an existing Pebble DB.
func NewPebbleStore(db *pebble.DB, logger *slog.Logger) *PebbleEventStore {
	return &PebbleEventStore{
		db:     db,
		logger: logger,
		prefix: "cqrs_event:",
	}
}

// eventKey generates a storage key for an event.
// Pattern: cqrs_event:{aggregateType}:{aggregateID}:{version}.
func (a *PebbleEventStore) eventKey(
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	version event.Version,
) []byte {
	return fmt.Appendf(nil, "%s%s:%s:%010d", a.prefix, aggregateType, aggregateID, version.Int())
}

// aggregatePrefix returns the prefix for all events of an aggregate.
func (a *PebbleEventStore) aggregatePrefix(
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
) []byte {
	return fmt.Appendf(nil, "%s%s:%s:", a.prefix, aggregateType, aggregateID)
}

// Save implements event.Store.Save with per-aggregate locking for concurrency safety.
func (a *PebbleEventStore) Save(
	_ context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	events []event.Event,
	expectedVersion event.Version,
) error {
	if len(events) == 0 {
		return nil
	}

	a.lockAggregate(aggregateType, aggregateID)
	defer a.unlockAggregate(aggregateType, aggregateID)

	err := a.checkVersion(aggregateType, aggregateID, expectedVersion)
	if err != nil {
		return event.WrapInfrastructure(err, "pebble.check_version",
			fmt.Sprintf("pebble check version for %s %s", aggregateType, aggregateID))
	}

	batch := a.db.NewBatch()

	defer func() { _ = batch.Close() }()

	err = a.writeEventsToBatch(
		batch, aggregateType, aggregateID, events, expectedVersion,
	)
	if err != nil {
		return event.WrapInfrastructure(
			err,
			"pebble.write_events",
			fmt.Sprintf(
				"pebble write %d events for %s %s",
				len(events),
				aggregateType,
				aggregateID,
			),
		)
	}

	return a.commitAndLog(batch, "events saved", aggregateType, aggregateID, len(events))
}

// eventPredicate returns true when iteration should stop BEFORE appending
// the current event. nil means "never stop early" (collect all events).
type eventPredicate func(event.Event) bool

// iterateEvents iterates over events in the database using the provided
// iterator configuration. If shouldStop is non-nil, iteration stops before
// appending the first event for which shouldStop returns true.
func (a *PebbleEventStore) iterateEvents(
	lowerBound, upperBound []byte,
	shouldStop eventPredicate,
) ([]event.Event, error) {
	iter, err := a.db.NewIter(&pebble.IterOptions{
		LowerBound: lowerBound,
		UpperBound: upperBound,
	})
	if err != nil {
		return nil, event.WrapInfrastructure(err, "pebble.create_iterator",
			"failed to create iterator")
	}

	defer func() { _ = iter.Close() }()

	var events []event.Event

	for iter.First(); iter.Valid(); iter.Next() {
		evt, err := a.deserializeEvent(iter.Value())
		if err != nil {
			if a.logger != nil {
				a.logger.Warn("corrupt event in pebble store",
					"key", string(iter.Key()), "error", err)
			}

			return nil, event.WrapCorruption(err, "pebble.corrupt_event",
				"corrupt event at key "+string(iter.Key()))
		}

		if shouldStop != nil && shouldStop(evt) {
			break
		}

		events = append(events, evt)
	}

	return events, checkIteratorError(iter)
}

// Load implements event.Store.Load.
func (a *PebbleEventStore) Load(
	_ context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
) ([]event.Event, error) {
	prefix := a.aggregatePrefix(aggregateType, aggregateID)
	upperBound := fmt.Appendf(nil, "%s%s:%s:\xff", a.prefix, aggregateType, aggregateID)

	return a.iterateEvents(prefix, upperBound, nil)
}

// LoadFromVersion implements event.Store.LoadFromVersion.
// Returns events with version strictly greater than the given version.
func (a *PebbleEventStore) LoadFromVersion(
	_ context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	version event.Version,
) ([]event.Event, error) {
	lowerBound := a.eventKey(aggregateType, aggregateID, version+1)
	upperBound := fmt.Appendf(nil, "%s%s:%s:\xff", a.prefix, aggregateType, aggregateID)

	return a.iterateEvents(lowerBound, upperBound, nil)
}

// loadFiltered iterates events and returns them filtered by predicate.
// Returns ErrAggregateNotFound if no events match the filter.
func (a *PebbleEventStore) loadFiltered(
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	upperBound []byte,
	predicate eventPredicate,
) ([]event.Event, error) {
	prefix := a.aggregatePrefix(aggregateType, aggregateID)

	events, err := a.iterateEvents(prefix, upperBound, predicate)
	if err != nil {
		return nil, err
	}

	if len(events) == 0 {
		return nil, event.ErrAggregateNotFound
	}

	return events, nil
}

// LoadToVersion retrieves events up to and including maxVersion.
func (a *PebbleEventStore) LoadToVersion(
	_ context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	maxVersion event.Version,
) ([]event.Event, error) {
	upperBound := a.eventKey(aggregateType, aggregateID, maxVersion+1)

	return a.loadFiltered(aggregateType, aggregateID, upperBound, nil)
}

// LoadToTimestamp retrieves events where OccurredAt <= maxTime.
// Uses early termination: since events are stored in version order and
// OccurredAt is monotonically increasing, the iterator stops as soon as it
// encounters an event past maxTime — avoiding a full aggregate scan.
func (a *PebbleEventStore) LoadToTimestamp(
	_ context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	maxTime time.Time,
) ([]event.Event, error) {
	upperBound := fmt.Appendf(nil, "%s%s:%s:\xff", a.prefix, aggregateType, aggregateID)
	predicate := func(evt event.Event) bool {
		return evt.OccurredAt().After(maxTime)
	}

	return a.loadFiltered(aggregateType, aggregateID, upperBound, predicate)
}

func (a *PebbleEventStore) aggregateLockKey(
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
) string {
	return string(aggregateType) + ":" + aggregateID.String()
}

func (a *PebbleEventStore) lockAggregate(
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
) {
	key := a.aggregateLockKey(aggregateType, aggregateID)

	m := &sync.Mutex{}
	actual, loaded := a.locks.LoadOrStore(key, m)
	if loaded {
		m = actual.(*sync.Mutex)
	}

	m.Lock()
}

func (a *PebbleEventStore) unlockAggregate(
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
) {
	key := a.aggregateLockKey(aggregateType, aggregateID)

	val, _ := a.locks.Load(key)
	val.(*sync.Mutex).Unlock()
}

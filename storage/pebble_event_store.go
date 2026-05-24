package storage

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/cockroachdb/pebble"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// PebbleEventStore implements go-cqrs-lite/event.Store using Pebble.
type PebbleEventStore struct {
	db     *pebble.DB
	logger *slog.Logger
	prefix string
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

// Save implements event.Store.Save with optimistic concurrency control.
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

	err := a.checkVersion(aggregateType, aggregateID, expectedVersion)
	if err != nil {
		return fmt.Errorf("pebble check version for %s %s: %w", aggregateType, aggregateID, err)
	}

	batch := a.db.NewBatch()

	defer func() { _ = batch.Close() }()

	err = a.writeEventsToBatch(
		batch, aggregateType, aggregateID, events, expectedVersion,
	)
	if err != nil {
		return fmt.Errorf(
			"pebble write %d events for %s %s: %w",
			len(events),
			aggregateType,
			aggregateID,
			err,
		)
	}

	return a.commitAndLog(batch, "events saved", aggregateType, aggregateID, len(events))
}

// iterateEvents iterates over events in the database using the provided iterator configuration.
func (a *PebbleEventStore) iterateEvents(lowerBound, upperBound []byte) ([]event.Event, error) {
	iter, err := a.db.NewIter(&pebble.IterOptions{
		LowerBound: lowerBound,
		UpperBound: upperBound,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create iterator: %w", err)
	}

	defer func() { _ = iter.Close() }()

	var events []event.Event

	for iter.First(); iter.Valid(); iter.Next() {
		evt, err := a.deserializeEvent(iter.Value())
		if err != nil {
			return nil, fmt.Errorf("corrupt event at key %s: %w", string(iter.Key()), err)
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

	return a.iterateEvents(prefix, upperBound)
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

	return a.iterateEvents(lowerBound, upperBound)
}

// LoadToVersion retrieves events up to and including maxVersion.
func (a *PebbleEventStore) LoadToVersion(
	_ context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	maxVersion event.Version,
) ([]event.Event, error) {
	prefix := a.aggregatePrefix(aggregateType, aggregateID)
	upperBound := a.eventKey(aggregateType, aggregateID, maxVersion+1)

	events, err := a.iterateEvents(prefix, upperBound)
	if err != nil {
		return nil, err
	}

	if len(events) == 0 {
		return nil, event.ErrAggregateNotFound
	}

	return events, nil
}

// LoadToTimestamp retrieves events where OccurredAt <= maxTime.
func (a *PebbleEventStore) LoadToTimestamp(
	_ context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	maxTime time.Time,
) ([]event.Event, error) {
	prefix := a.aggregatePrefix(aggregateType, aggregateID)
	upperBound := fmt.Appendf(nil, "%s%s:%s:\xff", a.prefix, aggregateType, aggregateID)

	all, err := a.iterateEvents(prefix, upperBound)
	if err != nil {
		return nil, err
	}

	if len(all) == 0 {
		return nil, event.ErrAggregateNotFound
	}

	var filtered []event.Event

	for _, e := range all {
		if !e.OccurredAt().After(maxTime) {
			filtered = append(filtered, e)
		}
	}

	if len(filtered) == 0 {
		return nil, event.ErrAggregateNotFound
	}

	return filtered, nil
}

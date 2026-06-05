package pebble

import (
	"context"
	"time"

	"github.com/cockroachdb/pebble"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

// eventPredicate returns true when iteration should stop BEFORE appending
// the current event. nil means "never stop early" (collect all events).
type eventPredicate func(event.Event) bool

// iterateEvents iterates over events in the database using the provided
// iterator configuration. If shouldStop is non-nil, iteration stops before
// appending the first event for which shouldStop returns true.
func (a *EventStore) iterateEvents(
	lowerBound, upperBound []byte,
	shouldStop eventPredicate,
) ([]event.Event, error) {
	iter, err := a.db.NewIter(
		&pebble.IterOptions{ //nolint:exhaustruct // only Lower/Upper bound needed
			LowerBound: lowerBound,
			UpperBound: upperBound,
		},
	)
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
func (a *EventStore) Load(
	_ context.Context,
	ref event.AggregateRef,
) ([]event.Event, error) {
	prefix := a.aggregatePrefix(ref)
	upperBound := a.aggregateUpperBound(ref)

	return a.iterateEvents(prefix, upperBound, nil)
}

// LoadFromVersion implements event.Store.LoadFromVersion.
// Returns events with version strictly greater than the given version.
func (a *EventStore) LoadFromVersion(
	_ context.Context,
	ref event.AggregateRef,
	version event.Version,
) ([]event.Event, error) {
	lowerBound := a.eventKey(ref, version+1)
	upperBound := a.aggregateUpperBound(ref)

	return a.iterateEvents(lowerBound, upperBound, nil)
}

// loadFiltered iterates events and returns them filtered by predicate.
// Returns ErrAggregateNotFound if no events match the filter.
func (a *EventStore) loadFiltered(
	ref event.AggregateRef,
	upperBound []byte,
	predicate eventPredicate,
) ([]event.Event, error) {
	prefix := a.aggregatePrefix(ref)

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
func (a *EventStore) LoadToVersion(
	_ context.Context,
	ref event.AggregateRef,
	maxVersion event.Version,
) ([]event.Event, error) {
	upperBound := a.eventKey(ref, maxVersion+1)

	return a.loadFiltered(ref, upperBound, nil)
}

// LoadToTimestamp retrieves events where OccurredAt <= maxTime.
// Uses early termination: since events are stored in version order and
// OccurredAt is monotonically increasing, the iterator stops as soon as it
// encounters an event past maxTime — avoiding a full aggregate scan.
func (a *EventStore) LoadToTimestamp(
	_ context.Context,
	ref event.AggregateRef,
	maxTime time.Time,
) ([]event.Event, error) {
	upperBound := a.aggregateUpperBound(ref)
	predicate := func(evt event.Event) bool {
		return evt.OccurredAt().After(maxTime)
	}

	return a.loadFiltered(ref, upperBound, predicate)
}

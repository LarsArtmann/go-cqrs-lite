package pebble

import (
	"fmt"

	"github.com/cockroachdb/pebble"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

func (a *EventStore) checkVersion(
	ref event.AggregateRef,
	expectedVersion event.Version,
) error {
	count, err := a.countEvents(ref)
	if err != nil {
		return event.WrapInfrastructure(err, "pebble.concurrency_check",
			"concurrency check")
	}

	err = event.CheckVersionConflict(count, expectedVersion)
	if err != nil {
		return event.WrapConflict(err, "pebble.version_conflict",
			"concurrency check")
	}

	return nil
}

func (a *EventStore) countEvents(ref event.AggregateRef) (int, error) {
	prefix := a.aggregatePrefix(ref)
	upperBound := a.aggregateUpperBound(ref)

	iter, err := a.db.NewIter(
		&pebble.IterOptions{ //nolint:exhaustruct // only Lower/Upper bound needed
			LowerBound: prefix,
			UpperBound: upperBound,
		},
	)
	if err != nil {
		return 0, event.WrapInfrastructure(err, "pebble.create_iterator",
			"failed to create count iterator")
	}

	defer func() { _ = iter.Close() }()

	count := 0

	for iter.First(); iter.Valid(); iter.Next() {
		count++
	}

	err = iter.Error()
	if err != nil {
		return 0, event.WrapInfrastructure(err, "pebble.iterator_error",
			"count iterator error")
	}

	return count, nil
}

func (a *EventStore) writeEventsToBatch(
	batch *pebble.Batch,
	ref event.AggregateRef,
	events []event.Event,
	expectedVersion event.Version,
) error {
	for i, evt := range events {
		err := validateEventOwnership(evt, ref)
		if err != nil {
			return event.WrapCorruption(err, "pebble.validate_event",
				fmt.Sprintf("validate event %d", i))
		}

		expectedEventVersion := expectedVersion.Int() + i + 1
		if evt.Version() != event.Version(expectedEventVersion) {
			return event.WrapConflict(ErrVersionMismatch, "pebble.version_mismatch",
				fmt.Sprintf("expected %d, got %d", expectedEventVersion, evt.Version()))
		}

		key := a.eventKey(ref, event.Version(expectedEventVersion))

		err = a.serializeAndAddToBatch(batch, key, evt)
		if err != nil {
			return event.WrapCorruption(err, "pebble.serialize_event",
				fmt.Sprintf("serialize event %d for %s %s", i, ref.Type, ref.ID))
		}
	}

	return a.appendToJournal(batch, events)
}

func validateEventOwnership(
	evt event.Event,
	ref event.AggregateRef,
) error {
	if evt.AggregateType() != ref.Type {
		return event.WrapConflict(ErrAggregateTypeMismatch, "pebble.aggregate_type_mismatch",
			fmt.Sprintf("expected %s, got %s", ref.Type, evt.AggregateType()))
	}

	if evt.AggregateID() != ref.ID {
		return event.WrapConflict(ErrAggregateIDMismatch, "pebble.aggregate_id_mismatch",
			fmt.Sprintf("expected %s, got %s", ref.ID, evt.AggregateID()))
	}

	return nil
}

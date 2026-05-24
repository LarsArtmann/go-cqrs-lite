package storage

import (
	"fmt"

	"github.com/cockroachdb/pebble"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

func (a *PebbleEventStore) checkVersion(
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	expectedVersion event.Version,
) error {
	prefix := a.aggregatePrefix(aggregateType, aggregateID)
	upperBound := fmt.Appendf(nil, "%s%s:%s:\xff", a.prefix, aggregateType, aggregateID)

	existing, err := a.iterateEvents(prefix, upperBound)
	if err != nil {
		return fmt.Errorf("concurrency check: %w", err)
	}

	err = event.CheckVersionConflict(len(existing), expectedVersion)
	if err != nil {
		return fmt.Errorf("concurrency check: %w", err)
	}

	return nil
}

func (a *PebbleEventStore) writeEventsToBatch(
	batch *pebble.Batch,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	events []event.Event,
	expectedVersion event.Version,
) error {
	for i, evt := range events {
		err := validateEventOwnership(evt, aggregateType, aggregateID)
		if err != nil {
			return fmt.Errorf("validate event %d: %w", i, err)
		}

		expectedEventVersion := expectedVersion.Int() + i + 1
		if evt.Version() != event.Version(expectedEventVersion) {
			return fmt.Errorf(
				"%w: expected %d, got %d",
				ErrVersionMismatch,
				expectedEventVersion,
				evt.Version(),
			)
		}

		key := a.eventKey(aggregateType, aggregateID, event.Version(expectedEventVersion))

		err = a.serializeAndAddToBatch(batch, key, evt)
		if err != nil {
			return fmt.Errorf(
				"serialize event %d for %s %s: %w",
				i,
				aggregateType,
				aggregateID,
				err,
			)
		}
	}

	return nil
}

func validateEventOwnership(
	evt event.Event,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
) error {
	if evt.AggregateType() != aggregateType {
		return fmt.Errorf(
			"%w: expected %s, got %s",
			ErrAggregateTypeMismatch,
			aggregateType,
			evt.AggregateType(),
		)
	}

	if evt.AggregateID() != aggregateID {
		return fmt.Errorf(
			"%w: expected %s, got %s",
			ErrAggregateIDMismatch,
			aggregateID,
			evt.AggregateID(),
		)
	}

	return nil
}

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

// CQRSAdapter implements go-cqrs-lite/event.Store using Pebble.
type CQRSAdapter struct {
	db     *pebble.DB
	logger *slog.Logger
	prefix string
}

// NewCQRSAdapter creates a new adapter using an existing Pebble DB.
func NewCQRSAdapter(db *pebble.DB, logger *slog.Logger) *CQRSAdapter {
	return &CQRSAdapter{
		db:     db,
		logger: logger,
		prefix: "cqrs_event:",
	}
}

// eventKey generates a storage key for an event.
// Pattern: cqrs_event:{aggregateType}:{aggregateID}:{version}.
func (a *CQRSAdapter) eventKey(
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	version event.Version,
) []byte {
	return fmt.Appendf(nil, "%s%s:%s:%010d", a.prefix, aggregateType, aggregateID, version.Int())
}

// aggregatePrefix returns the prefix for all events of an aggregate.
func (a *CQRSAdapter) aggregatePrefix(
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
) []byte {
	return fmt.Appendf(nil, "%s%s:%s:", a.prefix, aggregateType, aggregateID)
}

// Save implements event.Store.Save with optimistic concurrency control.
func (a *CQRSAdapter) Save(
	_ context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	events []event.Event,
	expectedVersion event.Version,
) error {
	if len(events) == 0 {
		return nil
	}

	// Optimistic concurrency check: verify current stream length matches expectedVersion.
	prefix := a.aggregatePrefix(aggregateType, aggregateID)
	upperBound := fmt.Appendf(nil, "%s%s:%s:\xff", a.prefix, aggregateType, aggregateID)

	existing, err := a.iterateEvents(prefix, upperBound)
	if err != nil {
		return fmt.Errorf("concurrency check: %w", err)
	}

	if len(existing) != expectedVersion.Int() {
		return fmt.Errorf(
			"%w: expected version %d, got %d",
			event.ErrVersionConflict,
			expectedVersion,
			len(existing),
		)
	}

	batch := a.db.NewBatch()

	defer func() { _ = batch.Close() }()

	for i, evt := range events {
		// Verify event belongs to this aggregate
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

		err := a.serializeAndAddToBatch(batch, key, evt)
		if err != nil {
			return err
		}
	}

	return a.commitAndLog(batch, "events saved", aggregateType, aggregateID, len(events))
}

// iterateEvents iterates over events in the database using the provided iterator configuration.
func (a *CQRSAdapter) iterateEvents(lowerBound, upperBound []byte) ([]event.Event, error) {
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
		event, err := a.deserializeEvent(iter.Value())
		if err != nil {
			a.logger.Warn("failed to deserialize event", slog.String("error", err.Error()))

			continue
		}

		events = append(events, event)
	}

	return events, checkIteratorError(iter)
}

// Load implements event.Store.Load.
func (a *CQRSAdapter) Load(
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
func (a *CQRSAdapter) LoadFromVersion(
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
func (a *CQRSAdapter) LoadToVersion(
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
func (a *CQRSAdapter) LoadToTimestamp(
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

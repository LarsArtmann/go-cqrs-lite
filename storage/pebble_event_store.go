package storage

import (
	"context"
	"encoding/json"
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

// Save implements event.Store.Save.
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

// logEventOperation logs a debug message for event operations.
func (a *CQRSAdapter) logEventOperation(
	msg string,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	count int,
) {
	a.logger.Debug(
		msg,
		slog.String("aggregate_type", string(aggregateType)),
		slog.String("aggregate_id", aggregateID.String()),
		slog.Int("count", count),
	)
}

// serializeAndAddToBatch serializes an event and adds it to the batch.
func (a *CQRSAdapter) serializeAndAddToBatch(
	batch *pebble.Batch,
	key []byte,
	evt event.Event,
) error {
	data, err := a.serializeEvent(evt)
	if err != nil {
		return fmt.Errorf("failed to serialize event: %w", err)
	}

	return a.addToBatch(batch, key, data)
}

// addToBatch is a helper that adds a key-value pair to a batch with error handling.
func (a *CQRSAdapter) addToBatch(batch *pebble.Batch, key, data []byte) error {
	err := batch.Set(key, data, nil)
	if err != nil {
		return fmt.Errorf("failed to add event to batch: %w", err)
	}

	return nil
}

// commitAndLog commits the batch and logs the operation.
func (a *CQRSAdapter) commitAndLog(
	batch *pebble.Batch,
	logMsg string,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	count int,
) error {
	err := batch.Commit(pebble.Sync)
	if err != nil {
		return fmt.Errorf("failed to commit %d events (%s): %w", count, logMsg, err)
	}

	a.logEventOperation(logMsg, aggregateType, aggregateID, count)

	return nil
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

// checkIteratorError checks an iterator for errors and returns an appropriate error.
func checkIteratorError(iter *pebble.Iterator) error {
	err := iter.Error()
	if err != nil {
		return fmt.Errorf("iterator error: %w", err)
	}

	return nil
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

// Delete implements event.Store.Delete.
func (a *CQRSAdapter) Delete(
	_ context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
) error {
	prefix := a.aggregatePrefix(aggregateType, aggregateID)
	upperBound := fmt.Appendf(nil, "%s%s:%s:\xff", a.prefix, aggregateType, aggregateID)

	iter, err := a.db.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: upperBound,
	})
	if err != nil {
		return fmt.Errorf("failed to create iterator: %w", err)
	}

	defer func() { _ = iter.Close() }()

	batch := a.db.NewBatch()

	defer func() { _ = batch.Close() }()

	count := 0

	for iter.First(); iter.Valid(); iter.Next() {
		err := batch.Delete(iter.Key(), nil)
		if err != nil {
			return fmt.Errorf("failed to delete event: %w", err)
		}

		count++
	}

	commitErr := batch.Commit(pebble.Sync)
	if commitErr != nil {
		return fmt.Errorf("failed to commit deletions: %w", commitErr)
	}

	a.logger.Debug(
		"events deleted",
		slog.String("aggregate_type", string(aggregateType)),
		slog.String("aggregate_id", aggregateID.String()),
		slog.Int("count", count),
	)

	return nil
}

// AppendBatch implements event.Store.AppendBatch.
// Appends events without optimistic concurrency checks.
func (a *CQRSAdapter) AppendBatch(
	_ context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	events []event.Event,
) error {
	if len(events) == 0 {
		return nil
	}

	batch := a.db.NewBatch()

	defer func() { _ = batch.Close() }()

	for _, evt := range events {
		key := a.eventKey(aggregateType, aggregateID, evt.Version())

		err := a.serializeAndAddToBatch(batch, key, evt)
		if err != nil {
			return err
		}
	}

	return a.commitAndLog(
		batch,
		"events appended in batch",
		aggregateType,
		aggregateID,
		len(events),
	)
}

// Close releases the Pebble database.
func (a *CQRSAdapter) Close() error {
	if a.db != nil {
		err := a.db.Close()
		if err != nil {
			return fmt.Errorf("close pebble db: %w", err)
		}
	}

	return nil
}

// Ensure CQRSAdapter implements event.Store.
var _ event.Store = (*CQRSAdapter)(nil)

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
				"event aggregate type mismatch: expected %s, got %s",
				aggregateType,
				evt.AggregateType(),
			)
		}

		if evt.AggregateID() != aggregateID {
			return fmt.Errorf(
				"event aggregate ID mismatch: expected %s, got %s",
				aggregateID,
				evt.AggregateID(),
			)
		}

		expectedEventVersion := expectedVersion.Int() + i + 1
		if evt.Version() != event.Version(expectedEventVersion) {
			return fmt.Errorf(
				"event version mismatch: expected %d, got %d",
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
	a.logger.Debug(msg,
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
		return fmt.Errorf("failed to commit events: %w", err)
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

	a.logger.Debug("events deleted",
		slog.String("aggregate_type", string(aggregateType)),
		slog.String("aggregate_id", aggregateID.String()),
		slog.Int("count", count),
	)

	return nil
}

// serializeEvent converts a CQRS event to JSON.
func (a *CQRSAdapter) serializeEvent(evt event.Event) ([]byte, error) {
	s := serializableEvent{
		ID:            evt.ID().String(),
		Type:          string(evt.Type()),
		AggregateID:   evt.AggregateID().String(),
		AggregateType: string(evt.AggregateType()),
		Version:       evt.Version().Int(),
		Payload:       evt.Payload(),
		OccurredAt:    evt.OccurredAt().UnixNano(),
	}

	if m := evt.Metadata(); m != nil {
		s.Metadata = &serializableMetadata{
			CorrelationID: m.CorrelationID.String(),
			CausationID:   m.CausationID.String(),
			UserID:        m.UserID.String(),
			RequestID:     m.RequestID.String(),
			Source:        string(m.Source),
			IPAddress:     string(m.IPAddress),
			UserAgent:     string(m.UserAgent),
		}
		if len(m.Custom) > 0 {
			s.Metadata.Custom = make(map[string]string, len(m.Custom))
			for k, v := range m.Custom {
				s.Metadata.Custom[string(k)] = v
			}
		}
	}

	return json.Marshal(s)
}

// deserializeEvent converts JSON to a CQRS-compatible event.
func (a *CQRSAdapter) deserializeEvent(data []byte) (event.Event, error) {
	var s serializableEvent

	err := json.Unmarshal(data, &s)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal event: %w", err)
	}

	aggregateID, err := id.ParseAggregateID(s.AggregateID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse aggregate ID: %w", err)
	}

	var opts []event.Option

	eventID, err := id.ParseEventID(s.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse event ID: %w", err)
	}

	opts = append(opts,
		event.WithEventID(eventID),
		event.WithOccurredAt(time.Unix(0, s.OccurredAt)),
	)

	if s.Metadata != nil {
		m := event.NewMetadata()
		if s.Metadata.CorrelationID != "" {
			m.CorrelationID, _ = id.ParseCorrelationID(s.Metadata.CorrelationID)
		}

		if s.Metadata.CausationID != "" {
			m.CausationID, _ = id.ParseCausationID(s.Metadata.CausationID)
		}

		if s.Metadata.UserID != "" {
			m.UserID, _ = id.ParseUserID(s.Metadata.UserID)
		}

		if s.Metadata.RequestID != "" {
			m.RequestID, _ = id.ParseRequestID(s.Metadata.RequestID)
		}

		m.Source = event.Source(s.Metadata.Source)
		m.IPAddress = event.IPAddress(s.Metadata.IPAddress)

		m.UserAgent = event.UserAgent(s.Metadata.UserAgent)
		if len(s.Metadata.Custom) > 0 {
			m.Custom = make(map[event.MetadataKey]string, len(s.Metadata.Custom))
			for k, v := range s.Metadata.Custom {
				m.Custom[event.MetadataKey(k)] = v
			}
		}

		opts = append(opts, event.WithMetadata(m))
	}

	return event.NewEvent(
		event.Type(s.Type),
		aggregateID,
		event.AggregateType(s.AggregateType),
		s.Version,
		s.Payload,
		opts...,
	)
}

// serializableEvent represents the JSON storage format for events.
type serializableEvent struct {
	ID            string                `json:"id"`
	Type          string                `json:"type"`
	AggregateID   string                `json:"aggregate_id"`
	AggregateType string                `json:"aggregate_type"`
	Version       int                   `json:"version"`
	Payload       []byte                `json:"payload"`
	OccurredAt    int64                 `json:"occurred_at"`
	Metadata      *serializableMetadata `json:"metadata,omitempty"`
}

type serializableMetadata struct {
	CorrelationID string            `json:"correlation_id,omitempty"`
	CausationID   string            `json:"causation_id,omitempty"`
	UserID        string            `json:"user_id,omitempty"`
	RequestID     string            `json:"request_id,omitempty"`
	Source        string            `json:"source,omitempty"`
	IPAddress     string            `json:"ip_address,omitempty"`
	UserAgent     string            `json:"user_agent,omitempty"`
	Custom        map[string]string `json:"custom,omitempty"`
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
		return a.db.Close()
	}

	return nil
}

// Ensure CQRSAdapter implements event.Store.
var _ event.Store = (*CQRSAdapter)(nil)

package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// Load retrieves all events for an aggregate, ordered by version.
// Returns ErrAggregateNotFound if no events exist for the aggregate.
func (s *SQLEventStore) Load(
	ctx context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
) ([]event.Event, error) {
	p1, p2 := s.dialect.Placeholder(1), s.dialect.Placeholder(2)

	query := fmt.Sprintf(
		`SELECT id, event_type, aggregate_type, aggregate_id, version, schema_version, payload, metadata, occurred_at
		FROM events
		WHERE aggregate_type = %s AND aggregate_id = %s
		ORDER BY version ASC`,
		p1,
		p2,
	)

	rows, err := s.db.QueryContext(ctx, query, string(aggregateType), aggregateID)
	if err != nil {
		return nil, fmt.Errorf("query events: %w", err)
	}

	defer func() {
		_ = rows.Close()
	}()

	events, err := s.scanEvents(rows)
	if err != nil {
		return nil, err
	}

	if len(events) == 0 {
		return nil, event.ErrAggregateNotFound
	}

	return events, nil
}

// LoadFromVersion retrieves events starting from a given version.
func (s *SQLEventStore) LoadFromVersion(
	ctx context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	version event.Version,
) ([]event.Event, error) {
	p1, p2, p3 := s.dialect.Placeholder(1), s.dialect.Placeholder(2), s.dialect.Placeholder(3)

	query := fmt.Sprintf(
		`SELECT id, event_type, aggregate_type, aggregate_id, version, schema_version, payload, metadata, occurred_at
		FROM events
		WHERE aggregate_type = %s AND aggregate_id = %s AND version > %s
		ORDER BY version ASC`,
		p1,
		p2,
		p3,
	)

	rows, err := s.db.QueryContext(
		ctx,
		query,
		string(aggregateType),
		aggregateID,
		version.Int(),
	)
	if err != nil {
		return nil, fmt.Errorf("query events from version: %w", err)
	}

	defer func() {
		_ = rows.Close()
	}()

	return s.scanEvents(rows)
}

// LoadToVersion retrieves events up to and including maxVersion.
// Returns ErrAggregateNotFound if no events exist for the aggregate.
func (s *SQLEventStore) LoadToVersion(
	ctx context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	maxVersion event.Version,
) ([]event.Event, error) {
	p1, p2, p3 := s.dialect.Placeholder(1), s.dialect.Placeholder(2), s.dialect.Placeholder(3)

	query := fmt.Sprintf(
		`SELECT id, event_type, aggregate_type, aggregate_id, version, schema_version, payload, metadata, occurred_at
		FROM events
		WHERE aggregate_type = %s AND aggregate_id = %s AND version <= %s
		ORDER BY version ASC`,
		p1,
		p2,
		p3,
	)

	rows, err := s.db.QueryContext(ctx, query, string(aggregateType), aggregateID, maxVersion.Int())
	if err != nil {
		return nil, fmt.Errorf("query events to version: %w", err)
	}

	defer func() {
		_ = rows.Close()
	}()

	events, err := s.scanEvents(rows)
	if err != nil {
		return nil, err
	}

	if len(events) == 0 {
		return nil, event.ErrAggregateNotFound
	}

	return events, nil
}

// LoadToTimestamp retrieves events where OccurredAt <= maxTime.
// Returns ErrAggregateNotFound if no events exist for the aggregate.
func (s *SQLEventStore) LoadToTimestamp(
	ctx context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	maxTime time.Time,
) ([]event.Event, error) {
	p1, p2, p3 := s.dialect.Placeholder(1), s.dialect.Placeholder(2), s.dialect.Placeholder(3)

	query := fmt.Sprintf(
		`SELECT id, event_type, aggregate_type, aggregate_id, version, schema_version, payload, metadata, occurred_at
		FROM events
		WHERE aggregate_type = %s AND aggregate_id = %s AND occurred_at <= %s
		ORDER BY version ASC`,
		p1,
		p2,
		p3,
	)

	rows, err := s.db.QueryContext(
		ctx, query,
		string(aggregateType),
		aggregateID,
		s.dialect.FormatTime(maxTime),
	)
	if err != nil {
		return nil, fmt.Errorf("query events to timestamp: %w", err)
	}

	defer func() {
		_ = rows.Close()
	}()

	events, err := s.scanEvents(rows)
	if err != nil {
		return nil, err
	}

	if len(events) == 0 {
		return nil, event.ErrAggregateNotFound
	}

	return events, nil
}

// Delete removes all events for an aggregate.
func (s *SQLEventStore) Delete(
	ctx context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
) error {
	p1, p2 := s.dialect.Placeholder(1), s.dialect.Placeholder(2)

	return deleteByAggregate(s.db, ctx, aggregateType, aggregateID, "events", p1, p2, "events")
}

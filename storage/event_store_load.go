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
	return s.queryEvents(ctx, aggregateType, aggregateID,
		"ORDER BY version ASC", nil,
		true, "query events",
	)
}

// LoadFromVersion retrieves events starting from a given version.
func (s *SQLEventStore) LoadFromVersion(
	ctx context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	version event.Version,
) ([]event.Event, error) {
	return s.queryEvents(ctx, aggregateType, aggregateID,
		fmt.Sprintf("AND version > %s ORDER BY version ASC", s.dialect.Placeholder(3)),
		[]any{version.Int()},
		false, "query events from version",
	)
}

// LoadToVersion retrieves events up to and including maxVersion.
// Returns ErrAggregateNotFound if no events exist for the aggregate.
func (s *SQLEventStore) LoadToVersion(
	ctx context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	maxVersion event.Version,
) ([]event.Event, error) {
	return s.queryEvents(ctx, aggregateType, aggregateID,
		fmt.Sprintf("AND version <= %s ORDER BY version ASC", s.dialect.Placeholder(3)),
		[]any{maxVersion.Int()},
		true, "query events to version",
	)
}

// LoadToTimestamp retrieves events where OccurredAt <= maxTime.
// Returns ErrAggregateNotFound if no events exist for the aggregate.
func (s *SQLEventStore) LoadToTimestamp(
	ctx context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	maxTime time.Time,
) ([]event.Event, error) {
	return s.queryEvents(ctx, aggregateType, aggregateID,
		fmt.Sprintf("AND occurred_at <= %s ORDER BY version ASC", s.dialect.Placeholder(3)),
		[]any{s.dialect.FormatTime(maxTime)},
		true, "query events to timestamp",
	)
}

func (s *SQLEventStore) queryEvents(
	ctx context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	whereSuffix string,
	extraArgs []any,
	requireNonEmpty bool,
	errMsg string,
) ([]event.Event, error) {
	p1, p2 := s.dialect.Placeholder(1), s.dialect.Placeholder(2)

	query := fmt.Sprintf(
		`SELECT id, event_type, aggregate_type, aggregate_id, version, schema_version, payload, metadata, occurred_at
		FROM events
		WHERE aggregate_type = %s AND aggregate_id = %s %s`,
		p1, p2, whereSuffix,
	)

	args := []any{string(aggregateType), aggregateID}
	args = append(args, extraArgs...)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errMsg, err)
	}

	defer func() {
		_ = rows.Close()
	}()

	events, err := s.scanEvents(rows)
	if err != nil {
		return nil, err
	}

	if requireNonEmpty && len(events) == 0 {
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

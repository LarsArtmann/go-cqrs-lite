package storage

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel"
)

// Load retrieves all events for an aggregate, ordered by version.
// Returns ErrAggregateNotFound if no events exist for the aggregate.
func (s *SQLEventStore) Load(
	ctx context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
) ([]event.Event, error) {
	ctx, span := cqrsotel.StartSpan(
		ctx, tracer(), "event.store.load",
		trace.SpanKindClient,
		trace.WithAttributes(aggregateAttrs(string(aggregateType), aggregateID.String())...),
	)
	defer span.End()

	events, err := s.queryEvents(
		ctx, aggregateType, aggregateID,
		"ORDER BY version ASC", nil,
		true, "query events",
	)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return nil, err
	}

	span.SetAttributes(attribute.Int(cqrsotel.AttrEventCount, len(events)))

	return events, nil
}

// LoadFromVersion retrieves events starting from a given version.
func (s *SQLEventStore) LoadFromVersion(
	ctx context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	version event.Version,
) ([]event.Event, error) {
	ctx, span := cqrsotel.StartSpan(
		ctx, tracer(), "event.store.load_from_version",
		trace.SpanKindClient,
		trace.WithAttributes(aggregateAttrsWithVersion(
			string(aggregateType), aggregateID.String(), version.Int(),
		)...),
	)
	defer span.End()

	events, err := s.queryEvents(
		ctx, aggregateType, aggregateID,
		fmt.Sprintf("AND version > %s ORDER BY version ASC", s.dialect.Placeholder(3)),
		[]any{version.Int()},
		false, "query events from version",
	)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return nil, err
	}

	span.SetAttributes(attribute.Int(cqrsotel.AttrEventCount, len(events)))

	return events, nil
}

// LoadToVersion retrieves events up to and including maxVersion.
// Returns ErrAggregateNotFound if no events exist for the aggregate.
func (s *SQLEventStore) LoadToVersion(
	ctx context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
	maxVersion event.Version,
) ([]event.Event, error) {
	ctx, span := cqrsotel.StartSpan(
		ctx, tracer(), "event.store.load_to_version",
		trace.SpanKindClient,
		trace.WithAttributes(aggregateAttrsWithVersion(
			string(aggregateType), aggregateID.String(), maxVersion.Int(),
		)...),
	)
	defer span.End()

	events, err := s.queryEvents(
		ctx, aggregateType, aggregateID,
		fmt.Sprintf("AND version <= %s ORDER BY version ASC", s.dialect.Placeholder(3)),
		[]any{maxVersion.Int()},
		true, "query events to version",
	)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return nil, err
	}

	span.SetAttributes(attribute.Int(cqrsotel.AttrEventCount, len(events)))

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
	ctx, span := cqrsotel.StartSpan(
		ctx, tracer(), "event.store.load_to_timestamp",
		trace.SpanKindClient,
		trace.WithAttributes(aggregateAttrs(string(aggregateType), aggregateID.String())...),
	)
	defer span.End()

	events, err := s.queryEvents(
		ctx, aggregateType, aggregateID,
		fmt.Sprintf("AND occurred_at <= %s ORDER BY version ASC", s.dialect.Placeholder(3)),
		[]any{s.dialect.FormatTime(maxTime)},
		true, "query events to timestamp",
	)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return nil, err
	}

	span.SetAttributes(attribute.Int(cqrsotel.AttrEventCount, len(events)))

	return events, nil
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
		FROM `+tableEvents+`
		WHERE aggregate_type = %s AND aggregate_id = %s %s`,
		p1,
		p2,
		whereSuffix,
	)

	args := make([]any, 0, 2+len(extraArgs))
	args = append(args, string(aggregateType), aggregateID)
	args = append(args, extraArgs...)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, event.WrapInfrastructure(err, "storage.query_events",
			errMsg+" (where="+whereSuffix+")")
	}

	defer func() {
		_ = rows.Close()
	}()

	events, err := s.scanEvents(rows)
	if err != nil {
		return nil, event.WrapInfrastructure(err, "storage.scan_events",
			errMsg+" (where="+whereSuffix+")")
	}

	if requireNonEmpty && len(events) == 0 {
		return nil, event.ErrAggregateNotFound
	}

	return events, nil
}

// LoadBackwards returns events in reverse version order (newest first).
// Returns ErrAggregateNotFound if no events exist for the aggregate.
func (s *SQLEventStore) LoadBackwards(
	ctx context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
) ([]event.Event, error) {
	ctx, span := cqrsotel.StartSpan(
		ctx, tracer(), "event.store.load_backwards",
		trace.SpanKindClient,
		trace.WithAttributes(aggregateAttrs(string(aggregateType), aggregateID.String())...),
	)
	defer span.End()

	events, err := s.queryEvents(
		ctx, aggregateType, aggregateID,
		"ORDER BY version DESC", nil,
		true, "query events backwards",
	)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return nil, err
	}

	span.SetAttributes(attribute.Int(cqrsotel.AttrEventCount, len(events)))

	return events, nil
}

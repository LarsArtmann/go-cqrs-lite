package storage

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel"
)

type loadParams struct {
	spanName   string
	attrs      []attribute.KeyValue
	where      string
	extraArgs  []any
	requireHit bool
	errMsg     string
}

// loadWithSpan starts a traced span, delegates to queryEvents, and records
// the result count. All Load* methods share this skeleton.
func (s *SQLEventStore) loadWithSpan(
	ctx context.Context,
	ref event.AggregateRef,
	p loadParams,
) ([]event.Event, error) {
	ctx, span := cqrsotel.StartSpan(
		ctx, tracer(), p.spanName,
		trace.SpanKindClient,
		trace.WithAttributes(p.attrs...),
	)
	defer span.End()

	events, err := s.queryEvents(
		ctx, ref,
		p.where, p.extraArgs,
		p.requireHit, p.errMsg,
	)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return nil, err
	}

	span.SetAttributes(attribute.Int(cqrsotel.AttrEventCount, len(events)))

	return events, nil
}

// loadSimple retrieves events for an aggregate with configurable sort order.
func (s *SQLEventStore) loadSimple(
	ctx context.Context,
	ref event.AggregateRef,
	spanName string,
	order string,
	errMsg string,
) ([]event.Event, error) {
	return s.loadWithSpan(ctx, ref, loadParams{
		spanName:   spanName,
		attrs:      cqrsotel.AggregateAttrs(ref.Type, ref.ID),
		where:      order,
		requireHit: true,
		errMsg:     errMsg,
	})
}

// Load retrieves all events for an aggregate, ordered by version.
// Returns ErrAggregateNotFound if no events exist for the aggregate.
func (s *SQLEventStore) Load(
	ctx context.Context,
	ref event.AggregateRef,
) ([]event.Event, error) {
	return s.loadSimple(
		ctx,
		ref,
		"event.store.load",
		"ORDER BY version ASC",
		"query events",
	)
}

// LoadFromVersion retrieves events starting from a given version.
func (s *SQLEventStore) LoadFromVersion(
	ctx context.Context,
	ref event.AggregateRef,
	version event.Version,
) ([]event.Event, error) {
	return s.loadWithSpan(ctx, ref, loadParams{
		spanName: "event.store.load_from_version",
		attrs: append(
			cqrsotel.AggregateAttrs(ref.Type, ref.ID),
			attribute.Int(cqrsotel.AttrAggregateVersion, version.Int()),
		),
		where:      fmt.Sprintf("AND version > %s ORDER BY version ASC", s.dialect.Placeholder(3)),
		extraArgs:  []any{version.Int()},
		requireHit: false,
		errMsg:     "query events from version",
	})
}

// LoadToVersion retrieves events up to and including maxVersion.
// Returns ErrAggregateNotFound if no events exist for the aggregate.
func (s *SQLEventStore) LoadToVersion(
	ctx context.Context,
	ref event.AggregateRef,
	maxVersion event.Version,
) ([]event.Event, error) {
	return s.loadWithSpan(ctx, ref, loadParams{
		spanName: "event.store.load_to_version",
		attrs: append(
			cqrsotel.AggregateAttrs(ref.Type, ref.ID),
			attribute.Int(cqrsotel.AttrAggregateVersion, maxVersion.Int()),
		),
		where:      fmt.Sprintf("AND version <= %s ORDER BY version ASC", s.dialect.Placeholder(3)),
		extraArgs:  []any{maxVersion.Int()},
		requireHit: true,
		errMsg:     "query events to version",
	})
}

// LoadToTimestamp retrieves events where OccurredAt <= maxTime.
// Returns ErrAggregateNotFound if no events exist for the aggregate.
func (s *SQLEventStore) LoadToTimestamp(
	ctx context.Context,
	ref event.AggregateRef,
	maxTime time.Time,
) ([]event.Event, error) {
	return s.loadWithSpan(ctx, ref, loadParams{
		spanName: "event.store.load_to_timestamp",
		attrs:    cqrsotel.AggregateAttrs(ref.Type, ref.ID),
		where: fmt.Sprintf(
			"AND occurred_at <= %s ORDER BY version ASC",
			s.dialect.Placeholder(3),
		),
		extraArgs:  []any{s.dialect.FormatTime(maxTime)},
		requireHit: true,
		errMsg:     "query events to timestamp",
	})
}

// LoadBackwards returns events in reverse version order (newest first).
// Returns ErrAggregateNotFound if no events exist for the aggregate.
func (s *SQLEventStore) LoadBackwards(
	ctx context.Context,
	ref event.AggregateRef,
) ([]event.Event, error) {
	return s.loadSimple(
		ctx,
		ref,
		"event.store.load_backwards",
		"ORDER BY version DESC",
		"query events backwards",
	)
}

func (s *SQLEventStore) queryEvents(
	ctx context.Context,
	ref event.AggregateRef,
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
	args = append(args, string(ref.Type), ref.ID)
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
		return nil, event.WrapRejection(event.ErrAggregateNotFound, "storage.aggregate_not_found",
			fmt.Sprintf("no events found for %s %s", ref.Type, ref.ID))
	}

	return events, nil
}

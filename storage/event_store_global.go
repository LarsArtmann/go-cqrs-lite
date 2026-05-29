package storage

import (
	"context"
	"database/sql"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel"
)

// ReadAll retrieves all events across all aggregates, ordered by occurrence time.
// Returns an empty slice (not an error) if no events exist.
func (s *SQLEventStore) ReadAll(ctx context.Context) ([]event.Event, error) {
	ctx, span := cqrsotel.StartSpan(
		ctx, tracer(), "event.store.read_all",
		trace.SpanKindClient,
	)
	defer span.End()

	query := `SELECT id, event_type, aggregate_type, aggregate_id, version, schema_version, payload, metadata, occurred_at
		FROM ` + tableEvents + `
		ORDER BY occurred_at ASC`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return nil, event.WrapInfrastructure(err, "storage.query_all_events",
			"query all events")
	}

	defer func() {
		_ = rows.Close()
	}()

	events, scanErr := s.scanEvents(rows)
	if scanErr != nil {
		cqrsotel.RecordError(span, scanErr)
	}

	span.SetAttributes(attribute.Int(cqrsotel.AttrEventCount, len(events)))

	return events, scanErr
}

// ReadFrom retrieves events ordered by OccurredAt, starting after the given event ID.
// Returns up to limit events. Implements event.SeekableJournal.
func (s *SQLEventStore) ReadFrom(
	ctx context.Context,
	afterEventID id.EventID,
	limit int,
) ([]event.Event, error) {
	ctx, span := cqrsotel.StartSpan(
		ctx, tracer(), "event.store.read_from",
		trace.SpanKindClient,
		trace.WithAttributes(attribute.Int("cqrs.outbox.limit", limit)),
	)
	defer span.End()

	if afterEventID.IsZero() {
		events, err := s.loadAllFromStart(ctx, limit)
		if err != nil {
			cqrsotel.RecordError(span, err)

			return events, fmt.Errorf("read from start (limit=%d): %w", limit, err)
		}

		span.SetAttributes(attribute.Int(cqrsotel.AttrEventCount, len(events)))

		return events, nil
	}

	p1 := s.dialect.Placeholder(1)
	p2 := s.dialect.Placeholder(2)

	query := fmt.Sprintf(
		`SELECT id, event_type, aggregate_type, aggregate_id, version, schema_version, payload, metadata, occurred_at
		FROM `+tableEvents+`
		WHERE id > %s
		ORDER BY occurred_at ASC`,
		p1,
	)

	args := []any{afterEventID.String()}

	if limit > 0 {
		query += " LIMIT " + p2

		args = append(args, limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return nil, event.WrapInfrastructure(err, "storage.query_from_position",
			fmt.Sprintf("query events from position (limit=%d)", limit))
	}

	defer func() {
		_ = rows.Close()
	}()

	events, scanErr := s.scanEvents(rows)
	if scanErr != nil {
		cqrsotel.RecordError(span, scanErr)
	}

	span.SetAttributes(attribute.Int(cqrsotel.AttrEventCount, len(events)))

	if scanErr != nil {
		return events, fmt.Errorf("read from position (limit=%d): %w", limit, scanErr)
	}

	return events, nil
}

// loadAllFromStart loads from the beginning, with optional limit.
func (s *SQLEventStore) loadAllFromStart(
	ctx context.Context,
	limit int,
) ([]event.Event, error) {
	if limit <= 0 {
		return s.ReadAll(ctx)
	}

	p1 := s.dialect.Placeholder(1)

	query := `SELECT id, event_type, aggregate_type, aggregate_id, version, schema_version, payload, metadata, occurred_at
		FROM ` + tableEvents + `
		ORDER BY occurred_at ASC
		LIMIT ` + p1

	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, event.WrapInfrastructure(err, "storage.query_from_start",
			fmt.Sprintf("query events from start (limit=%d)", limit))
	}

	defer func() {
		_ = rows.Close()
	}()

	return s.scanEvents(rows)
}

// queryContextWithError executes a db.QueryContext and wraps the error with
// otel recording and infrastructure classification.
func queryContextWithError(
	ctx context.Context,
	span trace.Span,
	db *sql.DB,
	query string,
	op string,
	msg string,
	queryArgs ...any,
) (*sql.Rows, error) {
	rows, err := db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		cqrsotel.RecordError(span, err)
		return nil, event.WrapInfrastructure(err, op, msg)
	}
	return rows, nil
}

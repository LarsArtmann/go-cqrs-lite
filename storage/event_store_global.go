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
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/sql"
)

func (s *SQLEventStore) ReadAll(ctx context.Context) ([]event.Event, error) {
	ctx, span := cqrsotel.StartSpan(ctx, sqlpkg.Tracer(), "event.store.read_all", trace.SpanKindClient)
	defer span.End()
	query := `SELECT id, event_type, aggregate_type, aggregate_id, version, schema_version, payload, metadata, occurred_at
		FROM ` + sqlpkg.TableEvents + ` ORDER BY occurred_at ASC`
	rows, err := s.DB.QueryContext(ctx, query)
	if err != nil {
		cqrsotel.RecordError(span, err)
		return nil, event.WrapInfrastructure(err, "storage.query_all_events", "query all events")
	}
	defer func() { _ = rows.Close() }()
	events, scanErr := s.scanEvents(rows)
	if scanErr != nil {
		cqrsotel.RecordError(span, scanErr)
	}
	span.SetAttributes(attribute.Int(cqrsotel.AttrEventCount, len(events)))
	return events, scanErr
}

func (s *SQLEventStore) ReadFrom(ctx context.Context, afterEventID id.EventID, limit int) ([]event.Event, error) {
	ctx, span := cqrsotel.StartSpan(ctx, sqlpkg.Tracer(), "event.store.read_from", trace.SpanKindClient,
		trace.WithAttributes(attribute.Int("cqrs.outbox.limit", limit)))
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
	p1 := s.Dialect.Placeholder(1)
	p2 := s.Dialect.Placeholder(2)
	query := fmt.Sprintf(
		`SELECT id, event_type, aggregate_type, aggregate_id, version, schema_version, payload, metadata, occurred_at
		FROM `+sqlpkg.TableEvents+` WHERE id > %s ORDER BY occurred_at ASC`,
		p1,
	)
	args := []any{afterEventID.String()}
	if limit > 0 {
		query += " LIMIT " + p2
		args = append(args, limit)
	}
	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		cqrsotel.RecordError(span, err)
		return nil, event.WrapInfrastructure(err, "storage.query_from_position",
			fmt.Sprintf("query events from position (limit=%d)", limit))
	}
	defer func() { _ = rows.Close() }()
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

func (s *SQLEventStore) loadAllFromStart(ctx context.Context, limit int) ([]event.Event, error) {
	if limit <= 0 {
		return s.ReadAll(ctx)
	}
	p1 := s.Dialect.Placeholder(1)
	query := `SELECT id, event_type, aggregate_type, aggregate_id, version, schema_version, payload, metadata, occurred_at
		FROM ` + sqlpkg.TableEvents + ` ORDER BY occurred_at ASC LIMIT ` + p1
	rows, err := s.DB.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, event.WrapInfrastructure(err, "storage.query_from_start",
			fmt.Sprintf("query events from start (limit=%d)", limit))
	}
	defer func() { _ = rows.Close() }()
	return s.scanEvents(rows)
}

func queryContextWithError(
	ctx context.Context,
	span trace.Span,
	db *sql.DB,
	query, op, msg string,
	queryArgs ...any,
) (*sql.Rows, error) {
	rows, err := db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		cqrsotel.RecordError(span, err)
		return nil, event.WrapInfrastructure(err, op, msg)
	}
	return rows, nil
}

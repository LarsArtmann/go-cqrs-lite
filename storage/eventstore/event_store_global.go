package eventstore

import (
	"context"
	"fmt"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

func (s *SQLEventStore) ReadAll(ctx context.Context) ([]event.Event, error) {
	if err := s.checkClosed(); err != nil {
		return nil, err
	}

	ctx, span := cqrsotel.StartSpan(
		ctx,
		sqlpkg.Tracer(),
		"event.store.read_all",
		cqrsotel.SpanKindClient,
	)
	defer span.End()
	query := `SELECT ` + sqlpkg.EventColumns + `
		FROM ` + sqlpkg.TableEvents + ` ORDER BY occurred_at ASC`
	rows, err := s.DB.QueryContext(ctx, query)
	if err != nil {
		cqrsotel.RecordError(span, err)
		return nil, errorfamily.WrapInfrastructure(
			err,
			"storage.query_all_events",
			"query all events",
		)
	}
	defer sqlpkg.CloseRows(rows)
	events, scanErr := s.scanEvents(rows)
	if scanErr != nil {
		cqrsotel.RecordError(span, scanErr)
	}
	span.SetAttributes(cqrsotel.AttrInt(cqrsotel.AttrEventCount, len(events)))
	return events, scanErr
}

func (s *SQLEventStore) ReadFrom(
	ctx context.Context,
	afterEventID id.EventID,
	limit int,
) ([]event.Event, error) {
	if err := s.checkClosed(); err != nil {
		return nil, errorfamily.Wrapf(err, errorfamily.Infrastructure, "storage.event_read_from",
			"read from store (limit=%d, after=%s)", limit, afterEventID)
	}

	ctx, span := cqrsotel.StartSpan(
		ctx,
		sqlpkg.Tracer(),
		"event.store.read_from",
		cqrsotel.SpanKindClient,
		cqrsotel.WithAttributes(cqrsotel.AttrInt("cqrs.journal.limit", limit)),
	)
	defer span.End()
	if afterEventID.IsZero() {
		events, err := s.loadAllFromStart(ctx, limit)
		if err != nil {
			cqrsotel.RecordError(span, err)
			return events, errorfamily.WrapInfrastructure(err, "storage.read_from_start",
				fmt.Sprintf("read from start (limit=%d)", limit))
		}
		span.SetAttributes(cqrsotel.AttrInt(cqrsotel.AttrEventCount, len(events)))
		return events, nil
	}
	p1 := s.Dialect.Placeholder(1)
	p2 := s.Dialect.Placeholder(2)
	p3 := s.Dialect.Placeholder(3)
	query := fmt.Sprintf(
		`SELECT e.id, e.event_type, e.aggregate_type, e.aggregate_id, e.version, e.schema_version, e.payload, e.payload_encoding, e.metadata, e.occurred_at
		FROM `+sqlpkg.TableEvents+` e
		JOIN `+sqlpkg.TableEvents+` c ON c.id = %s
		WHERE (e.occurred_at > c.occurred_at) OR (e.occurred_at = c.occurred_at AND e.id > %s)
		ORDER BY e.occurred_at ASC, e.id ASC`,
		p1,
		p2,
	)
	args := sqlpkg.CursorArgs(afterEventID.String())
	if limit > 0 {
		query += " LIMIT " + p3
		args = append(args, limit)
	}
	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		cqrsotel.RecordError(span, err)
		return nil, errorfamily.WrapInfrastructure(err, "storage.query_from_position",
			fmt.Sprintf("query events from position (limit=%d)", limit))
	}
	defer sqlpkg.CloseRows(rows)
	events, scanErr := s.scanEvents(rows)
	if scanErr != nil {
		cqrsotel.RecordError(span, scanErr)
		return events, errorfamily.WrapInfrastructure(scanErr, "storage.scan_from_position",
			fmt.Sprintf("scan events from position (limit=%d)", limit))
	}
	span.SetAttributes(cqrsotel.AttrInt(cqrsotel.AttrEventCount, len(events)))
	return events, nil
}

func (s *SQLEventStore) loadAllFromStart(ctx context.Context, limit int) ([]event.Event, error) {
	if limit <= 0 {
		return s.ReadAll(ctx)
	}
	p1 := s.Dialect.Placeholder(1)
	query := `SELECT ` + sqlpkg.EventColumns + `
		FROM ` + sqlpkg.TableEvents + ` ORDER BY occurred_at ASC LIMIT ` + p1
	rows, err := s.DB.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "storage.query_from_start",
			fmt.Sprintf("query events from start (limit=%d)", limit))
	}
	defer sqlpkg.CloseRows(rows)
	return s.scanEvents(rows)
}

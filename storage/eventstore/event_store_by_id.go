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

// EventByIDLoader is an opt-in interface for stores that can load a single
// event by its ID. This enables one indexed query instead of a version scan.
type EventByIDLoader interface {
	LoadByEventID(ctx context.Context, eventID id.EventID) (event.Event, error)
}

// LoadByEventID retrieves a single event by its globally unique ID.
// Returns event.ErrEventNotFound if no event with the given ID exists.
func (s *SQLEventStore) LoadByEventID(
	ctx context.Context,
	eventID id.EventID,
) (event.Event, error) {
	if err := s.checkClosed(); err != nil {
		return nil, err
	}

	ctx, span := cqrsotel.StartSpan(
		ctx, sqlpkg.Tracer(), "event.store.load_by_event_id",
		cqrsotel.SpanKindClient,
		cqrsotel.WithAttributes(cqrsotel.AttrString("cqrs.event.id", eventID.String())),
	)
	defer span.End()

	p1 := s.Dialect.Placeholder(1)
	query := fmt.Sprintf("SELECT %s FROM %s WHERE id = %s LIMIT 1",
		sqlpkg.EventColumns, sqlpkg.TableEvents, p1)

	rows, err := s.DB.QueryContext(ctx, query, eventID.String())
	if err != nil {
		cqrsotel.RecordError(span, err)

		return nil, errorfamily.WrapInfrastructure(err, "storage.query_by_event_id",
			"query event by ID "+eventID.String())
	}

	defer sqlpkg.CloseRows(rows)

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			cqrsotel.RecordError(span, err)

			return nil, errorfamily.WrapInfrastructure(err, "storage.scan_by_event_id",
				"scan event by ID "+eventID.String())
		}

		return nil, errorfamily.WrapRejection(event.ErrEventNotFound, "storage.event_not_found",
			"event "+eventID.String()+" not found")
	}

	evt, err := s.scanEvent(rows)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return nil, err
	}

	return evt, nil
}

var _ EventByIDLoader = (*SQLEventStore)(nil)

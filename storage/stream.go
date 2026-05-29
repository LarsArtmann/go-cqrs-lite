package storage

import (
	"context"
	"database/sql"
	"fmt"

	"go.opentelemetry.io/otel/trace"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/sql"
)

var _ event.StreamLoader = (*SQLEventStore)(nil)

func (s *SQLEventStore) LoadStream(ctx context.Context, ref event.AggregateRef) (event.EventStream, error) {
	ctx, span := cqrsotel.StartSpan(ctx, sqlpkg.Tracer(), "event.store.load_stream", trace.SpanKindClient,
		trace.WithAttributes(cqrsotel.AggregateAttrs(ref.Type, ref.ID)...))
	defer span.End()
	p1, p2 := s.Dialect.Placeholder(1), s.Dialect.Placeholder(2)
	query := fmt.Sprintf(
		`SELECT id, event_type, aggregate_type, aggregate_id, version, schema_version, payload, metadata, occurred_at
		FROM `+sqlpkg.TableEvents+` WHERE aggregate_type = %s AND aggregate_id = %s ORDER BY version ASC`,
		p1,
		p2,
	)
	rows, err := s.DB.QueryContext(ctx, query, string(ref.Type), ref.ID)
	if err != nil {
		cqrsotel.RecordError(span, err)
		return nil, event.WrapInfrastructure(err, "storage.stream_query", "sql stream query")
	}
	return &sqlEventStream{rows: rows, store: s}, nil
}

type sqlEventStream struct {
	rows  *sql.Rows
	store *SQLEventStore
	err   error
}

func (s *sqlEventStream) Next() (event.Event, bool) {
	if s.err != nil {
		return nil, false
	}
	if !s.rows.Next() {
		return nil, false
	}
	evt, err := s.store.scanEvent(s.rows)
	if err != nil {
		s.err = err
		return nil, false
	}
	return evt, true
}

func (s *sqlEventStream) Err() error {
	if s.err != nil {
		return s.err
	}
	return s.rows.Err()
}
func (s *sqlEventStream) Close() error { return s.rows.Close() }

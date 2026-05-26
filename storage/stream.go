package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

var _ event.StreamLoader = (*SQLEventStore)(nil)

// LoadStream returns a cursor-based event stream for the given aggregate.
// Events are yielded one at a time as Next() is called, minimizing memory usage
// for aggregates with large event histories.
func (s *SQLEventStore) LoadStream(
	ctx context.Context,
	aggregateType event.AggregateType,
	aggregateID id.AggregateID,
) (event.EventStream, error) {
	p1, p2 := s.dialect.Placeholder(1), s.dialect.Placeholder(2)

	query := fmt.Sprintf(
		`SELECT id, event_type, aggregate_type, aggregate_id, version, schema_version, payload, metadata, occurred_at
		FROM events
		WHERE aggregate_type = %s AND aggregate_id = %s
		ORDER BY version ASC`,
		p1, p2,
	)

	rows, err := s.db.QueryContext(ctx, query, string(aggregateType), aggregateID)
	if err != nil {
		return nil, fmt.Errorf("sql stream query: %w", err)
	}

	return &sqlEventStream{rows: rows, store: s}, nil
}

// sqlEventStream yields events from an open sql.Rows cursor.
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

func (s *sqlEventStream) Close() error {
	return s.rows.Close()
}

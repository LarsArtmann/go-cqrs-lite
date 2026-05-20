package storage

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// LoadAll retrieves all events across all aggregates, ordered by occurrence time.
// Returns an empty slice (not an error) if no events exist.
func (s *SQLEventStore) LoadAll(ctx context.Context) ([]event.Event, error) {
	query := `SELECT id, event_type, aggregate_type, aggregate_id, version, schema_version, payload, metadata, occurred_at
		FROM events
		ORDER BY occurred_at ASC`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query all events: %w", err)
	}

	defer func() {
		_ = rows.Close()
	}()

	return s.scanEvents(rows)
}

// LoadAllFromPosition retrieves events ordered by OccurredAt, starting after the given event ID.
// Returns up to limit events. Implements event.PositionalLoader.
func (s *SQLEventStore) LoadAllFromPosition(
	ctx context.Context,
	afterEventID id.EventID,
	limit int,
) ([]event.Event, error) {
	if afterEventID.IsZero() {
		return s.loadAllFromStart(ctx, limit)
	}

	p1 := s.dialect.Placeholder(1)
	p2 := s.dialect.Placeholder(2)

	query := fmt.Sprintf(
		`SELECT id, event_type, aggregate_type, aggregate_id, version, schema_version, payload, metadata, occurred_at
		FROM events
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
		return nil, fmt.Errorf("query events from position: %w", err)
	}

	defer func() {
		_ = rows.Close()
	}()

	return s.scanEvents(rows)
}

// loadAllFromStart loads from the beginning, with optional limit.
func (s *SQLEventStore) loadAllFromStart(
	ctx context.Context,
	limit int,
) ([]event.Event, error) {
	if limit <= 0 {
		return s.LoadAll(ctx)
	}

	p1 := s.dialect.Placeholder(1)

	query := `SELECT id, event_type, aggregate_type, aggregate_id, version, schema_version, payload, metadata, occurred_at
		FROM events
		ORDER BY occurred_at ASC
		LIMIT ` + p1

	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("query events from start: %w", err)
	}

	defer func() {
		_ = rows.Close()
	}()

	return s.scanEvents(rows)
}

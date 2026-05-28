package storage

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// ReadAll retrieves all events across all aggregates, ordered by occurrence time.
// Returns an empty slice (not an error) if no events exist.
func (s *SQLEventStore) ReadAll(ctx context.Context) ([]event.Event, error) {
	query := `SELECT id, event_type, aggregate_type, aggregate_id, version, schema_version, payload, metadata, occurred_at
		FROM ` + tableEvents + `
		ORDER BY occurred_at ASC`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, event.WrapInfrastructure(err, "storage.query_all_events",
			"query all events")
	}

	defer func() {
		_ = rows.Close()
	}()

	return s.scanEvents(rows)
}

// ReadFrom retrieves events ordered by OccurredAt, starting after the given event ID.
// Returns up to limit events. Implements event.SeekableJournal.
func (s *SQLEventStore) ReadFrom(
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
		return nil, event.WrapInfrastructure(err, "storage.query_from_position",
			fmt.Sprintf("query events from position (limit=%d)", limit))
	}

	defer func() {
		_ = rows.Close()
	}()

	return s.scanEvents(rows)
}

// LoadAll retrieves all events across all aggregates, ordered by occurrence time.
//
// Deprecated: use ReadAll instead.
func (s *SQLEventStore) LoadAll(ctx context.Context) ([]event.Event, error) {
	return s.ReadAll(ctx)
}

// LoadAllFromPosition retrieves events ordered by OccurredAt, starting after the given event ID.
//
// Deprecated: use ReadFrom instead.
func (s *SQLEventStore) LoadAllFromPosition(
	ctx context.Context,
	afterEventID id.EventID,
	limit int,
) ([]event.Event, error) {
	return s.ReadFrom(ctx, afterEventID, limit)
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

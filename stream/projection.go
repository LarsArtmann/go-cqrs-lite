package stream

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/core/event"
)

// AggregateProjection maintains the stream_aggregates read model table.
// Register it with projection.Runner to keep the read model in sync.
type AggregateProjection struct {
	db        *sql.DB
	tableName string
}

// NewAggregateProjection creates a projection that maintains the aggregates table.
// The table is created if it does not exist.
func NewAggregateProjection(db *sql.DB, tablePrefix string) (*AggregateProjection, error) {
	err := validateTablePrefix(tablePrefix)
	if err != nil {
		return nil, fmt.Errorf("invalid table prefix %q: %w", tablePrefix, err)
	}

	p := &AggregateProjection{
		db:        db,
		tableName: tablePrefix + "stream_aggregates",
	}

	err = p.createTable()
	if err != nil {
		return nil, event.WrapInfrastructure(
			err,
			"stream.create_table",
			"create aggregates table",
		)
	}

	return p, nil
}

// Name returns the projection name for checkpoint tracking.
func (p *AggregateProjection) Name() string { return "stream.aggregate_projection" }

// EventTypes returns nil to subscribe to all event types.
func (p *AggregateProjection) EventTypes() []event.Type { return nil }

// Handle upserts the aggregate row for each event.
func (p *AggregateProjection) Handle(ctx context.Context, evt event.Event) error {
	status := detectStatusFromMetadata(evt)

	if status == event.TombstoneUndetermined {
		_, err := p.db.ExecContext(
			ctx,
			fmt.Sprintf(`INSERT INTO %s
				(aggregate_type, aggregate_id, version, event_count, last_event_at, tombstone_status)
				VALUES (?, ?, ?, 1, ?, 0)
				ON CONFLICT (aggregate_type, aggregate_id) DO UPDATE SET
					version = excluded.version,
					event_count = %s.event_count + 1,
					last_event_at = excluded.last_event_at`, p.tableName, p.tableName),
			evt.AggregateType(),
			evt.AggregateID().String(),
			evt.Version().Int(),
			evt.OccurredAt(),
		)

		return err
	}

	_, err := p.db.ExecContext(
		ctx,
		fmt.Sprintf(`INSERT INTO %s
			(aggregate_type, aggregate_id, version, event_count, last_event_at, tombstone_status)
			VALUES (?, ?, ?, 1, ?, ?)
			ON CONFLICT (aggregate_type, aggregate_id) DO UPDATE SET
				version = excluded.version,
				event_count = %s.event_count + 1,
				last_event_at = excluded.last_event_at,
				tombstone_status = excluded.tombstone_status`, p.tableName, p.tableName),
		evt.AggregateType(),
		evt.AggregateID().String(),
		evt.Version().Int(),
		evt.OccurredAt(),
		int(status),
	)

	return err
}

func (p *AggregateProjection) createTable() error {
	_, err := p.db.Exec(fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		aggregate_type  TEXT NOT NULL,
		aggregate_id    TEXT NOT NULL,
		version         INT  NOT NULL,
		event_count     INT  NOT NULL DEFAULT 0,
		last_event_at   TIMESTAMP NOT NULL,
		tombstone_status INT NOT NULL DEFAULT 0,
		PRIMARY KEY (aggregate_type, aggregate_id)
	)`, p.tableName))

	return err
}

// detectStatusFromMetadata checks a single event's metadata for tombstone/rebirth markers.
// Returns Undetermined if no markers found (normal event).
func detectStatusFromMetadata(evt event.Event) event.TombstoneStatus {
	md := evt.Metadata()
	if md == nil || md.Custom == nil {
		return event.TombstoneUndetermined
	}

	if md.Custom[event.MetadataKeyRebirth] == "true" {
		return event.TombstoneActive
	}

	if md.Custom[event.MetadataKeyTombstone] == "true" {
		return event.TombstoneTombstoned
	}

	return event.TombstoneUndetermined
}

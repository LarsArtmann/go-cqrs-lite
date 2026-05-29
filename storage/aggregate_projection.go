package storage

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"

	"github.com/larsartmann/go-cqrs-lite/event"
)

var validListingTablePrefix = regexp.MustCompile(`^[a-z_][a-z0-9_]*$`)

func validateListingTablePrefix(prefix string) error {
	if !validListingTablePrefix.MatchString(prefix) {
		return event.NewRejection(
			"listing.invalid_table_prefix",
			"invalid table prefix: must match ^[a-z_][a-z0-9_]*$",
		)
	}

	return nil
}

type AggregateProjection struct {
	db        *sql.DB
	tableName string
}

func NewAggregateProjection(db *sql.DB, tablePrefix string) (*AggregateProjection, error) {
	err := validateListingTablePrefix(tablePrefix)
	if err != nil {
		return nil, fmt.Errorf("invalid table prefix %q: %w", tablePrefix, err)
	}

	p := &AggregateProjection{
		db:        db,
		tableName: tablePrefix + "listing_aggregates",
	}

	err = p.createTable()
	if err != nil {
		return nil, event.WrapInfrastructure(
			err,
			"listing.create_table",
			"create aggregates table",
		)
	}

	return p, nil
}

func (p *AggregateProjection) Name() string { return "listing.aggregate_projection" }

func (p *AggregateProjection) EventTypes() []event.Type { return nil }

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
	_, err := p.db.ExecContext(context.Background(), fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
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

func detectStatusFromMetadata(evt event.Event) event.TombstoneStatus {
	md := evt.Metadata()
	if md.Custom == nil {
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

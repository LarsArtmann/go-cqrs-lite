package storage

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v2/sql"
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
	dialect   sqlpkg.Dialect
	tableName string
}

func NewAggregateProjection(
	ctx context.Context,
	db *sql.DB,
	tablePrefix string,
	dialect sqlpkg.Dialect,
) (*AggregateProjection, error) {
	err := validateListingTablePrefix(tablePrefix)
	if err != nil {
		return nil, event.WrapRejection(err, "listing.invalid_table_prefix",
			fmt.Sprintf("invalid table prefix %q", tablePrefix))
	}

	p := &AggregateProjection{
		db:        db,
		dialect:   dialect,
		tableName: tablePrefix + "listing_aggregates",
	}

	err = p.createTable(ctx)
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
		p1 := p.dialect.Placeholder(1)
		p2 := p.dialect.Placeholder(2)
		p3 := p.dialect.Placeholder(3)
		p4 := p.dialect.Placeholder(4)

		_, err := p.db.ExecContext(
			ctx,
			fmt.Sprintf(`INSERT INTO %s
				(aggregate_type, aggregate_id, version, event_count, last_event_at, tombstone_status)
				VALUES (%s, %s, %s, 1, %s, 0)
				ON CONFLICT (aggregate_type, aggregate_id) DO UPDATE SET
					version = excluded.version,
					event_count = %s.event_count + 1,
					last_event_at = excluded.last_event_at`, p.tableName, p1, p2, p3, p4, p.tableName),
			evt.AggregateType(),
			evt.AggregateID().String(),
			evt.Version().Int(),
			evt.OccurredAt(),
		)

		return err
	}

	p1 := p.dialect.Placeholder(1)
	p2 := p.dialect.Placeholder(2)
	p3 := p.dialect.Placeholder(3)
	p4 := p.dialect.Placeholder(4)
	p5 := p.dialect.Placeholder(5)

	_, err := p.db.ExecContext(
		ctx,
		fmt.Sprintf(`INSERT INTO %s
			(aggregate_type, aggregate_id, version, event_count, last_event_at, tombstone_status)
			VALUES (%s, %s, %s, 1, %s, %s)
			ON CONFLICT (aggregate_type, aggregate_id) DO UPDATE SET
				version = excluded.version,
				event_count = %s.event_count + 1,
				last_event_at = excluded.last_event_at,
				tombstone_status = excluded.tombstone_status`, p.tableName, p1, p2, p3, p4, p5, p.tableName),
		evt.AggregateType(),
		evt.AggregateID().String(),
		evt.Version().Int(),
		evt.OccurredAt(),
		int(status),
	)

	return err
}

func (p *AggregateProjection) createTable(ctx context.Context) error {
	_, err := p.db.ExecContext(ctx, fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
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

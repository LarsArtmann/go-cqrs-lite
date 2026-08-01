package storage

import (
	"context"
	"database/sql"
	"fmt"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

type StreamProjection struct {
	db      *sql.DB
	dialect sqlpkg.Dialect
	table   listingTable
}

func NewStreamProjection(
	ctx context.Context,
	db *sql.DB,
	tablePrefix string,
	dialect sqlpkg.Dialect,
) (*StreamProjection, error) {
	tbl, err := newListingTable(tablePrefix)
	if err != nil {
		return nil, errorfamily.WrapRejection(err, "listing.invalid_table_prefix",
			fmt.Sprintf("invalid table prefix %q", tablePrefix))
	}

	p := &StreamProjection{
		db:      db,
		dialect: dialect,
		table:   tbl,
	}

	err = p.createTable(ctx)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(
			err,
			"listing.create_table",
			"create streams table",
		)
	}

	return p, nil
}

func (p *StreamProjection) Name() string { return "listing.aggregate_projection" }

func (p *StreamProjection) EventTypes() []event.Type { return nil }

func (p *StreamProjection) Handle(ctx context.Context, evt event.Event) error {
	status := detectStatusFromMetadata(evt)

	if status == event.TombstoneUndetermined {
		p1 := p.dialect.Placeholder(1)
		p2 := p.dialect.Placeholder(2)
		p3 := p.dialect.Placeholder(3)
		p4 := p.dialect.Placeholder(4)

		setExprs := []string{
			"version = " + p.dialect.ExcludedRef("version"),
			"event_count = " + p.table.name + ".event_count + 1",
			"last_event_at = " + p.dialect.ExcludedRef("last_event_at"),
		}
		_, err := p.db.ExecContext(
			ctx,
			fmt.Sprintf(`INSERT INTO %s
				(aggregate_type, aggregate_id, version, event_count, last_event_at, tombstone_status)
				VALUES (%s, %s, %s, 1, %s, 0) %s`,
				p.table.name, p1, p2, p3, p4,
				p.dialect.OnConflictDoUpdate([]string{"aggregate_type", "aggregate_id"}, setExprs)), //nolint:goconst // SQL col
			evt.StreamType(),
			evt.StreamID().String(),
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

	setExprs := []string{
		"version = " + p.dialect.ExcludedRef("version"),
		"event_count = " + p.table.name + ".event_count + 1",
		"last_event_at = " + p.dialect.ExcludedRef("last_event_at"),
		"tombstone_status = " + p.dialect.ExcludedRef("tombstone_status"),
	}
	_, err := p.db.ExecContext(
		ctx,
		fmt.Sprintf(`INSERT INTO %s
			(aggregate_type, aggregate_id, version, event_count, last_event_at, tombstone_status)
			VALUES (%s, %s, %s, 1, %s, %s) %s`,
			p.table.name, p1, p2, p3, p4, p5,
			p.dialect.OnConflictDoUpdate([]string{"aggregate_type", "aggregate_id"}, setExprs)), //nolint:goconst // SQL col
		evt.StreamType(),
		evt.StreamID().String(),
		evt.Version().Int(),
		evt.OccurredAt(),
		int(status),
	)

	return err
}

func (p *StreamProjection) createTable(ctx context.Context) error {
	_, err := p.db.ExecContext(ctx, fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		aggregate_type  TEXT NOT NULL,
		aggregate_id    TEXT NOT NULL,
		version         INT  NOT NULL,
		event_count     INT  NOT NULL DEFAULT 0,
		last_event_at   TIMESTAMP NOT NULL,
		tombstone_status INT NOT NULL DEFAULT 0,
		PRIMARY KEY (aggregate_type, aggregate_id)
	)`, p.table.name))

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

// Deprecated: use StreamProjection.
type AggregateProjection = StreamProjection

// Deprecated: use NewStreamProjection.
func NewAggregateProjection(
	ctx context.Context,
	db *sql.DB,
	tablePrefix string,
	dialect sqlpkg.Dialect,
) (*StreamProjection, error) {
	return NewStreamProjection(ctx, db, tablePrefix, dialect)
}

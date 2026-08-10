package storage

import (
	"context"
	"database/sql"
	"fmt"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

// SQL column names used in ON CONFLICT clauses across multiple functions.
const (
	aggregateTypeCol = "aggregate_type"
	aggregateIDCol   = "aggregate_id"
)

// StreamProjectionOption configures a [StreamProjection].
type StreamProjectionOption func(*StreamProjection)

// WithDeleteTypes configures which event types signal stream deletion
// (ADR-0114). When the projection receives an event whose type matches one of
// these, it marks the stream as deleted (tombstone_status = 1). Without this
// option, all streams are considered active.
func WithDeleteTypes(types ...event.Type) StreamProjectionOption {
	return func(p *StreamProjection) {
		for _, t := range types {
			p.deleteTypes[t] = struct{}{}
		}
	}
}

type StreamProjection struct {
	db          *sql.DB
	dialect     sqlpkg.Dialect
	table       listingTable
	deleteTypes map[event.Type]struct{}
}

func NewStreamProjection(
	ctx context.Context,
	db *sql.DB,
	tablePrefix string,
	dialect sqlpkg.Dialect,
	opts ...StreamProjectionOption,
) (*StreamProjection, error) {
	tbl, err := newListingTable(tablePrefix)
	if err != nil {
		return nil, errorfamily.WrapRejection(err, "listing.invalid_table_prefix",
			fmt.Sprintf("invalid table prefix %q", tablePrefix))
	}

	p := &StreamProjection{
		db:          db,
		dialect:     dialect,
		table:       tbl,
		deleteTypes: map[event.Type]struct{}{},
	}

	for _, opt := range opts {
		opt(p)
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
	statusInt := 0 // active
	if _, isDelete := p.deleteTypes[evt.Type()]; isDelete {
		statusInt = 1 // deleted
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
			p.dialect.OnConflictDoUpdate([]string{aggregateTypeCol, aggregateIDCol}, setExprs)),
		evt.StreamType(),
		evt.StreamID().String(),
		evt.Version().Int(),
		evt.OccurredAt(),
		statusInt,
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

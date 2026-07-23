package relational

import (
	"context"
	"database/sql"
	"fmt"
	"slices"

	errorfamily "github.com/larsartmann/go-error-family"

	cqrsevent "github.com/larsartmann/go-cqrs-lite/event/v4"
	cqrsprojection "github.com/larsartmann/go-cqrs-lite/projection/v4"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

// RelationalHandler processes one event, writing through sink to any number of
// the projection's tables. All sink writes commit atomically when the handler
// returns nil and roll back when it returns an error.
//
// The handler is dialect-agnostic: it never references *sql.DB, *sql.Tx, or a
// specific SQL dialect. The backend (SQLite or PostgreSQL) is fixed when the
// [RelationalProjection] is constructed.
type RelationalHandler func(ctx context.Context, evt cqrsevent.Event, sink ProjectionSink) error

// RelationalProjection is an [cqrsprojection.Projection] that materialises events
// into a relational read model spanning multiple tables.
//
// It is the multi-table counterpart to [stack.Materialize]: where Materialize
// writes a single record to a single [kv.ViewStore] per event, RelationalProjection
// opens one transaction per event and hands the handler a [ProjectionSink] that
// can write across all the schema's tables atomically. This is the right tool
// when one event must update several related tables — messages plus their
// attachments, members plus their roles, a parent row plus its child
// collections, a junction table, or an append-only history table.
//
// All writes within one Handle call are atomic (BEGIN … COMMIT), so a partial
// failure leaves the read model untouched and the event can be retried.
type RelationalProjection struct {
	name        string
	schema      RelationalSchema
	db          *sql.DB
	dialect     sqlpkg.Dialect
	handler     RelationalHandler
	types       []cqrsevent.Type
	autoMigrate bool
}

// RelationalProjectionOption configures a RelationalProjection.
type RelationalProjectionOption func(*RelationalProjection)

// NewRelationalProjection creates a projection that writes events into schema
// across multiple tables, committing each event atomically. db and dialect
// select the backend (e.g. SQLiteDialect{} or PostgresDialect{}); handler is
// dialect-agnostic. types filters which event types the projection receives.
//
// The schema is auto-migrated (CREATE TABLE IF NOT EXISTS) at construction.
// Pass [WithoutRelationalAutoMigrate] to manage migrations externally.
func NewRelationalProjection(
	name string,
	schema RelationalSchema,
	db *sql.DB,
	dialect sqlpkg.Dialect,
	handler RelationalHandler,
	types []cqrsevent.Type,
	opts ...RelationalProjectionOption,
) (*RelationalProjection, error) {
	if name == "" {
		return nil, errRelationalNoName
	}

	if db == nil {
		return nil, errRelationalNilDB
	}

	if dialect == nil {
		return nil, errRelationalNilDialect
	}

	if handler == nil {
		return nil, errRelationalNilHandler
	}

	if err := schema.Validate(); err != nil {
		return nil, err
	}

	p := &RelationalProjection{
		name:        name,
		schema:      schema,
		db:          db,
		dialect:     dialect,
		handler:     handler,
		types:       slices.Clone(types),
		autoMigrate: true,
	}

	for _, opt := range opts {
		opt(p)
	}

	if p.autoMigrate {
		if err := p.schema.Migrate(context.Background(), db); err != nil {
			return nil, err
		}
	}

	return p, nil
}

// Name implements [cqrsprojection.Projection].
func (p *RelationalProjection) Name() string { return p.name }

// EventTypes implements [cqrsprojection.Projection].
func (p *RelationalProjection) EventTypes() []cqrsevent.Type { return slices.Clone(p.types) }

// Handle runs the handler inside a single transaction, committing on success
// and rolling back on error.
func (p *RelationalProjection) Handle(ctx context.Context, evt cqrsevent.Event) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return errorfamily.WrapTransient(err, "relational.projection_begin_tx",
			fmt.Sprintf("projection %q: begin tx", p.name))
	}

	defer func() { _ = tx.Rollback() }()

	sink := newSQLSink(tx, p.schema, p.dialect)

	if err := p.handler(ctx, evt, sink); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return errorfamily.WrapTransient(err, "relational.projection_commit",
			fmt.Sprintf("projection %q: commit", p.name))
	}

	return nil
}

// WithoutRelationalAutoMigrate skips CREATE TABLE IF NOT EXISTS at construction.
// Use it when the caller manages schema via external migrations.
func WithoutRelationalAutoMigrate() RelationalProjectionOption {
	return func(p *RelationalProjection) { p.autoMigrate = false }
}

// Reset implements [projectionhost.Resettable]. It clears all rows from every
// table in the projection's schema, so a subsequent replay rebuilds the read
// model from zero. Called automatically by projectionhost.Host.Reset — after
// Reset, the host drops the checkpoint and replays from the beginning.
func (p *RelationalProjection) Reset(ctx context.Context) error {
	for i := range p.schema.Tables {
		t := p.schema.Tables[i]

		if _, err := p.db.ExecContext(ctx, "DELETE FROM "+t.Name); err != nil {
			return errorfamily.WrapTransient(err, "relational.projection_reset",
				fmt.Sprintf("projection %q: delete from %s", p.name, t.Name))
		}
	}

	return nil
}

var (
	errRelationalNoName = errorfamily.NewRejection(
		"relational.no_name",
		"relational projection: name is required",
	)
	errRelationalNilDB = errorfamily.NewRejection(
		"relational.nil_db",
		"relational projection: db must not be nil",
	)
	errRelationalNilDialect = errorfamily.NewRejection(
		"relational.nil_dialect",
		"relational projection: dialect must not be nil",
	)
	errRelationalNilHandler = errorfamily.NewRejection(
		"relational.nil_handler",
		"relational projection: handler must not be nil",
	)
)

var _ cqrsprojection.Projection = (*RelationalProjection)(nil)

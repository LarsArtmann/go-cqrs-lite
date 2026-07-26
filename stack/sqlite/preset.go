package sqlite

import (
	"context"
	"database/sql"
	"io"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4/sqlopt"
	"github.com/larsartmann/go-cqrs-lite/storage/v4"
	cqrswatermill "github.com/larsartmann/go-cqrs-lite/watermill/v4"
)

// Option configures the SQLite preset.
type Option func(*config)

type config struct {
	sqlopt.DSNConfig
	sqlopt.PragmaConfig
}

func defaultConfig() config {
	return config{
		PragmaConfig: sqlopt.PragmaConfig{
			WAL:         true,
			Optimize:    false,
			ForeignKeys: false,
		},
		DSNConfig: sqlopt.DSNConfig{
			AutoMigrate: true,
			EventDSN:    "",
			QueryDSN:    "",
			ViewDSN:     "",
		},
	}
}

// WithPragmas applies shared SQLite PRAGMA options from sqlopt (WAL,
// optimizations, foreign keys):
//
//	b, _ := sqlite.New(dsn, sqlite.WithPragmas(
//	    sqlopt.WithoutWAL(),
//	    sqlopt.WithForeignKeys(),
//	))
func WithPragmas(opts ...sqlopt.PragmaOption) Option {
	return func(c *config) { sqlopt.ApplyTo(opts, &c.PragmaConfig) }
}

// WithDSN applies shared multi-database DSN options from sqlopt. Use this to
// configure event, query, or view database separation, or to disable
// auto-migration:
//
//	b, _ := sqlite.New(dsn, sqlite.WithDSN(
//	    sqlopt.WithoutAutoMigrate(),
//	    sqlopt.WithEventDB("events.db"),
//	    sqlopt.WithViewDB("views.db"),
//	))
func WithDSN(opts ...sqlopt.DSNOption) Option {
	return func(c *config) { sqlopt.ApplyTo(opts, &c.DSNConfig) }
}

// New opens a SQLite database at dsn, configures it, and returns a
// fully-wired [stack.Bundle].
//
// dsn is the SQLite connection string — typically a file path ("log.db") or
// ":memory:" for an ephemeral in-process database. The database is opened
// with the pure-Go modernc.org/sqlite driver (no CGo required).
//
// Events, commands, queries, snapshots, checkpoints, AND read models are all
// persisted to the database. The event bus uses watermill.EventBus (GoChannel-
// backed, in-process) since SQLite has no pub/sub semantics.
//
// On any setup failure the database is closed before the error is returned —
// no resource leaks. The returned Bundle owns the *sql.DB; Close releases it.
func New(dsn string, opts ...Option) (*stack.Bundle, error) {
	cfg := defaultConfig()

	for _, opt := range opts {
		opt(&cfg)
	}

	return newBundle(dsn, cfg)
}

func newBundle(dsn string, cfg config) (*stack.Bundle, error) {
	stackOpts, backend, sqlDB, _, err := sqlopt.InitStack(
		dsn,
		"sqlite",
		cfg.EventDSN,
		cfg.QueryDSN,
		func(d string) (*sql.DB, *storage.SQLBackend, error) { return openBackend(d, cfg) },
		func(d string) (*storage.SQLBackend, io.Closer, error) { return openSecondaryBackend(d, cfg) },
	)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "sqlite_preset.init_stack",
			"initialize SQLite stack")
	}

	// Bus is in-process GoChannel (SQLite has no pub/sub).
	stackOpts = append(stackOpts, stack.WithBus(cqrswatermill.NewEventBus()))

	bundle, err := sqlopt.FinalizeBundle(stackOpts, backend, sqlDB, "sqlite", cfg.ViewDSN,
		func(dsn string) (*sql.DB, error) { return openSecondaryDB(dsn, cfg) },
		storage.NewSQLiteBackend)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "sqlite_preset.finalize_bundle",
			"finalize SQLite bundle")
	}

	return bundle, nil
}

// openBackend opens the database, applies pragmas and schema, and returns
// both the *sql.DB (for lifecycle) and the SQLBackend (for store access).
func openBackend(dsn string, cfg config) (db *sql.DB, backend *storage.SQLBackend, err error) {
	db, err = sqlopt.OpenDBOrErr("sqlite", dsn, "sqlite_preset.open_primary")
	if err != nil {
		return nil, nil, err
	}

	defer func() {
		if err != nil && db != nil {
			_ = db.Close()
		}
	}()

	ctx := context.Background()

	if cfg.WAL {
		if err = storage.SQLiteEnableWAL(ctx, db); err != nil {
			return nil, nil, errorfamily.WrapInfrastructure(err, "sqlite_preset.enable_wal",
				"enable WAL mode")
		}
	}

	// SQLite WAL serializes writes; capping at 1 connection prevents
	// SQLITE_BUSY errors under concurrent access (see storage.ConfigureSQLitePool).
	storage.ConfigureSQLitePool(db)

	if cfg.ForeignKeys {
		if err = storage.SQLiteEnableForeignKeys(ctx, db); err != nil {
			return nil, nil, errorfamily.WrapInfrastructure(
				err,
				"sqlite_preset.enable_foreign_keys",
				"enable foreign keys",
			)
		}
	}

	if cfg.AutoMigrate {
		if err = storage.SQLiteInitSchema(ctx, db); err != nil {
			return nil, nil, errorfamily.WrapInfrastructure(err, "sqlite_preset.init_schema",
				"initialize sqlite schema")
		}
	}

	if cfg.Optimize {
		if err = storage.SQLiteApplyOptimizations(ctx, db); err != nil {
			return nil, nil, errorfamily.WrapInfrastructure(
				err,
				"sqlite_preset.apply_optimizations",
				"apply sqlite optimizations",
			)
		}
	}

	backend, err = storage.NewSQLiteBackend(db)
	if err != nil {
		return nil, nil, errorfamily.WrapInfrastructure(err, "sqlite_preset.create_backend",
			"create SQL backend")
	}

	return db, backend, nil
}

package mysql

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

type Option func(*config)

type config struct {
	sqlopt.DSNConfig

	extraStackOpts []stack.Option
}

func defaultConfig() config {
	return config{
		DSNConfig: sqlopt.DSNConfig{
			AutoMigrate: true,
			EventDSN:    "",
			QueryDSN:    "",
			ViewDSN:     "",
		},
		extraStackOpts: nil,
	}
}

// WithDSN configures multi-database topology (event/query/view DSN overrides)
// and auto-migration behavior.
func WithDSN(opts ...sqlopt.DSNOption) Option {
	return func(c *config) { sqlopt.ApplyTo(opts, &c.DSNConfig) }
}

// WithStack passes through additional [stack.Option] values (e.g.
// [stack.WithMetaEngine]).
func WithStack(opts ...stack.Option) Option {
	return func(c *config) { c.extraStackOpts = append(c.extraStackOpts, opts...) }
}

// New creates a [stack.Bundle] backed by MySQL/MariaDB. The dsn must be a
// valid go-sql-driver/mysql DSN (e.g.
// "user:password@tcp(localhost:3306)/dbname?parseTime=true").
//
// The bundle auto-migrates all CQRS tables unless [mysql.WithDSN] is called
// with [sqlopt.WithoutAutoMigrate].
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
		"mysql",
		cfg.EventDSN,
		cfg.QueryDSN,
		func(d string) (*sql.DB, *storage.SQLBackend, error) { return openBackend(d, cfg) },
		func(d string) (*storage.SQLBackend, io.Closer, error) { return openSecondaryBackend(d, cfg) },
	)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "mysql_preset.init_stack",
			"initialize MySQL stack")
	}

	stackOpts = append(stackOpts, stack.WithBus(cqrswatermill.NewEventBus()))
	stackOpts = append(stackOpts, cfg.extraStackOpts...)

	bundle, err := sqlopt.FinalizeBundle(stackOpts, backend, sqlDB, "mysql", cfg.ViewDSN,
		func(dsn string) (*sql.DB, error) { return openSecondaryDB(dsn, cfg) },
		storage.NewMySQLBackend)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "mysql_preset.finalize_bundle",
			"finalize MySQL bundle")
	}

	return bundle, nil
}

func openBackend(
	dsn string,
	cfg config,
) (*sql.DB, *storage.SQLBackend, error) {
	return sqlopt.OpenPrimaryBackend(
		func() (*sql.DB, error) {
			return sqlopt.OpenDBOrErr("mysql", dsn, "mysql_preset.open_primary")
		},
		func(ctx context.Context, sqlDB *sql.DB) error {
			if cfg.AutoMigrate {
				if err := storage.MySQLInitSchema(ctx, sqlDB); err != nil {
					return errorfamily.WrapInfrastructure(err, "mysql_preset.init_schema",
						"initialize MySQL schema")
				}
			}
			return nil
		},
		storage.NewMySQLBackend,
		"mysql_preset.create_backend",
	)
}

package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"strings"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4/sqlopt"
	"github.com/larsartmann/go-cqrs-lite/storage/v4"
	cqrswatermill "github.com/larsartmann/go-cqrs-lite/watermill/v4"
)

// Option configures the DuckDB preset.
type Option func(*config)

type config struct {
	sqlopt.DSNConfig
	duckdbConfig
}

// duckdbConfig holds DuckDB-specific settings.
type duckdbConfig struct {
	// Threads limits the number of DuckDB worker threads (0 = unlimited).
	// Maps to the DuckDB "threads" configuration option.
	Threads int

	// MemoryLimit caps DuckDB's memory usage (e.g. "1GB", "512MB").
	// Empty = DuckDB default (80% of available RAM). Maps to the DuckDB
	// "memory_limit" configuration option.
	MemoryLimit string
}

func defaultConfig() config {
	return config{
		DSNConfig: sqlopt.DSNConfig{
			AutoMigrate: true,
			EventDSN:    "",
			QueryDSN:    "",
			ViewDSN:     "",
		},
		duckdbConfig: duckdbConfig{
			Threads:     0,
			MemoryLimit: "",
		},
	}
}

// WithThreads limits the number of DuckDB worker threads.
// 0 (default) means unlimited (uses all available cores).
func WithThreads(n int) Option {
	return func(c *config) { c.Threads = n }
}

// WithMemoryLimit caps DuckDB's memory usage (e.g. "1GB", "512MB").
// Empty (default) lets DuckDB choose (80% of available RAM).
func WithMemoryLimit(limit string) Option {
	return func(c *config) { c.MemoryLimit = limit }
}

// WithDSN applies shared multi-database DSN options from sqlopt. Use this to
// configure event, query, or view database separation, or to disable
// auto-migration.
func WithDSN(opts ...sqlopt.DSNOption) Option {
	return func(c *config) { sqlopt.ApplyTo(opts, &c.DSNConfig) }
}

// New opens a DuckDB database at dsn, configures it, and returns a
// fully-wired [stack.Bundle].
//
// dsn is the DuckDB connection string: a file path ("analytics.db") for
// persistent storage, or "" (empty string) for an ephemeral in-memory database.
// Configuration options (threads, memory_limit, etc.) are passed as query
// parameters in the DSN (e.g. "?threads=4&memory_limit=1GB"), or via the
// [WithThreads] and [WithMemoryLimit] options which append them automatically.
//
// DuckDB is an embedded analytical (OLAP) engine. It excels at read-heavy
// analytical workloads, complex aggregations, and columnar scans. For
// write-heavy OLTP workloads, consider the SQLite or Postgres presets instead.
//
// Events, commands, queries, snapshots, checkpoints, AND read models are all
// persisted to the database. The event bus uses watermill.EventBus (GoChannel-
// backed, in-process) since DuckDB has no pub/sub semantics.
//
// # CGo Requirement
//
// DuckDB statically links a C++ engine. This module requires CGO_ENABLED=1 and
// a C/C++ compiler (gcc or clang). It is isolated in its own Go module so that
// consumers who do not import it never need CGo.
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
	dsn = appendDuckDBOptions(dsn, cfg)

	stackOpts, backend, sqlDB, _, err := sqlopt.InitStack(
		dsn,
		"duckdb",
		cfg.EventDSN,
		cfg.QueryDSN,
		func(d string) (*sql.DB, *storage.SQLBackend, error) { return openBackend(d, cfg) },
		func(d string) (*storage.SQLBackend, io.Closer, error) { return openSecondaryBackend(d, cfg) },
	)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "duckdb_preset.init_stack",
			"initialize DuckDB stack")
	}

	// Bus is in-process GoChannel (DuckDB has no pub/sub).
	stackOpts = append(stackOpts, stack.WithBus(cqrswatermill.NewEventBus()))

	bundle, err := sqlopt.FinalizeBundle(stackOpts, backend, sqlDB, "duckdb", cfg.ViewDSN,
		func(dsn string) (*sql.DB, error) { return openSecondaryDB(dsn, cfg) },
		storage.NewDuckDBBackend)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "duckdb_preset.finalize_bundle",
			"finalize DuckDB bundle")
	}

	return bundle, nil
}

// appendDuckDBOptions appends threads and memory_limit query parameters to the
// DSN if they were set via options and not already present in the DSN string.
func appendDuckDBOptions(dsn string, cfg config) string {
	if cfg.Threads > 0 {
		dsn = appendQueryParam(dsn, "threads", fmt.Sprintf("%d", cfg.Threads))
	}

	if cfg.MemoryLimit != "" {
		dsn = appendQueryParam(dsn, "memory_limit", cfg.MemoryLimit)
	}

	return dsn
}

// appendQueryParam appends a name=value pair to the DSN as a query parameter.
func appendQueryParam(dsn, name, value string) string {
	sep := "?"

	if strings.Contains(dsn, "?") {
		sep = "&"
	}

	return dsn + sep + name + "=" + value
}

// openBackend opens the database, runs schema migration, and returns both the
// *sql.DB (for lifecycle) and the SQLBackend (for store access).
func openBackend(
	dsn string,
	cfg config,
) (*sql.DB, *storage.SQLBackend, error) {
	return sqlopt.OpenPrimaryBackend(
		func() (*sql.DB, error) {
			return sqlopt.OpenDBOrErr("duckdb", dsn, "duckdb_preset.open_primary")
		},
		func(ctx context.Context, sqlDB *sql.DB) error {
			if cfg.AutoMigrate {
				if err := storage.DuckDBInitSchema(ctx, sqlDB); err != nil {
					return errorfamily.WrapInfrastructure(err, "duckdb_preset.init_schema",
						"initialize duckdb schema")
				}
			}

			return nil
		},
		storage.NewDuckDBBackend,
		"duckdb_preset.create_backend",
	)
}

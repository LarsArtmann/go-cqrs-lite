package turso

import (
	"context"
	"database/sql"
	"fmt"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/stack/v4/sqlopt"
	cqrsturso "github.com/larsartmann/go-cqrs-lite/storage/turso/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/v4"
)

// openLocalBackend opens the database, configures the pool, applies the schema,
// and returns both the *sql.DB (for lifecycle) and the Backend (for stores).
func openLocalBackend(
	dbPath string,
	cfg config,
) (*sql.DB, *storage.SQLBackend, error) {
	sqlDB, err := cqrsturso.Open(cqrsturso.DbPath(dbPath))
	if err != nil {
		return nil, nil, errorfamily.WrapInfrastructure(err, "turso.open",
			fmt.Sprintf("open %q", dbPath))
	}

	cqrsturso.ConfigurePool(sqlDB)

	if err := applySchemaAndPragmas(sqlDB, cfg); err != nil {
		//cqrs-lint:ignore(C023) library code or intentional pattern
		_ = sqlDB.Close()

		return nil, nil, err
	}

	backend, err := cqrsturso.NewBackend(sqlDB)
	if err != nil {
		//cqrs-lint:ignore(C023) library code or intentional pattern
		_ = sqlDB.Close()

		return nil, nil, errorfamily.WrapInfrastructure(err, "turso.create_backend",
			"create backend")
	}

	return sqlDB, backend, nil
}

// applySchemaAndPragmas runs schema initialization (with optional
// optimizations), then applies foreign keys if enabled. Shared by all
// database-opening paths (local, secondary, sync).
func applySchemaAndPragmas(sqlDB *sql.DB, cfg config) error {
	ctx := context.Background()

	if cfg.WAL {
		err := storage.SQLiteEnableWAL(ctx, sqlDB)
		if err != nil {
			return errorfamily.WrapInfrastructure(err, "turso.enable_wal",
				"enable WAL")
		}

		// Apply durability tier after WAL setup so the override takes
		// precedence over the NORMAL default.
		if err := sqlopt.ApplySQLiteDurability(ctx, sqlDB, cfg.durability); err != nil {
			return errorfamily.WrapInfrastructure(err, "turso.apply_durability",
				"apply durability tier")
		}
	}

	if cfg.AutoMigrate {
		var err error
		if cfg.Optimize {
			err = cqrsturso.InitSchemaWithIndexesAndOptimizations(ctx, sqlDB)
		} else {
			err = cqrsturso.InitSchema(ctx, sqlDB)
		}

		if err != nil {
			return errorfamily.WrapInfrastructure(err, "turso.init_schema",
				"init schema")
		}
	}

	if cfg.ForeignKeys {
		err := storage.SQLiteEnableForeignKeys(ctx, sqlDB)
		if err != nil {
			return errorfamily.WrapInfrastructure(err, "turso.enable_fk",
				"enable foreign keys")
		}
	}

	return nil
}

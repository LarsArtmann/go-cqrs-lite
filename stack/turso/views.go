package turso

import (
	"database/sql"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/v4"
)

// buildViewOptions creates read-model options from either a separate view DB
// (when cfg.ViewDSN is set) or the primary backend's KV store.
func buildViewOptions(
	cfg config,
	backend *storage.SQLBackend,
	sqlDB *sql.DB,
) ([]stack.Option, error) {
	if cfg.ViewDSN == "" {
		return buildPrimaryViewOptions(backend, sqlDB)
	}

	return buildSecondaryViewOptions(cfg, backend, sqlDB)
}

func buildPrimaryViewOptions(
	backend *storage.SQLBackend,
	sqlDB *sql.DB,
) ([]stack.Option, error) {
	kvStore, err := backend.KVStore()
	if err != nil {
		//cqrs-lint:ignore(C023) library code or intentional pattern
		_ = backend.Close()
		//cqrs-lint:ignore(C023) library code or intentional pattern
		_ = sqlDB.Close()

		return nil, errorfamily.WrapInfrastructure(err, "turso.kv_store",
			"create KV store")
	}

	return []stack.Option{stack.WithReadModels(kvStore)}, nil
}

func buildSecondaryViewOptions(
	cfg config,
	backend *storage.SQLBackend,
	sqlDB *sql.DB,
) ([]stack.Option, error) {
	viewDB, err := openSecondaryDB(cfg.ViewDSN, cfg)
	if err != nil {
		//cqrs-lint:ignore(C023) library code or intentional pattern
		_ = backend.Close()
		//cqrs-lint:ignore(C023) library code or intentional pattern
		_ = sqlDB.Close()

		return nil, errorfamily.WrapInfrastructure(err, "turso.open_view_db",
			"open view database")
	}

	viewBackend, err := storage.NewSQLiteBackend(viewDB)
	if err != nil {
		//cqrs-lint:ignore(C023) library code or intentional pattern
		_ = backend.Close()
		//cqrs-lint:ignore(C023) library code or intentional pattern
		_ = sqlDB.Close()
		//cqrs-lint:ignore(C023) library code or intentional pattern
		_ = viewDB.Close()

		return nil, errorfamily.WrapInfrastructure(err, "turso.create_view_backend",
			"create view backend")
	}

	kvStore, err := viewBackend.KVStore()
	if err != nil {
		//cqrs-lint:ignore(C023) library code or intentional pattern
		_ = viewBackend.Close()
		//cqrs-lint:ignore(C023) library code or intentional pattern
		_ = backend.Close()
		//cqrs-lint:ignore(C023) library code or intentional pattern
		_ = sqlDB.Close()

		return nil, errorfamily.WrapInfrastructure(err, "turso.view_kv_store",
			"create KV store for view database")
	}

	return []stack.Option{
		stack.WithReadModels(kvStore),
		stack.WithCloser(viewBackend),
		stack.WithCloser(stack.NewFuncCloser(viewDB.Close)),
	}, nil
}

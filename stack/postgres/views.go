package postgres

import (
	"database/sql"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/v4"
)

// buildViewOptions creates read-model options from either a separate view DB
// (when cfg.viewDSN is set) or the primary backend's KV store.
func buildViewOptions(
	cfg config,
	backend *storage.SQLBackend,
	sqlDB *sql.DB,
) ([]stack.Option, error) {
	if cfg.viewDSN == "" {
		return buildPrimaryViewOptions(backend, sqlDB)
	}

	return buildSecondaryViewOptions(cfg, backend, sqlDB)
}

func buildPrimaryViewOptions(backend *storage.SQLBackend, sqlDB *sql.DB) ([]stack.Option, error) {
	kvStore, err := backend.KVStore()
	if err != nil {
		_ = backend.Close()
		_ = sqlDB.Close()

		return nil, errorfamily.WrapInfrastructure(err, "postgres.kv_store", "create KV store")
	}

	return []stack.Option{stack.WithReadModels(kvStore)}, nil
}

func buildSecondaryViewOptions(
	cfg config,
	backend *storage.SQLBackend,
	sqlDB *sql.DB,
) ([]stack.Option, error) {
	viewDB, err := openSecondaryDB(cfg.viewDSN, cfg)
	if err != nil {
		_ = backend.Close()
		_ = sqlDB.Close()

		return nil, errorfamily.WrapInfrastructure(
			err,
			"postgres.open_view_db",
			"open view database",
		)
	}

	viewBackend, err := storage.NewSQLBackend(viewDB)
	if err != nil {
		_ = backend.Close()
		_ = sqlDB.Close()
		_ = viewDB.Close()

		return nil, errorfamily.WrapInfrastructure(
			err,
			"postgres.create_view_backend",
			"create view backend",
		)
	}

	kvStore, err := viewBackend.KVStore()
	if err != nil {
		_ = viewBackend.Close()
		_ = backend.Close()
		_ = sqlDB.Close()

		return nil, errorfamily.WrapInfrastructure(err, "postgres.view_kv_store",
			"create KV store for view database")
	}

	return []stack.Option{
		stack.WithReadModels(kvStore),
		stack.WithCloser(viewBackend),
		stack.WithCloser(stack.NewFuncCloser(viewDB.Close)),
	}, nil
}

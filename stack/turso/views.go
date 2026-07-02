package turso

import (
	"database/sql"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/stack/v3"
	"github.com/larsartmann/go-cqrs-lite/storage/v3"
)

// buildViewOptions creates read-model options from either a separate view DB
// (when cfg.viewPath is set) or the primary backend's KV store.
func buildViewOptions(
	cfg config,
	backend *storage.SQLBackend,
	sqlDB *sql.DB,
) ([]stack.Option, error) {
	if cfg.viewPath == "" {
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
		_ = backend.Close()
		_ = sqlDB.Close()

		return nil, event.WrapInfrastructure(err, "turso.kv_store",
			"create KV store")
	}

	return []stack.Option{stack.WithReadModels(kvStore)}, nil
}

func buildSecondaryViewOptions(
	cfg config,
	backend *storage.SQLBackend,
	sqlDB *sql.DB,
) ([]stack.Option, error) {
	viewDB, err := openSecondaryDB(cfg.viewPath, cfg)
	if err != nil {
		_ = backend.Close()
		_ = sqlDB.Close()

		return nil, event.WrapInfrastructure(err, "turso.open_view_db",
			"open view database")
	}

	viewBackend, err := storage.NewSQLiteBackend(viewDB)
	if err != nil {
		_ = backend.Close()
		_ = sqlDB.Close()
		_ = viewDB.Close()

		return nil, event.WrapInfrastructure(err, "turso.create_view_backend",
			"create view backend")
	}

	kvStore, err := viewBackend.KVStore()
	if err != nil {
		_ = viewBackend.Close()
		_ = backend.Close()
		_ = sqlDB.Close()

		return nil, event.WrapInfrastructure(err, "turso.view_kv_store",
			"create KV store for view database")
	}

	return []stack.Option{
		stack.WithReadModels(kvStore),
		stack.WithCloser(viewBackend),
		stack.WithCloser(stack.NewFuncCloser(viewDB.Close)),
	}, nil
}

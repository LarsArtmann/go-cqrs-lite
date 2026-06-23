package postgres

import (
	"database/sql"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/stack/v3"
	"github.com/larsartmann/go-cqrs-lite/storage/v3"
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

		return nil, fmt.Errorf("postgres preset: kv store: %w", err)
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

		return nil, fmt.Errorf("postgres preset: open view db: %w", err)
	}

	viewBackend, err := storage.NewSQLBackend(viewDB)
	if err != nil {
		_ = backend.Close()
		_ = sqlDB.Close()
		_ = viewDB.Close()

		return nil, fmt.Errorf("postgres preset: create view backend: %w", err)
	}

	kvStore, err := viewBackend.KVStore()
	if err != nil {
		_ = viewBackend.Close()
		_ = backend.Close()
		_ = sqlDB.Close()

		return nil, fmt.Errorf("postgres preset: kv store (view sqlDB): %w", err)
	}

	return []stack.Option{
		stack.WithReadModels(kvStore),
		stack.WithCloser(viewBackend),
		stack.WithCloser(&funcCloser{fn: viewDB.Close}),
	}, nil
}

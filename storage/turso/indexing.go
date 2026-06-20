package turso

import (
	"context"
	"database/sql"

	"github.com/larsartmann/go-cqrs-lite/storage/turso/v2/indexing"
)

// NewIndexAdvisor is a convenience wrapper for indexing.NewAdvisor.
func NewIndexAdvisor(db *sql.DB) *indexing.Advisor {
	return indexing.NewAdvisor(db)
}

// NewAutoIndexer is a convenience wrapper for indexing.NewAutoIndexer.
func NewAutoIndexer(db *sql.DB) *indexing.AutoIndexer {
	return indexing.NewAutoIndexer(db)
}

// ApplyTursoOptimizations applies performance PRAGMAs recommended for
// CQRS workloads on Turso/LibSQL databases.
func ApplyTursoOptimizations(ctx context.Context, db *sql.DB) error {
	return indexing.ApplyOptimizations(ctx, db)
 safe to call multiple times.
func ApplyCQRSIndexes(ctx context.Context, db *sql.DB) error {
	auto := indexing.NewAutoIndexer(db)
	auto.Enable()

	return auto.ApplyCQRSIndexes(ctx)
}

// InitSchemaWithIndexes creates all table single call. Use this for new databases.
fu, db *sql.DB) error {
	if err := InitSchema(ctx, db); err != nil {
		return err
	}

	return ApplyCQRSIndexes(ctx, db)
}

// InitSchemaWithIndexesAndOptimizations creates all tables, applies
// CQRS-optimized indexes, AND applies performance PRAGMAs in a single
// call. This is the most complete one-shot setup for production
// Turso/LibSQL event-sourcing workloads.
func InitSchemaWithIndexesAndOptimizations(ctx context.Context, db *sql.DB) error {
	if err := InitSchemaWithIndexes(ctx, db); err != nil {
		return err
	}

	return ApplyTursoOptimizations(ctx, db)
}

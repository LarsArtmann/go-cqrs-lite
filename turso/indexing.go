package turso

import (
	"context"
	"database/sql"

	"github.com/larsartmann/go-cqrs-lite/turso/v2/indexing"
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
	return indexing.ApplyOptimizations(ctx, db) //nolint:wrapcheck // transparent delegation
}

// ApplyCQRSIndexes creates the recommended CQRS-optimized indexes.
// Idempotent — safe to call multiple times.
func ApplyCQRSIndexes(ctx context.Context, db *sql.DB) error {
	return indexing.NewAutoIndexer(db).ApplyCQRSIndexes(ctx) //nolint:wrapcheck // transparent delegation
}

// InitSchemaWithIndexes creates all tables AND applies CQRS-optimized
// indexes in a single call. Use this for new databases.
func InitSchemaWithIndexes(ctx context.Context, db *sql.DB) error {
	if err := InitSchema(ctx, db); err != nil {
		return err
	}

	return ApplyCQRSIndexes(ctx, db)
}

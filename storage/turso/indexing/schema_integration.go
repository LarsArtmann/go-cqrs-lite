package indexing

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
)

// SchemaChangeHook returns a hook that re-analyzes indexes when schema
// version changes are detected. This bridges the indexing advisor with
// schema evolution workflows.
//
// Usage:
//
//	auto := indexing.NewAutoIndexer(db,
//	    indexing.WithIndexingHooks(
//	        indexing.WithAfterCreateHook(indexing.SchemaChangeHook()),
//	    ),
//	)
//
// The hook is a no-op if the context has no version information.
func SchemaChangeHook() Hook {
	return func(ctx context.Context, hctx HookContext) error {
		return analyzeAfterSchemaChange(ctx, hctx.AutoIndexer)
	}
}

func analyzeAfterSchemaChange(ctx context.Context, a *AutoIndexer) error {
	if a == nil {
		return nil
	}

	if !a.IsEnabled() {
		return nil
	}

	recs, err := a.Recommendations(ctx)
	if err != nil {
		return event.WrapInfrastructure(err, "indexing.schema_change",
			"analyze indexes after schema change")
	}

	if len(recs) == 0 {
		return nil
	}

	return a.Apply(ctx, recs)
}

// MigrateWithIndexing runs a schema migration (via the provided function)
// and then re-analyzes indexes for the affected tables. This is the
// recommended way to handle DDL changes that might affect query plans.
func MigrateWithIndexing(
	ctx context.Context,
	db *sql.DB,
	autoIndexer *AutoIndexer,
	migration func(ctx context.Context, tx *sql.Tx) error,
) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return event.WrapInfrastructure(err, "indexing.migrate",
			"begin migration transaction")
	}

	defer func() { _ = tx.Rollback() }()

	if err := migration(ctx, tx); err != nil {
		return event.WrapInfrastructure(err, "indexing.migrate",
			fmt.Sprintf("run migration: %v", err))
	}

	if err := tx.Commit(); err != nil {
		return event.WrapInfrastructure(err, "indexing.migrate",
			"commit migration")
	}

	if autoIndexer != nil && autoIndexer.IsEnabled() {
		if err := analyzeAfterSchemaChange(ctx, autoIndexer); err != nil {
			return event.WrapInfrastructure(err, "indexing.migrate",
				"analyze indexes after migration")
		}
	}

	return nil
}

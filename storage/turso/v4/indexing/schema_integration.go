package indexing

import (
	"context"
	"database/sql"
	"fmt"

	errorfamily "github.com/larsartmann/go-error-family"
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
// The hook runs once per index creation and never starts a nested analysis
// of its own: while an analysis is running, the hooks it triggers do nothing
// and the remaining indexes are created by the running analysis. The hook is
// not fired in dry-run mode, where nothing is created, and does nothing while
// the auto-indexer is disabled.
func SchemaChangeHook() Hook {
	return func(ctx context.Context, hctx HookContext) error {
		return analyzeAfterSchemaChange(ctx, hctx.AutoIndexer)
	}
}

func analyzeAfterSchemaChange(ctx context.Context, indexer *AutoIndexer) error {
	if indexer == nil || !indexer.beginAnalysis() {
		return nil
	}
	defer indexer.endAnalysis()

	if !indexer.IsEnabled() {
		return nil
	}

	recs, err := indexer.Recommendations(ctx)
	if err != nil {
		return errorfamily.WrapInfrastructure(err, "indexing.schema_change",
			"analyze indexes after schema change")
	}

	if len(recs) == 0 {
		return nil
	}

	return indexer.Apply(ctx, recs)
}

// beginAnalysis reports whether the caller may run a schema-change analysis.
// It is false while another one is in progress on this auto-indexer, which is
// what stops the after-create hook from recursing into Apply.
func (a *AutoIndexer) beginAnalysis() bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.analyzing {
		return false
	}
	a.analyzing = true

	return true
}

func (a *AutoIndexer) endAnalysis() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.analyzing = false
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
		return errorfamily.WrapInfrastructure(err, "indexing.migrate",
			"begin migration transaction")
	}

	defer func() { _ = tx.Rollback() }()

	if err := migration(ctx, tx); err != nil {
		return errorfamily.WrapInfrastructure(err, "indexing.migrate",
			fmt.Sprintf("run migration: %v", err))
	}

	if err := tx.Commit(); err != nil {
		return errorfamily.WrapInfrastructure(err, "indexing.migrate",
			"commit migration")
	}

	if autoIndexer != nil && autoIndexer.IsEnabled() {
		if err := analyzeAfterSchemaChange(ctx, autoIndexer); err != nil {
			return errorfamily.WrapInfrastructure(err, "indexing.migrate",
				"analyze indexes after migration")
		}
	}

	return nil
}

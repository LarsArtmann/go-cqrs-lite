package indexing_test

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/storage/turso/v4/indexing"
)

func TestSchemaChangeHook_DryRunTerminates(t *testing.T) {
	t.Parallel()

	db := setupIndexingDB(t)
	auto := indexing.NewAutoIndexer(
		db,
		indexing.WithDryRun(),
		indexing.WithIndexingHooks(
			indexing.WithAfterCreateHook(indexing.SchemaChangeHook()),
		),
	)
	auto.Enable()

	if err := auto.ApplyCQRSIndexes(context.Background()); err != nil {
		t.Fatalf("ApplyCQRSIndexes: %v", err)
	}

	if len(auto.LastDDL()) != len(indexing.RecommendedCQRSIndexes()) {
		t.Errorf("captured %d DDL statements, want %d", len(auto.LastDDL()),
			len(indexing.RecommendedCQRSIndexes()))
	}
}

func TestSchemaChangeHook_FiresOncePerCreatedIndex(t *testing.T) {
	t.Parallel()

	db := setupIndexingDB(t)

	var afterCreate atomic.Int32
	auto := indexing.NewAutoIndexer(
		db,
		indexing.WithIndexingHooks(
			indexing.WithAfterCreateHook(indexing.SchemaChangeHook()),
			indexing.WithAfterCreateHook(func(context.Context, indexing.HookContext) error {
				afterCreate.Add(1)
				return nil
			}),
		),
	)
	auto.Enable()

	if err := auto.ApplyCQRSIndexes(context.Background()); err != nil {
		t.Fatalf("ApplyCQRSIndexes: %v", err)
	}

	if want := int32(len(indexing.RecommendedCQRSIndexes())); afterCreate.Load() != want {
		t.Errorf("after-create fired %d times, want %d (one per index, no nested Apply)",
			afterCreate.Load(), want)
	}
}

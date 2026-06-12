package turso_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/turso/v2"
	"github.com/larsartmann/go-cqrs-lite/turso/v2/indexing"
)

func setupIndexingTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := turso.OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	if err := turso.InitSchema(context.Background(), db); err != nil {
		t.Fatalf("InitSchema: %v", err)
	}

	return db
}

func TestNewIndexAdvisor(t *testing.T) {
	t.Parallel()

	db := setupIndexingTestDB(t)
	advisor := turso.NewIndexAdvisor(db)

	if advisor == nil {
		t.Fatal("expected non-nil advisor")
	}
}

func TestNewAutoIndexer(t *testing.T) {
	t.Parallel()

	db := setupIndexingTestDB(t)
	auto := turso.NewAutoIndexer(db)

	if auto == nil {
		t.Fatal("expected non-nil auto-indexer")
	}
}

func TestApplyTursoOptimizations(t *testing.T) {
	t.Parallel()

	db := setupIndexingTestDB(t)

	if err := turso.ApplyTursoOptimizations(context.Background(), db); err != nil {
		t.Fatalf("ApplyTursoOptimizations: %v", err)
	}
}

func TestApplyCQRSIndexes(t *testing.T) {
	t.Parallel()

	db := setupIndexingTestDB(t)

	if err := turso.ApplyCQRSIndexes(context.Background(), db); err != nil {
		t.Fatalf("ApplyCQRSIndexes: %v", err)
	}

	advisor := indexing.NewAdvisor(db)
	if err := advisor.ExistingIndexes(context.Background()); err != nil {
		t.Fatalf("ExistingIndexes: %v", err)
	}

	for _, idx := range indexing.RecommendedCQRSIndexes() {
		if !advisor.HasIndex(idx.Name) {
			t.Errorf("expected index %s to exist", idx.Name)
		}
	}
}

func TestInitSchemaWithIndexes(t *testing.T) {
	t.Parallel()

	db, err := turso.OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	if err := turso.InitSchemaWithIndexes(context.Background(), db); err != nil {
		t.Fatalf("InitSchemaWithIndexes: %v", err)
	}

	advisor := indexing.NewAdvisor(db)
	if err := advisor.ExistingIndexes(context.Background()); err != nil {
		t.Fatalf("ExistingIndexes: %v", err)
	}

	if !advisor.HasIndex("idx_events_cursor") {
		t.Error("expected idx_events_cursor after InitSchemaWithIndexes")
	}
}

func TestInitSchemaWithIndexesAndOptimizations(t *testing.T) {
	t.Parallel()

	db, err := turso.OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	if err := turso.InitSchemaWithIndexesAndOptimizations(context.Background(), db); err != nil {
		t.Fatalf("InitSchemaWithIndexesAndOptimizations: %v", err)
	}

	advisor := indexing.NewAdvisor(db)
	if err := advisor.ExistingIndexes(context.Background()); err != nil {
		t.Fatalf("ExistingIndexes: %v", err)
	}

	if !advisor.HasIndex("idx_events_cursor") {
		t.Error("expected idx_events_cursor after InitSchemaWithIndexesAndOptimizations")
	}

	if !advisor.HasIndex("idx_events_agg_ver") {
		t.Error("expected idx_events_agg_ver after InitSchemaWithIndexesAndOptimizations")
	}
}

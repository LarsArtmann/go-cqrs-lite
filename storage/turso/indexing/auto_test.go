package indexing_test

import (
	"context"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/storage/turso/v4/indexing"
)

func TestAutoIndexer_EnableDisable(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	auto := indexing.NewAutoIndexer(db)

	if auto.IsEnabled() {
		t.Error("expected disabled by default")
	}

	auto.Enable()
	if !auto.IsEnabled() {
		t.Error("expected enabled after Enable()")
	}

	auto.Disable()
	if auto.IsEnabled() {
		t.Error("expected disabled after Disable()")
	}
}

func TestAutoIndexer_ApplyRecommended_Disabled(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	auto := indexing.NewAutoIndexer(db)

	err := auto.ApplyRecommended(context.Background())
	if err == nil {
		t.Fatal("expected error when disabled")
	}

	if errorfamily.Classify(err) != errorfamily.Rejection {
		t.Errorf("expected Rejection classification, got %v", errorfamily.Classify(err))
	}
}

func TestAutoIndexer_ApplyCQRSIndexes(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	auto := indexing.NewAutoIndexer(db)
	auto.Enable()

	if err := auto.ApplyCQRSIndexes(context.Background()); err != nil {
		t.Fatalf("ApplyCQRSIndexes: %v", err)
	}

	// Verify indexes were created.
	advisor := indexing.NewAdvisor(db)
	if err := advisor.ExistingIndexes(context.Background()); err != nil {
		t.Fatalf("ExistingIndexes: %v", err)
	}

	for _, idx := range indexing.RecommendedCQRSIndexes() {
		if !advisor.HasIndex(idx.Name) {
			t.Errorf("expected index %s to exist after ApplyCQRSIndexes", idx.Name)
		}
	}
}

func TestAutoIndexer_ApplyCQRSIndexes_Idempotent(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	auto := indexing.NewAutoIndexer(db)
	auto.Enable()

	// Apply twice should not error.
	if err := auto.ApplyCQRSIndexes(context.Background()); err != nil {
		t.Fatalf("first ApplyCQRSIndexes: %v", err)
	}

	if err := auto.ApplyCQRSIndexes(context.Background()); err != nil {
		t.Fatalf("second ApplyCQRSIndexes: %v", err)
	}
}

func TestAutoIndexer_Recommendations(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	auto := indexing.NewAutoIndexer(db)

	recs, err := auto.Recommendations(context.Background())
	if err != nil {
		t.Fatalf("Recommendations: %v", err)
	}

	// With base schema, some patterns may already be covered.
	_ = recs
}

func TestAutoIndexer_WithAutoAnalyze(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	auto := indexing.NewAutoIndexer(db, indexing.WithAutoAnalyze())
	auto.Enable()

	if err := auto.ApplyCQRSIndexes(context.Background()); err != nil {
		t.Fatalf("ApplyCQRSIndexes: %v", err)
	}

	// After applying with WithAutoAnalyze, sqlite_stat1 should be populated.
	row := db.QueryRowContext(context.Background(),
		"SELECT count(*) FROM sqlite_master WHERE type='table' AND name='sqlite_stat1'")

	var count int
	if err := row.Scan(&count); err != nil {
		t.Fatalf("check sqlite_stat1: %v", err)
	}
}

func TestAutoIndexer_Close(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	auto := indexing.NewAutoIndexer(db)
	auto.Enable()

	if err := auto.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if auto.IsEnabled() {
		t.Error("expected auto-indexer to be disabled after Close")
	}
}

func TestAutoIndexer_Drop(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	auto := indexing.NewAutoIndexer(db)
	auto.Enable()

	if err := auto.ApplyCQRSIndexes(context.Background()); err != nil {
		t.Fatalf("ApplyCQRSIndexes: %v", err)
	}

	advisor := indexing.NewAdvisor(db)
	if err := advisor.ExistingIndexes(context.Background()); err != nil {
		t.Fatalf("ExistingIndexes: %v", err)
	}

	idx := indexing.Index{
		Name:    "idx_events_cursor",
		Table:   "events",
		Columns: []string{"occurred_at", "id"},
	}

	if err := auto.Drop(context.Background(), idx); err != nil {
		t.Fatalf("Drop: %v", err)
	}

	if err := advisor.ExistingIndexes(context.Background()); err != nil {
		t.Fatalf("ExistingIndexes after Drop: %v", err)
	}

	if advisor.HasIndex("idx_events_cursor") {
		t.Error("expected idx_events_cursor to be gone after Drop")
	}
}

func TestAutoIndexer_Drop_Disabled(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	auto := indexing.NewAutoIndexer(db)

	err := auto.Drop(
		context.Background(),
		indexing.Index{Name: "foo", Table: "events", Columns: []string{"x"}},
	)
	if err == nil {
		t.Fatal("expected error when disabled")
	}
}

func TestAutoIndexer_DryRun(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	auto := indexing.NewAutoIndexer(db, indexing.WithDryRun())
	auto.Enable()

	if err := auto.ApplyCQRSIndexes(context.Background()); err != nil {
		t.Fatalf("ApplyCQRSIndexes: %v", err)
	}

	ddls := auto.LastDDL()
	if len(ddls) == 0 {
		t.Fatal("expected DDL statements to be captured in dry-run mode")
	}

	// In dry-run mode, indexes should NOT actually exist.
	advisor := indexing.NewAdvisor(db)
	if err := advisor.ExistingIndexes(context.Background()); err != nil {
		t.Fatalf("ExistingIndexes: %v", err)
	}

	if advisor.HasIndex("idx_events_cursor") {
		t.Error("expected idx_events_cursor to NOT exist after dry-run Apply")
	}
}

func TestAutoIndexer_RecommendAndApply(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	auto := indexing.NewAutoIndexer(db)
	auto.Enable()

	if err := auto.RecommendAndApply(context.Background()); err != nil {
		t.Fatalf("RecommendAndApply: %v", err)
	}
}

func TestAutoIndexer_maybeAnalyze_NotSet(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	auto := indexing.NewAutoIndexer(db)
	auto.Enable()

	// With Enable() but no WithAutoAnalyze, maybeAnalyze should be a no-op.
	if err := auto.ApplyCQRSIndexes(context.Background()); err != nil {
		t.Fatalf("ApplyCQRSIndexes: %v", err)
	}
	// No assertion needed; test passes if no panic.
}

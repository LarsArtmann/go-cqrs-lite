package indexing_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/turso/v2/indexing"
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

	if event.Classify(err) != event.Rejection {
		t.Errorf("expected Rejection classification, got %v", event.Classify(err))
	}
}

func TestAutoIndexer_ApplyCQRSIndexes(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	auto := indexing.NewAutoIndexer(db)

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

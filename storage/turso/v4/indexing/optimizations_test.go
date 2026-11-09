package indexing_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/storage/turso/v4/indexing"
)

func TestDefaultOptimizations(t *testing.T) {
	t.Parallel()

	pragmas := indexing.DefaultOptimizations()
	if len(pragmas) == 0 {
		t.Fatal("expected non-empty default optimizations")
	}

	hasWAL := false
	for _, p := range pragmas {
		if p.Name == "journal_mode" && p.Value == "WAL" {
			hasWAL = true
		}
	}

	if !hasWAL {
		t.Error("expected WAL mode in default optimizations")
	}
}

func TestApplyOptimizations(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)

	if err := indexing.ApplyOptimizations(context.Background(), db); err != nil {
		t.Fatalf("ApplyOptimizations: %v", err)
	}
}

func TestApplyWAL(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)

	if err := indexing.ApplyWAL(context.Background(), db); err != nil {
		t.Fatalf("ApplyWAL: %v", err)
	}
}

func TestSetCacheSize(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)

	if err := indexing.SetCacheSize(context.Background(), db, -64000); err != nil {
		t.Fatalf("SetCacheSize: %v", err)
	}
}

func TestSetMemoryMap(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)

	if err := indexing.SetMemoryMap(context.Background(), db, 268435456); err != nil {
		t.Fatalf("SetMemoryMap: %v", err)
	}
}

func TestRunOptimize(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)

	if err := indexing.RunOptimize(context.Background(), db); err != nil {
		t.Fatalf("RunOptimize: %v", err)
	}
}

func TestAnalyze(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)

	if err := indexing.Analyze(context.Background(), db); err != nil {
		t.Fatalf("Analyze: %v", err)
	}
}

func TestAnalyzeTable(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)

	if err := indexing.AnalyzeTable(context.Background(), db, "events"); err != nil {
		t.Fatalf("AnalyzeTable: %v", err)
	}
}

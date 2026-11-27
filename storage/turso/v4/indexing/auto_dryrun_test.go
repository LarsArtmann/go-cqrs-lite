package indexing_test

import (
	"context"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/storage/turso/v4/indexing"
)

func TestAutoIndexer_Drop_DryRunKeepsIndex(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	ctx := context.Background()
	auto := indexing.NewAutoIndexer(db, indexing.WithDryRun())
	auto.Enable()

	idx := indexing.Index{Name: "idx_events_cursor", Table: "events", Columns: []string{"occurred_at", "id"}}
	if _, err := db.ExecContext(ctx, idx.DDL()); err != nil {
		t.Fatalf("create index: %v", err)
	}

	if err := auto.Drop(ctx, idx); err != nil {
		t.Fatalf("Drop: %v", err)
	}

	advisor := indexing.NewAdvisor(db)
	if err := advisor.ExistingIndexes(ctx); err != nil {
		t.Fatalf("ExistingIndexes: %v", err)
	}

	if !advisor.HasIndex(idx.Name) {
		t.Errorf("dry-run Drop removed %s from the database", idx.Name)
	}

	var captured bool
	for _, ddl := range auto.LastDDL() {
		if strings.Contains(ddl, "DROP INDEX") {
			captured = true
		}
	}

	if !captured {
		t.Error("dry-run Drop did not report its DROP INDEX statement in LastDDL")
	}
}

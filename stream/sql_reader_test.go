package stream_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/stream"
)

func TestSQLAggregateReader_InvalidPrefix(t *testing.T) {
	t.Parallel()

	_, err := stream.NewSQLAggregateReader(nil, "INVALID")
	if err == nil {
		t.Error("expected error for invalid prefix")
	}
}

func TestSQLAggregateReader_ValidPrefix(t *testing.T) {
	t.Parallel()

	db := newSQLiteDB(t)
	defer db.Close()

	_, err := stream.NewSQLAggregateReader(db, "myapp_")
	if err != nil {
		t.Fatalf("valid prefix should succeed: %v", err)
	}
}

func TestSQLAggregateReader_EmptyTable(t *testing.T) {
	t.Parallel()

	db := newSQLiteDB(t)
	defer db.Close()

	_, _ = stream.NewAggregateProjection(db, "test_")
	reader, _ := stream.NewSQLAggregateReader(db, "test_")

	page, err := reader.List(context.Background(), stream.ListOptions{Type: "User"})
	if err != nil {
		t.Fatal(err)
	}

	if len(page.Items) != 0 {
		t.Errorf("got %d items, want 0", len(page.Items))
	}
}

func TestSQLAggregateReader_RequiresType(t *testing.T) {
	t.Parallel()

	db := newSQLiteDB(t)
	defer db.Close()

	reader, _ := stream.NewSQLAggregateReader(db, "test_")

	_, err := reader.List(context.Background(), stream.ListOptions{})
	if err == nil {
		t.Error("expected error when Type is empty")
	}
}

func TestSQLAggregateReader_TombstoneFiltering(t *testing.T) {
	t.Parallel()

	db := newSQLiteDB(t)

	proj, _ := stream.NewAggregateProjection(db, "test_")
	reader, _ := stream.NewSQLAggregateReader(db, "test_")

	ctx := context.Background()

	activeID := id.NewAggregateID()
	activeEvt := newProjEvent(t, "user.created", "User", activeID, 1, nil)
	if err := proj.Handle(ctx, activeEvt); err != nil {
		t.Fatal(err)
	}

	deletedID := id.NewAggregateID()
	tombstoneEvt := newProjEvent(t, "user.deleted", "User", deletedID, 1,
		map[string]string{string(event.MetadataKeyTombstone): "true"})
	if err := proj.Handle(ctx, tombstoneEvt); err != nil {
		t.Fatal(err)
	}

	page, err := reader.ListWithStatus(ctx, stream.ListOptions{
		Type:      "User",
		Tombstone: stream.TombstoneInclude,
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(page.Items) != 2 {
		t.Fatalf("got %d items, want 2 (all)", len(page.Items))
	}

	excluded, _ := reader.List(ctx, stream.ListOptions{Type: "User"})
	if len(excluded.Items) != 1 {
		t.Errorf("exclude: got %d items, want 1", len(excluded.Items))
	}

	onlyDeleted, _ := reader.List(ctx, stream.ListOptions{
		Type:      "User",
		Tombstone: stream.TombstoneOnly,
	})
	if len(onlyDeleted.Items) != 1 {
		t.Errorf("only deleted: got %d items, want 1", len(onlyDeleted.Items))
	}
}

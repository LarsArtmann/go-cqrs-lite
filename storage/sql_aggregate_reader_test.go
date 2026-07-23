package storage

import (
	"context"
	"database/sql"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/listing/v4"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

func openSQLiteListingDB(t *testing.T) (*sql.DB, *StreamProjection) {
	t.Helper()

	db, err := OpenSQLiteInMemory()
	if err != nil {
		t.Fatalf("OpenSQLiteInMemory: %v", err)
	}

	proj, err := NewStreamProjection(context.Background(), db, "test_", sqlpkg.SQLiteDialect{})
	if err != nil {
		t.Fatalf("NewStreamProjection: %v", err)
	}

	return db, proj
}

func seedListingAggregates(
	t *testing.T,
	proj *StreamProjection,
	aggType id.StreamType,
	count int,
) []id.StreamID {
	t.Helper()
	ctx := context.Background()
	ids := make([]id.StreamID, count)

	for i := range count {
		aggID := id.NewAggregateID()
		ids[i] = aggID

		evt, err := event.NewEvent(
			event.Type("test.created"), aggID, aggType,
			event.Version(1), []byte(`{}`),
		)
		if err != nil {
			t.Fatalf("NewEvent: %v", err)
		}

		err = proj.Handle(ctx, evt)
		if err != nil {
			t.Fatalf("Handle: %v", err)
		}
	}

	return ids
}

func TestNewSQLStreamReader_InvalidPrefix(t *testing.T) {
	t.Parallel()
	db, _ := openSQLiteListingDB(t)
	defer func() { _ = db.Close() }()

	_, err := NewSQLStreamReader(db, "INVALID", sqlpkg.SQLiteDialect{})
	if err == nil {
		t.Fatal("expected error for invalid prefix")
	}
}

func TestSQLStreamReader_List_Empty(t *testing.T) {
	t.Parallel()
	db, _ := openSQLiteListingDB(t)
	defer func() { _ = db.Close() }()

	reader, err := NewSQLStreamReader(db, "test_", sqlpkg.SQLiteDialect{})
	if err != nil {
		t.Fatalf("NewSQLStreamReader: %v", err)
	}

	page, err := reader.List(context.Background(), listing.ListOptions{
		Type: "User",
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(page.Items) != 0 {
		t.Fatalf("expected empty page, got %d items", len(page.Items))
	}

	if page.HasMore {
		t.Fatal("expected HasMore=false")
	}
}

func TestSQLStreamReader_List_WithAggregates(t *testing.T) {
	t.Parallel()
	db, proj := openSQLiteListingDB(t)
	defer func() { _ = db.Close() }()

	seedListingAggregates(t, proj, "User", 3)

	reader, err := NewSQLStreamReader(db, "test_", sqlpkg.SQLiteDialect{})
	if err != nil {
		t.Fatalf("NewSQLStreamReader: %v", err)
	}

	page, err := reader.List(context.Background(), listing.ListOptions{
		Type: "User",
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(page.Items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(page.Items))
	}

	for _, ref := range page.Items {
		if ref.Type != "User" {
			t.Fatalf("expected type User, got %s", ref.Type)
		}
	}
}

func TestSQLStreamReader_List_FilterByType(t *testing.T) {
	t.Parallel()
	db, proj := openSQLiteListingDB(t)
	defer func() { _ = db.Close() }()

	seedListingAggregates(t, proj, "User", 2)
	seedListingAggregates(t, proj, "Order", 3)

	reader, err := NewSQLStreamReader(db, "test_", sqlpkg.SQLiteDialect{})
	if err != nil {
		t.Fatalf("NewSQLStreamReader: %v", err)
	}

	userPage, err := reader.List(context.Background(), listing.ListOptions{Type: "User"})
	if err != nil {
		t.Fatalf("List User: %v", err)
	}

	if len(userPage.Items) != 2 {
		t.Fatalf("expected 2 users, got %d", len(userPage.Items))
	}

	orderPage, err := reader.List(context.Background(), listing.ListOptions{Type: "Order"})
	if err != nil {
		t.Fatalf("List Order: %v", err)
	}

	if len(orderPage.Items) != 3 {
		t.Fatalf("expected 3 orders, got %d", len(orderPage.Items))
	}
}

func TestSQLStreamReader_List_Pagination(t *testing.T) {
	t.Parallel()
	db, proj := openSQLiteListingDB(t)
	defer func() { _ = db.Close() }()

	seedListingAggregates(t, proj, "User", 5)

	reader, err := NewSQLStreamReader(db, "test_", sqlpkg.SQLiteDialect{})
	if err != nil {
		t.Fatalf("NewSQLStreamReader: %v", err)
	}

	page1, err := reader.List(context.Background(), listing.ListOptions{
		Type:  "User",
		Limit: 2,
	})
	if err != nil {
		t.Fatalf("List page1: %v", err)
	}

	if len(page1.Items) != 2 {
		t.Fatalf("expected 2 items on page1, got %d", len(page1.Items))
	}

	if !page1.HasMore {
		t.Fatal("expected HasMore=true on page1")
	}

	page2, err := reader.List(context.Background(), listing.ListOptions{
		Type:  "User",
		Limit: 2,
		After: page1.Items[1].ID,
	})
	if err != nil {
		t.Fatalf("List page2: %v", err)
	}

	if len(page2.Items) != 2 {
		t.Fatalf("expected 2 items on page2, got %d", len(page2.Items))
	}

	if !page2.HasMore {
		t.Fatal("expected HasMore=true on page2")
	}

	page3, err := reader.List(context.Background(), listing.ListOptions{
		Type:  "User",
		Limit: 2,
		After: page2.Items[1].ID,
	})
	if err != nil {
		t.Fatalf("List page3: %v", err)
	}

	if len(page3.Items) != 1 {
		t.Fatalf("expected 1 item on page3, got %d", len(page3.Items))
	}

	if page3.HasMore {
		t.Fatal("expected HasMore=false on page3")
	}
}

func TestSQLStreamReader_List_TombstoneFilter(t *testing.T) {
	t.Parallel()
	db, proj := openSQLiteListingDB(t)
	defer func() { _ = db.Close() }()

	ctx := context.Background()
	aggID := id.NewAggregateID()

	evt, _ := event.NewEvent(
		event.Type("user.created"), aggID, id.StreamType("User"),
		event.Version(1), []byte(`{}`),
	)
	_ = proj.Handle(ctx, evt)

	tombstoneEvt, err := event.NewEvent(
		event.Type("user.deleted"), aggID, id.StreamType("User"),
		event.Version(2), []byte(`{}`),
	)
	if err != nil {
		t.Fatalf("NewEvent tombstone: %v", err)
	}

	marked, err := event.MarkTombstone(tombstoneEvt)
	if err != nil {
		t.Fatalf("MarkTombstone: %v", err)
	}

	_ = proj.Handle(ctx, marked)

	reader, _ := NewSQLStreamReader(db, "test_", sqlpkg.SQLiteDialect{})

	activePage, err := reader.ListWithStatus(ctx, listing.ListOptions{
		Type:      "User",
		Tombstone: listing.TombstoneExclude,
	})
	if err != nil {
		t.Fatalf("ListWithStatus Exclude: %v", err)
	}

	if len(activePage.Items) != 0 {
		t.Fatalf("expected 0 active items, got %d", len(activePage.Items))
	}

	tombstonedPage, err := reader.ListWithStatus(ctx, listing.ListOptions{
		Type:      "User",
		Tombstone: listing.TombstoneOnly,
	})
	if err != nil {
		t.Fatalf("ListWithStatus Only: %v", err)
	}

	if len(tombstonedPage.Items) != 1 {
		t.Fatalf("expected 1 tombstoned item, got %d", len(tombstonedPage.Items))
	}

	if tombstonedPage.Items[0].Status != event.TombstoneTombstoned {
		t.Fatalf("expected TombstoneTombstoned, got %d", tombstonedPage.Items[0].Status)
	}

	allPage, err := reader.ListWithStatus(ctx, listing.ListOptions{
		Type:      "User",
		Tombstone: listing.TombstoneInclude,
	})
	if err != nil {
		t.Fatalf("ListWithStatus Include: %v", err)
	}

	if len(allPage.Items) != 1 {
		t.Fatalf("expected 1 total item, got %d", len(allPage.Items))
	}
}

func TestSQLStreamReader_List_TypeRequired(t *testing.T) {
	t.Parallel()
	db, _ := openSQLiteListingDB(t)
	defer func() { _ = db.Close() }()

	reader, _ := NewSQLStreamReader(db, "test_", sqlpkg.SQLiteDialect{})

	_, err := reader.ListWithStatus(context.Background(), listing.ListOptions{
		Type: "",
	})
	if err == nil {
		t.Fatal("expected error for empty type")
	}
}

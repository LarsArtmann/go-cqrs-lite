package stream_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/memory"
	"github.com/larsartmann/go-cqrs-lite/stream"
)

func TestInMemoryAggregateReader_List(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	seedEvents(t, store)

	reader := stream.NewInMemoryAggregateReader(store)

	t.Run("lists all active users", func(t *testing.T) {
		t.Parallel()

		page, err := stream.NewListBuilder(reader).
			OfType("User").
			PageSize(10).
			List(context.Background())
		if err != nil {
			t.Fatal(err)
		}

		if len(page.Items) != 1 {
			t.Errorf("got %d items, want 1", len(page.Items))
		}

		if page.HasMore {
			t.Error("HasMore = true, want false")
		}
	})

	t.Run("lists with status shows tombstoned", func(t *testing.T) {
		t.Parallel()

		page, err := stream.NewListBuilder(reader).
			OfType("User").
			IncludeDeleted().
			PageSize(10).
			ListWithStatus(context.Background())
		if err != nil {
			t.Fatal(err)
		}

		if len(page.Items) != 2 {
			t.Fatalf("got %d items, want 2", len(page.Items))
		}

		var foundActive, foundTombstoned bool

		for _, item := range page.Items {
			if item.Status.IsTombstoned() {
				foundTombstoned = true
			}

			if !item.Status.IsTombstoned() {
				foundActive = true
			}
		}

		if !foundActive {
			t.Error("expected at least one active (non-tombstoned) user")
		}

		if !foundTombstoned {
			t.Error("expected at least one tombstoned user")
		}
	})

	t.Run("OnlyDeleted filters to tombstoned", func(t *testing.T) {
		t.Parallel()

		page, err := stream.NewListBuilder(reader).
			OfType("User").
			OnlyDeleted().
			PageSize(10).
			ListWithStatus(context.Background())
		if err != nil {
			t.Fatal(err)
		}

		if len(page.Items) != 1 {
			t.Fatalf("got %d items, want 1", len(page.Items))
		}

		if !page.Items[0].Status.IsTombstoned() {
			t.Error("expected tombstoned status")
		}
	})

	t.Run("pagination with cursor", func(t *testing.T) {
		t.Parallel()

		page1, err := stream.NewListBuilder(reader).
			OfType("User").
			IncludeDeleted().
			PageSize(1).
			List(context.Background())
		if err != nil {
			t.Fatal(err)
		}

		if !page1.HasMore {
			t.Fatal("expected HasMore = true for first page")
		}

		if len(page1.Items) != 1 {
			t.Fatalf("got %d items, want 1", len(page1.Items))
		}

		page2, err := stream.NewListBuilder(reader).
			OfType("User").
			IncludeDeleted().
			PageSize(1).
			After(page1.Items[0].ID).
			List(context.Background())
		if err != nil {
			t.Fatal(err)
		}

		if len(page2.Items) != 1 {
			t.Fatalf("got %d items, want 1", len(page2.Items))
		}
	})
}

func TestInMemoryAggregateReader_EmptyJournal(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	reader := stream.NewInMemoryAggregateReader(store)

	page, err := stream.NewListBuilder(reader).
		OfType("User").
		List(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if len(page.Items) != 0 {
		t.Errorf("got %d items, want 0", len(page.Items))
	}
}

func seedEvents(t *testing.T, store *memory.MemoryStore) {
	t.Helper()

	ctx := context.Background()

	// Active user
	activeID := id.NewAggregateID()
	activeEvt, err := event.NewEvent(
		"user.created", activeID, "User",
		event.Version(1), []byte(`{"name":"Alice"}`),
	)
	if err != nil {
		t.Fatal(err)
	}

	if err = store.Save(
		ctx, event.NewAggregateRef(event.AggregateType("User"), activeID),
		[]event.Event{activeEvt},
		event.Version(0),
	); err != nil {
		t.Fatal(err)
	}

	// Tombstoned user
	deletedID := id.NewAggregateID()
	deletedEvt, err := event.NewEvent(
		"user.deleted", deletedID, "User",
		event.Version(1), []byte(`{"reason":"gdpr"}`),
		event.WithCustom(event.MetadataKeyTombstone, "true"),
	)
	if err != nil {
		t.Fatal(err)
	}

	if err = store.Save(
		ctx, event.NewAggregateRef(event.AggregateType("User"), deletedID),
		[]event.Event{deletedEvt},
		event.Version(0),
	); err != nil {
		t.Fatal(err)
	}

	// Order (different type)
	orderID := id.NewAggregateID()
	orderEvt, err := event.NewEvent(
		"order.created", orderID, "Order",
		event.Version(1), []byte(`{"total":99}`),
	)
	if err != nil {
		t.Fatal(err)
	}

	if err = store.Save(
		ctx, event.NewAggregateRef(event.AggregateType("Order"), orderID),
		[]event.Event{orderEvt},
		event.Version(0),
	); err != nil {
		t.Fatal(err)
	}
}

// Package main demonstrates the stream module: aggregate listing,
// tombstone filtering, and cursor-based pagination using InMemoryAggregateReader.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/memory"
	"github.com/larsartmann/go-cqrs-lite/stream"
)

func main() {
	ctx := context.Background()

	fmt.Println("=== go-cqrs-lite: Stream Module Demo ===")
	fmt.Println()

	store := memory.NewMemoryStore()

	populateEvents(ctx, store)

	reader := stream.NewInMemoryAggregateReader(store)

	runBasicList(ctx, reader)
	runTypeFilter(ctx, reader)
	runTombstoneFilter(ctx, reader)
	runCursorPagination(ctx, reader)

	fmt.Println("=== Demo Complete ===")
}

func populateEvents(ctx context.Context, store event.Store) {
	userIDs := make([]id.AggregateID, 5)
	orderIDs := make([]id.AggregateID, 3)

	for i := range userIDs {
		userIDs[i] = id.NewAggregateID()

		evt, err := event.NewEvent("user.created", userIDs[i], "User", event.Version(1), nil)
		if err != nil {
			log.Fatalf("create event: %v", err)
		}

		err = store.Save(ctx, event.NewAggregateRef("User", userIDs[i]), []event.Event{evt}, event.Version(0))
		if err != nil {
			log.Fatalf("save event: %v", err)
		}
	}

	for i := range orderIDs {
		orderIDs[i] = id.NewAggregateID()

		evt, err := event.NewEvent("order.placed", orderIDs[i], "Order", event.Version(1), nil)
		if err != nil {
			log.Fatalf("create event: %v", err)
		}

		err = store.Save(ctx, event.NewAggregateRef("Order", orderIDs[i]), []event.Event{evt}, event.Version(0))
		if err != nil {
			log.Fatalf("save event: %v", err)
		}
	}

	deletedID := userIDs[4]

	deleteEvt, err := event.NewEvent("user.deleted", deletedID, "User", event.Version(2), nil)
	if err != nil {
		log.Fatalf("create delete event: %v", err)
	}

	marked, err := event.MarkTombstone(deleteEvt)
	if err != nil {
		log.Fatalf("mark tombstone: %v", err)
	}

	err = store.Save(ctx, event.NewAggregateRef("User", deletedID), []event.Event{marked}, event.Version(1))
	if err != nil {
		log.Fatalf("save delete event: %v", err)
	}

	fmt.Printf(
		"[setup] Created %d users, %d orders (1 tombstoned user)\n\n",
		len(userIDs),
		len(orderIDs),
	)
}

func runBasicList(ctx context.Context, reader *stream.InMemoryAggregateReader) {
	fmt.Println("--- Basic List (all active aggregates) ---")

	page, err := stream.NewListBuilder(reader).PageSize(10).List(ctx)
	if err != nil {
		log.Fatalf("list: %v", err)
	}

	fmt.Printf("→ %d active aggregates (hasMore=%v)\n", len(page.Items), page.HasMore)

	for _, ref := range page.Items {
		fmt.Printf("  %s/%s version=%d events=%d\n", ref.Type, ref.ID, ref.Version, ref.EventCount)
	}

	fmt.Println()
}

func runTypeFilter(ctx context.Context, reader *stream.InMemoryAggregateReader) {
	fmt.Println("--- Filter by Type: User ---")

	page, err := stream.NewListBuilder(reader).OfType("User").PageSize(10).List(ctx)
	if err != nil {
		log.Fatalf("list users: %v", err)
	}

	fmt.Printf("→ %d users (hasMore=%v)\n", len(page.Items), page.HasMore)

	for _, ref := range page.Items {
		fmt.Printf("  User/%s version=%d\n", ref.ID, ref.Version)
	}

	fmt.Println()
}

func runTombstoneFilter(ctx context.Context, reader *stream.InMemoryAggregateReader) {
	fmt.Println("--- Tombstone Filters ---")

	activePage, err := stream.NewListBuilder(reader).OfType("User").List(ctx)
	if err != nil {
		log.Fatalf("list active: %v", err)
	}

	fmt.Printf("→ Active users: %d (tombstoned excluded)\n", len(activePage.Items))

	deletedPage, err := stream.NewListBuilder(reader).
		OfType("User").
		OnlyDeleted().
		ListWithStatus(ctx)
	if err != nil {
		log.Fatalf("list deleted: %v", err)
	}

	fmt.Printf("→ Deleted users: %d\n", len(deletedPage.Items))

	for _, s := range deletedPage.Items {
		fmt.Printf("  %s/%s status=%s\n", s.Ref.Type, s.Ref.ID, s.Status)
	}

	fmt.Println()
}

func runCursorPagination(ctx context.Context, reader *stream.InMemoryAggregateReader) {
	fmt.Println("--- Cursor Pagination (page size 2) ---")

	builder := stream.NewListBuilder(reader).PageSize(2)

	pageNum := 1

	var lastID id.AggregateID

	for {
		b := builder
		if !lastID.IsZero() {
			b = builder.After(lastID)
		}

		page, err := b.List(ctx)
		if err != nil {
			log.Fatalf("page %d: %v", pageNum, err)
		}

		if len(page.Items) == 0 {
			break
		}

		fmt.Printf("→ Page %d: %d items (hasMore=%v)\n", pageNum, len(page.Items), page.HasMore)

		for _, ref := range page.Items {
			fmt.Printf("  %s/%s\n", ref.Type, ref.ID)
		}

		lastID = page.Items[len(page.Items)-1].ID
		pageNum++

		if !page.HasMore {
			break
		}
	}

	fmt.Println()
}

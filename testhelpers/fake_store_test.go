package testhelpers_test

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/testhelpers"
)

func TestFakeStore_LoadToVersion(t *testing.T) {
	t.Parallel()

	store := testhelpers.NewFakeStore()
	ctx := context.Background()

	aggID := id.NewAggregateID()
	evt1 := testhelpers.QuickEvent("Created", aggID, "User", 1, nil)
	evt2 := testhelpers.QuickEvent("Updated", aggID, "User", 2, nil)
	evt3 := testhelpers.QuickEvent("Deleted", aggID, "User", 3, nil)

	_ = store.AppendBatch(ctx, "User", aggID, []event.Event{evt1, evt2, evt3})

	events, err := store.LoadToVersion(ctx, "User", aggID, 2)
	if err != nil {
		t.Fatalf("LoadToVersion: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
}

func TestFakeStore_LoadToVersion_EmptyStream(t *testing.T) {
	t.Parallel()

	store := testhelpers.NewFakeStore()
	ctx := context.Background()

	aggID := id.NewAggregateID()

	events, err := store.LoadToVersion(ctx, "User", aggID, 5)
	if err != nil {
		t.Fatalf("LoadToVersion: %v", err)
	}

	if len(events) != 0 {
		t.Fatalf("expected 0 events, got %d", len(events))
	}
}

func TestFakeStore_LoadToTimestamp(t *testing.T) {
	t.Parallel()

	store := testhelpers.NewFakeStore()
	ctx := context.Background()

	aggID := id.NewAggregateID()
	now := time.Now()

	evt1, _ := event.NewEvent(
		"Created",
		aggID,
		"User",
		1,
		nil,
		event.WithOccurredAt(now.Add(-2*time.Hour)),
	)
	evt2, _ := event.NewEvent(
		"Updated",
		aggID,
		"User",
		2,
		nil,
		event.WithOccurredAt(now.Add(-1*time.Hour)),
	)
	evt3, _ := event.NewEvent("Deleted", aggID, "User", 3, nil, event.WithOccurredAt(now))

	_ = store.AppendBatch(ctx, "User", aggID, []event.Event{evt1, evt2, evt3})

	events, err := store.LoadToTimestamp(ctx, "User", aggID, now.Add(-30*time.Minute))
	if err != nil {
		t.Fatalf("LoadToTimestamp: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
}

func TestFakeStore_LoadToTimestamp_EmptyStream(t *testing.T) {
	t.Parallel()

	store := testhelpers.NewFakeStore()
	ctx := context.Background()

	aggID := id.NewAggregateID()

	events, err := store.LoadToTimestamp(ctx, "User", aggID, time.Now())
	if err != nil {
		t.Fatalf("LoadToTimestamp: %v", err)
	}

	if len(events) != 0 {
		t.Fatalf("expected 0 events, got %d", len(events))
	}
}

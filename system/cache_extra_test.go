package system_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// bareStore exposes ONLY event.Store — no Journal, no SeekableJournal — so
// CachedEventStore's capability fallbacks can be exercised.
type bareStore struct{ event.Store }

func TestCachedEventStore_InvalidCapacity(t *testing.T) {
	t.Parallel()

	_, err := system.NewCachedEventStore(eventtest.NewFakeStore(), 0)
	if !errors.Is(err, system.ErrCacheCapacityInvalid) {
		t.Fatalf("capacity 0 error = %v, want ErrCacheCapacityInvalid", err)
	}
}

func TestCachedEventStore_AppendBatchInvalidates(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ref := id.NewStreamRef("CacheBatch", id.NewStreamID())
	cached, err := system.NewCachedEventStore(eventtest.NewFakeStore(), 16)
	if err != nil {
		t.Fatalf("NewCachedEventStore: %v", err)
	}

	first := newCacheTestEvent(t, ref, 1)
	if err := cached.Save(ctx, ref, []event.Event{first}, 0); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if _, err := cached.Load(ctx, ref); err != nil {
		t.Fatalf("warm Load: %v", err)
	}

	second := newCacheTestEvent(t, ref, 2)
	if err := cached.AppendBatch(ctx, ref, []event.Event{second}); err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	loaded, err := cached.Load(ctx, ref)
	if err != nil {
		t.Fatalf("post-batch Load: %v", err)
	}

	if len(loaded) != 2 {
		t.Fatalf("AppendBatch must invalidate cache: got %d events, want 2", len(loaded))
	}
}

func TestCachedEventStore_PassthroughReads(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ref := id.NewStreamRef("CachePass", id.NewStreamID())
	cached, err := system.NewCachedEventStore(eventtest.NewFakeStore(), 16)
	if err != nil {
		t.Fatalf("NewCachedEventStore: %v", err)
	}

	events := []event.Event{
		newCacheTestEvent(t, ref, 1),
		newCacheTestEvent(t, ref, 2),
	}
	if err := cached.AppendBatch(ctx, ref, events); err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	fromVersion, err := cached.LoadFromVersion(ctx, ref, 1)
	if err != nil {
		t.Fatalf("LoadFromVersion: %v", err)
	}
	if len(fromVersion) != 1 || fromVersion[0].Version() != 2 {
		t.Fatalf("LoadFromVersion(1) = %v, want only version-2 event", fromVersion)
	}

	toVersion, err := cached.LoadToVersion(ctx, ref, 1)
	if err != nil {
		t.Fatalf("LoadToVersion: %v", err)
	}
	if len(toVersion) != 1 {
		t.Fatalf("LoadToVersion(1) = %d events, want 1", len(toVersion))
	}

	toTimestamp, err := cached.LoadToTimestamp(ctx, ref, time.Now().Add(time.Minute))
	if err != nil {
		t.Fatalf("LoadToTimestamp: %v", err)
	}
	if len(toTimestamp) != 2 {
		t.Fatalf("LoadToTimestamp(now+1m) = %d events, want 2", len(toTimestamp))
	}

	readAll, err := cached.ReadAll(ctx)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(readAll) != 2 {
		t.Fatalf("ReadAll = %d events, want 2", len(readAll))
	}

	readFrom, err := cached.ReadFrom(ctx, id.EventID{}, 10)
	if err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}
	if len(readFrom) != 2 {
		t.Fatalf("ReadFrom = %d events, want 2", len(readFrom))
	}
}

func TestCachedEventStore_CacheStats(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ref := id.NewStreamRef("CacheStats", id.NewStreamID())
	cached, err := system.NewCachedEventStore(eventtest.NewFakeStore(), 16)
	if err != nil {
		t.Fatalf("NewCachedEventStore: %v", err)
	}

	if err := cached.AppendBatch(
		ctx,
		ref,
		[]event.Event{newCacheTestEvent(t, ref, 1)},
	); err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}

	if _, err := cached.Load(ctx, ref); err != nil {
		t.Fatalf("Load: %v", err)
	}

	size, capacity := cached.CacheStats()
	if capacity != 16 {
		t.Fatalf("capacity = %d, want 16", capacity)
	}
	if size < 1 {
		t.Fatalf("after one warm Load, size = %d, want >= 1", size)
	}
}

func TestCachedEventStore_JournalCapabilityMissing(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cached, err := system.NewCachedEventStore(bareStore{eventtest.NewFakeStore()}, 16)
	if err != nil {
		t.Fatalf("NewCachedEventStore: %v", err)
	}

	if _, err := cached.ReadAll(ctx); !errors.Is(err, system.ErrJournalMissing) {
		t.Fatalf("ReadAll error = %v, want ErrJournalMissing", err)
	}

	if _, err := cached.ReadFrom(
		ctx,
		id.EventID{},
		10,
	); !errors.Is(
		err,
		system.ErrSeekableJournalMissing,
	) {
		t.Fatalf("ReadFrom error = %v, want ErrSeekableJournalMissing", err)
	}
}

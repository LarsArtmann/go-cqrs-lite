package system

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// TestCachedEventStore_CheckSetGapWriteCannotStrandStaleSnapshot pins the
// generation check and cache.Set as one critical section: a write completing
// entirely between the check and the Set (forced via gapHook) must not
// strand the pre-write snapshot in the cache for every later reader.
func TestCachedEventStore_CheckSetGapWriteCannotStrandStaleSnapshot(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ref := id.NewStreamRef("CacheGapRace", id.NewStreamID())
	cached, err := NewCachedEventStore(eventtest.NewFakeStore(), 16)
	if err != nil {
		t.Fatalf("NewCachedEventStore: %v", err)
	}

	seed := make([]event.Event, 0, 4)
	for i := range 4 {
		seed = append(seed, eventtest.NewEvent(
			t, "cache.gap", ref.ID, ref.Type, event.Version(i+1), []byte(`{"kind":"gap"}`),
		))
	}
	if err := cached.AppendBatch(ctx, ref, seed); err != nil {
		t.Fatalf("seed AppendBatch: %v", err)
	}

	atGap := make(chan struct{})
	releaseGap := make(chan struct{})
	cached.gapHook = func() {
		close(atGap)
		<-releaseGap
	}

	loadDone := make(chan error, 1)
	go func() {
		_, err := cached.Load(ctx, ref)
		loadDone <- err
	}()

	<-atGap

	writeDone := make(chan error, 1)
	go func() {
		writeDone <- cached.AppendBatch(ctx, ref, []event.Event{eventtest.NewEvent(
			t, "cache.gap", ref.ID, ref.Type, 5, []byte(`{"kind":"gap"}`),
		)})
	}()

	// Buggy ordering: the write completes while the reader is parked between
	// check and Set, so its invalidations cannot remove the snapshot the
	// reader is about to cache. Fixed ordering: the write cannot complete
	// (the reader holds the critical-section mutex), so the timeout fires.
	gapTimedOut := false
	select {
	case <-writeDone:
	case <-time.After(1 * time.Second):
		gapTimedOut = true
	}

	close(releaseGap)

	if gapTimedOut {
		if err := <-writeDone; err != nil {
			t.Fatalf("gap AppendBatch: %v", err)
		}
	}
	if err := <-loadDone; err != nil {
		t.Fatalf("gap Load: %v", err)
	}

	cached.gapHook = nil

	events, err := cached.Load(ctx, ref)
	if err != nil {
		t.Fatalf("post-write Load: %v", err)
	}
	if len(events) != 5 {
		t.Fatalf(
			"write completed between check and Set stranded a stale snapshot: got %d events, want 5",
			len(events),
		)
	}
}

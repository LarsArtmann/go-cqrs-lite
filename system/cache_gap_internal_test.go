package system

import (
	"context"
	"testing"

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

	close(releaseGap)

	if err := <-writeDone; err != nil {
		t.Fatalf("gap AppendBatch: %v", err)
	}
	if err := <-loadDone; err != nil {
		t.Fatalf("gap Load: %v", err)
	}

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

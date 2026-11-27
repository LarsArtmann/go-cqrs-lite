package pebble

import (
	"context"
	"sync"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// Concurrent saves of an older and a newer version must leave the newer one
// stored, whichever goroutine commits last.
func TestSnapshotStore_Save_ConcurrentOlderDoesNotRegress(t *testing.T) {
	t.Parallel()

	store := newSnapshotStore(t)
	ctx := context.Background()

	for range 200 {
		streamID := id.NewStreamID()

		var wg sync.WaitGroup
		start := make(chan struct{})
		wg.Add(2)

		for _, tc := range []struct {
			version int
			state   string
		}{{1, "older"}, {7, "newer"}} {
			go func() {
				defer wg.Done()
				<-start

				_ = store.Save(ctx, testSnapshot(t, streamID, tc.version, tc.state))
			}()
		}

		close(start)
		wg.Wait()

		ref := id.NewStreamRef("Order", streamID)
		got, err := store.Load(ctx, ref)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}

		if got.Version != event.Version(7) {
			t.Fatalf("stream %s stored version %d (%q), want 7: concurrent save regressed the snapshot",
				streamID, got.Version.Int(), string(got.State))
		}
	}
}

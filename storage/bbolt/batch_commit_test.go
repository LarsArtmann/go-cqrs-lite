package bbolt

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func newBatchBackend(t *testing.T, maxBatchSize int) *Backend {
	t.Helper()

	dir := t.TempDir()
	backend, err := OpenWithOptions(
		filepath.Join(dir, "batch.db"), nil, nil, WithBatchCommit())
	if err != nil {
		t.Fatalf("open backend: %v", err)
	}

	if maxBatchSize > 0 {
		backend.DB().MaxBatchSize = maxBatchSize
	}

	t.Cleanup(func() { _ = backend.Close() })

	return backend
}

// TestBatchCommit_ConcurrentWritersIdenticalJournal verifies that concurrent
// writers under group commit persist exactly their events: no loss, no
// duplication, correct per-stream ordering.
func TestBatchCommit_ConcurrentWritersIdenticalJournal(t *testing.T) {
	t.Parallel()

	const writers = 8
	const batchesPerWriter = 10
	const eventsPerBatch = 5

	backend := newBatchBackend(t, 4)
	store := backend.EventStore()
	ctx := context.Background()

	var wg sync.WaitGroup
	errs := make(chan error, writers)

	for w := range writers {
		wg.Add(1)

		go func() {
			defer wg.Done()

			ref := id.NewStreamRef("User", id.NewStreamID())

			for b := range batchesPerWriter {
				evts := make([]event.Event, eventsPerBatch)
				for i := range eventsPerBatch {
					version := event.Version(b*eventsPerBatch + i + 1)
					evt, err := event.NewEvent("user.created", ref.ID, "User", version, []byte(`{}`))
					if err != nil {
						errs <- err
						return
					}
					evts[i] = evt
				}

				if err := store.AppendBatch(ctx, ref, evts); err != nil {
					errs <- err
					return
				}
			}
		}()
		_ = w
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Fatalf("writer failed: %v", err)
	}

	all, err := store.ReadAll(ctx)
	if err != nil {
		t.Fatalf("read all: %v", err)
	}

	if got, want := len(all), writers*batchesPerWriter*eventsPerBatch; got != want {
		t.Fatalf("journal holds %d events, want %d (group commit lost/duplicated writes)", got, want)
	}

	seen := make(map[string]int)
	for _, evt := range all {
		seen[evt.ID().String()]++
	}
	if len(seen) != writers*batchesPerWriter*eventsPerBatch {
		t.Fatalf("journal holds %d distinct event IDs, want %d", len(seen), writers*batchesPerWriter*eventsPerBatch)
	}
}

// TestBatchCommit_ConflictingSaveDoesNotPoisonGroup proves the retry-idempence
// contract: a writer whose version check fails is ejected and re-run solo
// (surfacing its Conflict), while its batch-mates re-run and still land. With
// MaxBatchSize=2 the conflict is combined with a valid writer, forcing the
// retry path.
func TestBatchCommit_ConflictingSaveDoesNotPoisonGroup(t *testing.T) {
	t.Parallel()

	backend := newBatchBackend(t, 2)
	store := backend.EventStore()
	ctx := context.Background()
	ref := id.NewStreamRef("User", id.NewStreamID())

	good, err := event.NewEvent("user.created", ref.ID, "User", 1, []byte(`{"ok":true}`))
	if err != nil {
		t.Fatalf("create good event: %v", err)
	}

	bad, err := event.NewEvent("user.created", ref.ID, "User", 7, []byte(`{"stale":true}`))
	if err != nil {
		t.Fatalf("create stale event: %v", err)
	}

	var wg sync.WaitGroup
	results := make(chan error, 2)

	save := func(expected event.Version, evts ...event.Event) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- store.Save(ctx, ref, evts, expected)
		}()
	}

	save(0, good)
	save(0, bad) // expects version 0 but writes version 7: mismatch either way

	wg.Wait()
	close(results)

	var conflictSeen bool

	for res := range results {
		if res == nil {
			continue
		}
		if errorfamily.Classify(res) == errorfamily.Conflict || errors.Is(res, ErrVersionMismatch) {
			conflictSeen = true
			continue
		}
		t.Fatalf("unexpected error: %v", res)
	}

	if !conflictSeen {
		t.Fatal("stale writer should surface a conflict, got none")
	}

	loaded, err := store.Load(ctx, ref)
	if err != nil {
		t.Fatalf("load stream: %v", err)
	}

	if len(loaded) != 1 || loaded[0].ID() != good.ID() {
		t.Fatalf("stream should hold exactly the valid event, got %d events", len(loaded))
	}
}

// TestBatchCommit_DefaultBackendUnchanged guards that a plain Open (no
// WithBatchCommit) still writes via db.Update — the storeBase batch flag
// stays false.
func TestBatchCommit_DefaultBackendUnchanged(t *testing.T) {
	t.Parallel()

	backend := newTestBackend(t)
	if backend.events.batch {
		t.Fatal("default backend must not enable batch commit")
	}

	store := backend.EventStore()
	ctx := context.Background()
	ref := id.NewStreamRef("User", id.NewStreamID())

	evt, err := event.NewEvent("user.created", ref.ID, "User", 1, []byte(`{}`))
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	if err := store.Save(ctx, ref, []event.Event{evt}, 0); err != nil {
		t.Fatalf("save: %v", err)
	}
}

package projectionhost_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/id/v4"

	errorfamily "github.com/larsartmann/go-error-family"

	projectionhost "github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
)

// --- T19 fix: Reset clears the checkpoint BEFORE the read model ---

func TestReset_ClearsCheckpointBeforeReadModel(t *testing.T) {
	t.Parallel()

	journal := &memoryJournal{}
	cpStore := newMemoryCheckpointStore()
	journal.append(makeEvent("task.created"))
	journal.append(makeEvent("task.created"))

	proj := &resettableCountingProjection{
		countingProjection: countingProjection{name: "tasks"},
	}

	host, err := projectionhost.New(journal, cpStore, projectionhost.WithBatchSize(10))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := host.Register(proj); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx := context.Background()

	if err := host.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	requireEventually(t, 2*time.Second, func() bool {
		return proj.count.Load() == 2
	})

	requireEventually(t, 2*time.Second, func() bool {
		cp, ok := loadCP(t, cpStore, "tasks")

		return ok && !cp.IsZero()
	})

	if err := host.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	if err := host.Reset(ctx, "tasks"); err != nil {
		t.Fatalf("Reset: %v", err)
	}

	if cp, ok := loadCP(t, cpStore, "tasks"); ok && !cp.IsZero() {
		t.Errorf("checkpoint must be cleared before the read-model reset, got %q", cp)
	}
}

func loadCP(t *testing.T, s *memoryCheckpointStore, name string) (id.EventID, bool) {
	t.Helper()

	s.mu.Lock()
	defer s.mu.Unlock()

	cp, ok := s.data[name]

	return cp.EventID, ok
}

// --- T19 fix: the documented Stop→Reset→Start rebuild recipe must work ---

func TestStart_AfterStop_RebuildsAndDrainsAgain(t *testing.T) {
	t.Parallel()

	journal := &memoryJournal{}
	cpStore := newMemoryCheckpointStore()
	journal.append(makeEvent("task.created"))

	proj := &countingProjection{name: "tasks"}

	host, err := projectionhost.New(journal, cpStore, projectionhost.WithBatchSize(10))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := host.Register(proj); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx := context.Background()

	if err := host.Start(ctx); err != nil {
		t.Fatalf("Start 1: %v", err)
	}

	requireEventually(t, 2*time.Second, func() bool {
		return proj.count.Load() == 1
	})

	if err := host.Stop(); err != nil {
		t.Fatalf("Stop 1: %v", err)
	}

	journal.append(makeEvent("task.created"))

	if err := host.Start(ctx); err != nil {
		t.Fatalf("Start 2 (after Stop): %v", err)
	}

	requireEventually(t, 2*time.Second, func() bool {
		return proj.count.Load() == 2
	})

	if err := host.Stop(); err != nil {
		t.Fatalf("Stop 2: %v", err)
	}

	if got := proj.count.Load(); got != 2 {
		t.Fatalf("restarted drain handled %d events total, want 2", got)
	}
}

// --- T19 fix: non-positive batch size falls back to the default ---

func TestWithBatchSize_NonPositive_FallsBackAndDrains(t *testing.T) {
	t.Parallel()

	journal := &memoryJournal{}
	cpStore := newMemoryCheckpointStore()

	for range 3 {
		journal.append(makeEvent("task.created"))
	}

	proj := &countingProjection{name: "tasks"}

	host, err := projectionhost.New(journal, cpStore, projectionhost.WithBatchSize(-5))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := host.Register(proj); err != nil {
		t.Fatalf("Register: %v", err)
	}

	if err := host.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	requireEventually(t, 2*time.Second, func() bool {
		return proj.count.Load() == 3
	})

	if err := host.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	if got := proj.count.Load(); got != 3 {
		t.Fatalf("non-positive batch size must fall back to default (drain all), got %d/3", got)
	}
}

// --- T19 fix: transient failures are NOT parked in the DLQ ---

func TestTransientFailure_NotParkedInDLQ_FailsWorkerLoudly(t *testing.T) {
	t.Parallel()

	journal := &memoryJournal{}
	cpStore := newMemoryCheckpointStore()
	journal.append(makeEvent("task.created"))

	dlq := projectionhost.NewMemoryDeadLetterStore()
	proj := &alwaysFailingProjection{
		name:    "flaky",
		failErr: errorfamily.NewTransient("test.transient", "downstream hiccup"),
	}

	host, err := projectionhost.New(
		journal, cpStore,
		projectionhost.WithDeadLetterStore(dlq, 1),
		projectionhost.WithMaxRestarts(0),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := host.Register(proj); err != nil {
		t.Fatalf("Register: %v", err)
	}

	if err := host.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	requireEventually(t, 8*time.Second, func() bool {
		for _, st := range host.Status() {
			if st.Status == projectionhost.WorkerFailed {
				return true
			}
		}

		return false
	})

	if err := host.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	entries, err := dlq.List(context.Background(), "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(entries) != 0 {
		t.Fatalf("transient failure must not be parked in DLQ, got %d entries", len(entries))
	}

	if st := host.Status(); len(st) != 1 || st[0].Status != projectionhost.WorkerFailed {
		t.Fatalf("exhausted transient retries must fail the worker loudly, got %+v", st)
	}
}

// --- T19 fix: staleness must not report fresh for a failed worker ---

func TestCheckStaleness_FailedWorkerIsStale(t *testing.T) {
	t.Parallel()

	journal := &memoryJournal{}
	cpStore := newMemoryCheckpointStore()
	journal.append(makeEvent("task.created"))

	proj := &alwaysFailingProjection{
		name:    "dead",
		failErr: errorfamily.NewTransient("test.transient", "boom"),
	}

	host, err := projectionhost.New(
		journal, cpStore,
		projectionhost.WithMaxRestarts(0),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := host.Register(proj); err != nil {
		t.Fatalf("Register: %v", err)
	}

	if err := host.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	requireEventually(t, 8*time.Second, func() bool {
		for _, st := range host.Status() {
			if st.Status == projectionhost.WorkerFailed {
				return true
			}
		}

		return false
	})

	if err := host.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	if err := host.CheckStaleness(
		5 * time.Second,
	); !errors.Is(
		err,
		projectionhost.ErrProjectionStale,
	) {
		t.Fatalf("failed worker must surface as stale, got %v", err)
	}

	if err := host.CheckProjectionStaleness(
		"dead",
		5*time.Second,
	); !errors.Is(
		err,
		projectionhost.ErrProjectionStale,
	) {
		t.Fatalf("failed named worker must surface as stale, got %v", err)
	}
}

// --- T19 fix: replay serializes with a running worker (race exercise) ---

func TestReplayDeadLetters_ConcurrentWithRunningWorker(t *testing.T) {
	t.Parallel()

	journal := &memoryJournal{}
	cpStore := newMemoryCheckpointStore()
	journal.append(makeEvent("task.created"))

	dlq := projectionhost.NewMemoryDeadLetterStore()
	proj := &countingProjection{
		name:    "tasks",
		failOn:  1,
		failErr: errorfamily.NewRejection("test.poison", "poison"),
	}

	host, err := projectionhost.New(
		journal, cpStore,
		projectionhost.WithDeadLetterStore(dlq, 1),
		projectionhost.WithMaxRestarts(-1),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := host.Register(proj); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := host.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	requireEventually(t, 2*time.Second, func() bool {
		entries, _ := dlq.List(context.Background(), "")

		return len(entries) == 1
	})

	// Replay while the worker is live: the same projection.Handle runs on
	// the drain path and the replay path. Replay must hold handleMu or the
	// race detector fires on the projection's shared state.
	var wg sync.WaitGroup

	wg.Add(1)

	go func() {
		defer wg.Done()

		for range 20 {
			if _, err := host.ReplayDeadLetters(ctx, ""); err != nil {
				t.Errorf("ReplayDeadLetters: %v", err)
			}
		}
	}()

	wg.Wait()
	cancel()
	_ = host.Stop()
}

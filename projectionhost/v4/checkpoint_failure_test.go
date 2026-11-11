package projectionhost_test

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
)

// selectiveFailCheckpointStore wraps memoryCheckpointStore and fails Save
// for the staged event IDs it is told to fail — simulating live-phase
// checkpoint write failures under load (e.g. SQLITE_BUSY on the deployment
// engine). Failing by event ID (not by count) deterministically targets
// live-phase saves while the catch-up batch saves keep succeeding.
type selectiveFailCheckpointStore struct {
	*memoryCheckpointStore

	mu      sync.Mutex
	failIDs map[id.EventID]error
}

func (s *selectiveFailCheckpointStore) Save(
	ctx context.Context,
	name string,
	cp event.Checkpoint,
) error {
	s.mu.Lock()
	failErr := s.failIDs[cp.EventID]
	s.mu.Unlock()

	if failErr != nil {
		return failErr
	}

	return s.memoryCheckpointStore.Save(ctx, name, cp)
}

// heal clears all staged failures, so subsequent saves (e.g. the shutdown
// flush) succeed.
func (s *selectiveFailCheckpointStore) heal() {
	s.mu.Lock()
	s.failIDs = nil
	s.mu.Unlock()
}

// publishErr invokes the live handler synchronously and returns its error,
// mimicking the dispatch contract of in-process buses (EventBus sets
// BlockPublishUntilSubscriberAck): a handler error surfaces to the
// publishing caller.
func (s *appendingSub) publishErr(ctx context.Context, evt event.Event) error {
	s.mu.Lock()
	h := s.handler
	s.mu.Unlock()

	if h == nil {
		return nil
	}

	return h(ctx, evt)
}

// warnCapture collects Warn-level log records so tests can assert a failure
// was surfaced as a warning instead of an error.
type warnCapture struct {
	mu       sync.Mutex
	messages []string
}

func (h *warnCapture) Enabled(_ context.Context, _ slog.Level) bool { return true }

func (h *warnCapture) Handle(_ context.Context, r slog.Record) error {
	if r.Level == slog.LevelWarn {
		h.mu.Lock()
		h.messages = append(h.messages, r.Message)
		h.mu.Unlock()
	}

	return nil
}

func (h *warnCapture) WithAttrs(_ []slog.Attr) slog.Handler { return h }

func (h *warnCapture) WithGroup(_ string) slog.Handler { return h }

func (h *warnCapture) warns() []string {
	h.mu.Lock()
	defer h.mu.Unlock()

	return append([]string(nil), h.messages...)
}

// startFailingLiveHost boots a live host like startLiveHost, but with a
// checkpoint store that fails for staged event IDs and a warn-capturing
// logger.
func startFailingLiveHost(
	t *testing.T,
	failIDs map[id.EventID]error,
) (func(), *appendingSub, *selectiveFailCheckpointStore, *countingProjection, *warnCapture) {
	t.Helper()

	journal := &memoryJournal{}
	cpStore := &selectiveFailCheckpointStore{
		memoryCheckpointStore: newMemoryCheckpointStore(),
		failIDs:               failIDs,
	}

	sub := &appendingSub{journal: journal, appended: make(chan struct{})}
	proj := &countingProjection{name: "cadence", eventTypes: []event.Type{"task.created"}}

	logs := &warnCapture{}

	host, err := projectionhost.New(journal, cpStore,
		projectionhost.WithSubscriber(sub),
		projectionhost.WithBatchSize(10),
		projectionhost.WithLogger(slog.New(logs)),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := host.Register(proj); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	if err := host.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	stopped := false
	stop := func() {
		if !stopped {
			stopped = true
			_ = host.Stop()
		}
	}
	t.Cleanup(stop)

	<-sub.appended

	requireEventually(t, 3*time.Second, func() bool {
		return proj.count.Load() == 3
	})

	return stop, sub, cpStore, proj, logs
}

// TestLiveCheckpoint_SaveFailureDoesNotFailHandler pins issue #128's core
// contract: a live checkpoint save failure must NOT fail the live handler
// (which would fail the publisher's dispatch for an already-journaled,
// already-applied event). The failure degrades to a warn; the staged
// checkpoint carries forward and the NEXT live event's flush persists it.
func TestLiveCheckpoint_SaveFailureDoesNotFailHandler(t *testing.T) {
	t.Parallel()

	evt1 := makeEvent("task.created")
	evt2 := makeEvent("task.created")

	stop, sub, cpStore, proj, logs := startFailingLiveHost(t, map[id.EventID]error{
		evt1.ID(): errors.New("database is locked (5) (SQLITE_BUSY)"),
	})

	if err := sub.publishErr(context.Background(), evt1); err != nil {
		t.Fatalf("live handler failed on checkpoint save error: %v", err)
	}

	if got := proj.count.Load(); got != 4 {
		t.Errorf("processed events = %d, want 4 (failed-checkpoint event still processed)", got)
	}

	if err := sub.publishErr(context.Background(), evt2); err != nil {
		t.Fatalf("live handler failed after prior checkpoint save error: %v", err)
	}

	// The next event's flush persists the staged checkpoint (event 2
	// subsumes the failed event 1 — checkpoints only move forward).
	requireEventually(t, 3*time.Second, func() bool {
		cp, err := cpStore.Load(context.Background(), "cadence")
		return err == nil && cp.EventID == evt2.ID()
	})

	warned := false

	for _, msg := range logs.warns() {
		if msg == "live checkpoint save failed; will retry on next event" {
			warned = true
		}
	}

	if !warned {
		t.Errorf("expected a warn log for the failed live checkpoint save, got %v", logs.warns())
	}

	stop()
}

// TestLiveCheckpoint_SaveFailureFlushesAtShutdown pins the carry-forward
// contract for persistent live save failures: with every live flush failing,
// the staged checkpoint survives to the shutdown flush, which persists the
// LAST staged event when the store recovers.
func TestLiveCheckpoint_SaveFailureFlushesAtShutdown(t *testing.T) {
	t.Parallel()

	evt1 := makeEvent("task.created")
	evt2 := makeEvent("task.created")

	stop, sub, cpStore, proj, _ := startFailingLiveHost(t, map[id.EventID]error{
		evt1.ID(): errors.New("database is locked (5) (SQLITE_BUSY)"),
		evt2.ID(): errors.New("database is locked (5) (SQLITE_BUSY)"),
	})

	if err := sub.publishErr(context.Background(), evt1); err != nil {
		t.Fatalf("live handler failed evt1: %v", err)
	}

	if err := sub.publishErr(context.Background(), evt2); err != nil {
		t.Fatalf("live handler failed evt2: %v", err)
	}

	if got := proj.count.Load(); got != 5 {
		t.Errorf("processed events = %d, want 5 (both live events processed)", got)
	}

	cp, err := cpStore.Load(context.Background(), "cadence")
	if err != nil {
		t.Fatalf("Load before Stop: %v", err)
	}

	if cp.EventID == evt1.ID() || cp.EventID == evt2.ID() {
		t.Fatalf("live checkpoint persisted despite failing store: %v", cp.EventID)
	}

	cpStore.heal()
	stop()

	cp, err = cpStore.Load(context.Background(), "cadence")
	if err != nil {
		t.Fatalf("Load after Stop: %v", err)
	}

	if cp.EventID != evt2.ID() {
		t.Errorf("shutdown flush persisted %v, want the last staged event %v", cp.EventID, evt2.ID())
	}
}

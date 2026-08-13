package projectionhost_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
)

// appendingSub is a non-blocking subscriber that appends events to the journal
// during SubscribeAll. This deterministically simulates the TOCTOU race window:
// events are in the journal but were not present when the initial drain's last
// ReadFrom returned zero. Without the catch-up drain, these events are
// permanently lost.
type appendingSub struct {
	journal  *memoryJournal
	mu       sync.Mutex
	handler  event.Handler
	appended chan struct{}
}

func (s *appendingSub) Subscribe(_ event.Type, _ event.Handler) error {
	return errors.New("appendingSub only supports SubscribeAll")
}

func (s *appendingSub) SubscribeAll(handler event.Handler) error {
	for range 3 {
		s.journal.append(makeEvent("task.created"))
	}

	s.mu.Lock()
	s.handler = handler
	s.mu.Unlock()

	close(s.appended)

	return nil
}

func (s *appendingSub) publish(ctx context.Context, evt event.Event) {
	s.mu.Lock()
	h := s.handler
	s.mu.Unlock()

	if h != nil {
		_ = h(ctx, evt)
	}
}

// TestHost_CatchUpDrain_PicksUpEventsPublishedDuringSubscribeWindow is a
// regression test for the TOCTOU race where events published between the
// initial journal drain and subscriber registration were silently lost.
//
// The appendingSub deterministically triggers the race by appending events to
// the journal inside SubscribeAll — after the initial drain completed (0 events
// found) but before the catch-up drain runs. Without the catch-up drain, these
// events would never be processed.
func TestHost_CatchUpDrain_PicksUpEventsPublishedDuringSubscribeWindow(t *testing.T) {
	t.Parallel()

	journal := &memoryJournal{}
	cpStore := newMemoryCheckpointStore()

	sub := &appendingSub{
		journal:  journal,
		appended: make(chan struct{}),
	}

	proj := &countingProjection{name: "catchup", eventTypes: []event.Type{"task.created"}}

	host, err := projectionhost.New(
		journal, cpStore,
		projectionhost.WithSubscriber(sub),
		projectionhost.WithBatchSize(10),
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

	<-sub.appended

	requireEventually(t, 3*time.Second, func() bool {
		return proj.count.Load() == 3
	})

	cancel()
	_ = host.Stop()
}

// TestHost_CatchUpDrain_LiveDeliveryWorksAfterCatchUp verifies that after the
// catch-up drain processes missed events, the live handler still delivers
// events published via the subscriber callback.
func TestHost_CatchUpDrain_LiveDeliveryWorksAfterCatchUp(t *testing.T) {
	t.Parallel()

	journal := &memoryJournal{}
	cpStore := newMemoryCheckpointStore()

	sub := &appendingSub{
		journal:  journal,
		appended: make(chan struct{}),
	}

	proj := &countingProjection{
		name:       "live-after-catchup",
		eventTypes: []event.Type{"task.created"},
	}

	host, err := projectionhost.New(
		journal, cpStore,
		projectionhost.WithSubscriber(sub),
		projectionhost.WithBatchSize(10),
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

	// Wait for SubscribeAll to register the handler (and append 3 events).
	<-sub.appended

	// Wait for catch-up drain to process the 3 events.
	requireEventually(t, 3*time.Second, func() bool {
		return proj.count.Load() == 3
	})

	// Publish a live event via the subscriber callback.
	sub.publish(ctx, makeEvent("task.created"))

	requireEventually(t, 3*time.Second, func() bool {
		return proj.count.Load() == 4
	})

	cancel()
	_ = host.Stop()
}

// TestHost_CatchUpDrain_WorkerLiveVisibleDuringBlockingSubscribe verifies that
// WorkerLive status is set BEFORE SubscribeAll, so it is visible during the
// blocking subscriber's lifetime (not just nanoseconds before WorkerStopped).
//
// Without this, consumers polling host.Status() for WorkerLive readiness would
// never see it when using blocking subscribers (Watermill, NATS, etc.).
func TestHost_CatchUpDrain_WorkerLiveVisibleDuringBlockingSubscribe(t *testing.T) {
	t.Parallel()

	journal := &memoryJournal{}
	cpStore := newMemoryCheckpointStore()

	// channelSubscriber blocks in SubscribeAll (range over channel) — mimicking
	// a blocking message-broker subscriber like Watermill GoChannel.
	liveSub := newChannelSubscriber()

	proj := &countingProjection{name: "blocking-live", eventTypes: []event.Type{"task.created"}}

	host, err := projectionhost.New(
		journal, cpStore,
		projectionhost.WithSubscriber(liveSub),
		projectionhost.WithBatchSize(10),
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

	// The channelSubscriber blocks in SubscribeAll. If WorkerLive is set AFTER
	// SubscribeAll (the regression), this poll will time out because the worker
	// never reaches WorkerLive — it's stuck inside SubscribeAll.
	requireEventually(t, 3*time.Second, func() bool {
		for _, s := range host.Status() {
			if s.Status == "live" {
				return true
			}
		}

		return false
	})

	// Send a live event — should be delivered via the blocking subscriber.
	liveSub.send(makeEvent("task.created"))

	requireEventually(t, 3*time.Second, func() bool {
		return proj.count.Load() == 1
	})

	cancel()
	liveSub.close()
	_ = host.Stop()
}

package projectionhost_test

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
)

// startLiveHost boots a host with the appendingSub subscriber (3 catch-up
// events), waits for them to be processed, and returns an idempotent stop
// function plus the pieces needed to drive and observe live events. The
// catch-up drain persists exactly one batch checkpoint before any live event,
// which the save-count assertions account for.
func startLiveHost(
	t *testing.T,
	opts ...projectionhost.HostOption,
) (func(), *appendingSub, *memoryCheckpointStore, *countingProjection) {
	t.Helper()

	journal := &memoryJournal{}
	cpStore := newMemoryCheckpointStore()

	sub := &appendingSub{
		journal:  journal,
		appended: make(chan struct{}),
	}

	proj := &countingProjection{name: "cadence", eventTypes: []event.Type{"task.created"}}

	allOpts := append([]projectionhost.HostOption{
		projectionhost.WithSubscriber(sub),
		projectionhost.WithBatchSize(10),
	}, opts...)

	host, err := projectionhost.New(journal, cpStore, allOpts...)
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

	return stop, sub, cpStore, proj
}

// TestLiveCheckpoint_DefaultSavesEveryEvent guards the default behavior: with
// no checkpoint options, every live event persists its checkpoint immediately
// (1 catch-up batch save + 1 per live event).
func TestLiveCheckpoint_DefaultSavesEveryEvent(t *testing.T) {
	t.Parallel()

	stop, sub, cpStore, proj := startLiveHost(t)

	for range 4 {
		sub.publish(context.Background(), makeEvent("task.created"))
	}

	requireEventually(t, 3*time.Second, func() bool {
		return proj.count.Load() == 7
	})

	if got, want := cpStore.saves.Load(), int64(1+4); got != want {
		t.Errorf("checkpoint saves = %d, want %d (catch-up batch + one per live event)", got, want)
	}

	stop()

	if got := cpStore.saves.Load(); got != 5 {
		t.Errorf("saves after Stop = %d, want 5 (no pending to flush by default)", got)
	}
}

// TestLiveCheckpoint_EveryBatchesAndFlushesAtShutdown proves the N-batch
// cadence: 7 live events with WithCheckpointEvery(3) persist at live events 3
// and 6, and the pending event 7 flushes when the host stops.
func TestLiveCheckpoint_EveryBatchesAndFlushesAtShutdown(t *testing.T) {
	t.Parallel()

	stop, sub, cpStore, proj := startLiveHost(t, projectionhost.WithCheckpointEvery(3))

	for range 7 {
		sub.publish(context.Background(), makeEvent("task.created"))
	}

	// 3 catch-up events + 7 live events.
	requireEventually(t, 3*time.Second, func() bool {
		return proj.count.Load() == 10
	})

	// Catch-up batch save + live flushes at events 3 and 6.
	if got, want := cpStore.saves.Load(), int64(1+2); got != want {
		t.Fatalf("mid-stream checkpoint saves = %d, want %d", got, want)
	}

	stop()

	if got := cpStore.saves.Load(); got != 4 {
		t.Errorf("saves after Stop = %d, want 4 (shutdown flush of pending event)", got)
	}

	cp, err := cpStore.Load(context.Background(), "cadence")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cp.EventID.IsZero() {
		t.Error("shutdown flush did not persist the pending checkpoint")
	}
}

// TestLiveCheckpoint_CrashWindowIsBounded proves the documented at-least-once
// trade-off: live events beyond the cadence stay pending (only the catch-up
// batch is durable), which bounds crash reprocessing at n-1 live events.
func TestLiveCheckpoint_CrashWindowIsBounded(t *testing.T) {
	t.Parallel()

	stop, sub, cpStore, proj := startLiveHost(t, projectionhost.WithCheckpointEvery(100))

	for range 5 {
		sub.publish(context.Background(), makeEvent("task.created"))
	}

	requireEventually(t, 3*time.Second, func() bool {
		return proj.count.Load() == 8
	})

	if got := cpStore.saves.Load(); got != 1 {
		t.Errorf("checkpoint saves = %d, want 1 (catch-up batch only; live events pending)", got)
	}

	stop()

	if got := cpStore.saves.Load(); got != 2 {
		t.Errorf("saves after Stop = %d, want 2 (graceful shutdown flush)", got)
	}
}

// TestLiveCheckpoint_IntervalFlushesAfterElapsed proves the time cadence:
// events inside the interval stay pending; the first event after the interval
// elapses flushes the staged checkpoint.
func TestLiveCheckpoint_IntervalFlushesAfterElapsed(t *testing.T) {
	t.Parallel()

	stop, sub, cpStore, proj := startLiveHost(
		t, projectionhost.WithCheckpointInterval(120*time.Millisecond),
	)

	// First live event flushes (time since the zero-value last save exceeds
	// any interval); the second stays pending.
	sub.publish(context.Background(), makeEvent("task.created"))
	sub.publish(context.Background(), makeEvent("task.created"))

	requireEventually(t, 3*time.Second, func() bool {
		return proj.count.Load() == 5
	})

	if got, want := cpStore.saves.Load(), int64(1+1); got != want {
		t.Fatalf("checkpoint saves after burst = %d, want %d", got, want)
	}

	time.Sleep(150 * time.Millisecond)

	// Next event crosses the interval: the pending event flushes, the new
	// event becomes pending.
	sub.publish(context.Background(), makeEvent("task.created"))

	requireEventually(t, 3*time.Second, func() bool {
		return proj.count.Load() == 6
	})

	if got := cpStore.saves.Load(); got != 3 {
		t.Errorf("checkpoint saves after interval = %d, want 3", got)
	}

	cp, err := cpStore.Load(context.Background(), "cadence")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cp.EventID.IsZero() {
		t.Error("expected a persisted checkpoint after interval flush")
	}

	stop()
}

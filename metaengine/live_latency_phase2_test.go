package metaengine_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestStore_Replan_PicksUpLiveRTTShift proves Replan re-reads Profile() and
// re-routes when live measurements shift. The original plan routes to the
// remote engine (cheap prior RTT); after a live RTT shift and Replan, it
// routes to the local engine.
func TestStore_Replan_PicksUpLiveRTTShift(t *testing.T) {
	remote := newFakeRemote("remote", 1, 500*time.Microsecond)
	local := newFakeLocal("local", 1_000_000)

	store := planWith(t, local, remote)
	if w := winnerEngine(store); w != "remote" {
		t.Fatalf("initial winner = %q, want remote", w)
	}

	v1 := store.Plan().Version

	// Shift remote's RTT high via a live tracker.
	tracker := metaengine.NewLatencyTracker(metaengine.WithTrackerAlpha(0.5))
	remote.SetRTTTracker(tracker)
	for range 30 {
		tracker.Record(50 * time.Millisecond)
	}

	if err := store.Replan(context.Background()); err != nil {
		t.Fatalf("Replan: %v", err)
	}

	if w := winnerEngine(store); w != "local" {
		t.Fatalf("after Replan winner = %q, want local", w)
	}

	if store.Plan().Version <= v1 {
		t.Errorf("Version = %d, want > %d (incremented)", store.Plan().Version, v1)
	}
}

// TestStore_Replan_CancelledContext returns an error.
func TestStore_Replan_CancelledContext(t *testing.T) {
	remote := newFakeRemote("remote", 1, 1*time.Millisecond)
	store := planWith(t, remote)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := store.Replan(ctx); err == nil {
		t.Fatal("Replan with cancelled context should return error")
	}
}

// TestStore_CheckRouting_DetectsCheaperAlternative proves CheckRouting emits
// REPLAN-SUGGESTED when a live RTT shift makes an alternative engine cheaper
// beyond the hysteresis deadband.
func TestStore_CheckRouting_DetectsCheaperAlternative(t *testing.T) {
	remote := newFakeRemote("remote", 1, 500*time.Microsecond)
	local := newFakeLocal("local", 1_000_000)

	store := planWith(t, local, remote)
	if w := winnerEngine(store); w != "remote" {
		t.Fatalf("initial winner = %q, want remote", w)
	}

	// Shift remote RTT very high — local is now vastly cheaper.
	tracker := metaengine.NewLatencyTracker(metaengine.WithTrackerAlpha(0.5))
	remote.SetRTTTracker(tracker)
	for range 30 {
		tracker.Record(100 * time.Millisecond)
	}

	diags := store.CheckRouting(context.Background())
	if len(diags) == 0 {
		t.Fatal("expected REPLAN-SUGGESTED diagnostic, got none")
	}

	found := false
	for _, d := range diags {
		if strings.Contains(d.Message, "REPLAN-SUGGESTED") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("no REPLAN-SUGGESTED in diagnostics: %+v", diags)
	}
}

// TestStore_CheckRouting_NoSuggestionWithinDeadband proves the hysteresis
// deadband suppresses suggestions when the improvement is marginal.
func TestStore_CheckRouting_NoSuggestionWithinDeadband(t *testing.T) {
	// Two remote engines with very similar costs — within the 20% deadband.
	remoteA := newFakeRemote("a", 1_000, 1*time.Millisecond)
	remoteB := newFakeRemote("b", 1_100, 1*time.Millisecond) // 10% more expensive

	store := planWith(t, remoteA, remoteB)
	if w := winnerEngine(store); w != "a" {
		t.Fatalf("winner = %q, want a", w)
	}

	// No live shift — the gap is only 10%, below the 20% deadband.
	diags := store.CheckRouting(context.Background())

	for _, d := range diags {
		if strings.Contains(d.Message, "REPLAN-SUGGESTED") {
			t.Fatalf("unexpected REPLAN-SUGGESTED within deadband: %s", d.Message)
		}
	}
}

// TestStore_StartAutoReplan_StopsCleanly proves the auto-replan goroutine can
// be started and stopped without leaking.
func TestStore_StartAutoReplan_StopsCleanly(t *testing.T) {
	remote := newFakeRemote("remote", 1, 1*time.Millisecond)
	store := planWith(t, remote)

	stop := store.StartAutoReplan(context.Background(), 10*time.Millisecond)
	stop()
	// Calling stop twice should not panic.
	stop()
}

// TestLiveLatency_FreshIsRTTSpecific proves the OR-semantics fix: a
// read-only tracker does not make LiveLatency.Fresh true.
func TestLiveLatency_FreshIsRTTSpecific(t *testing.T) {
	eng := newFakeRemote("remote", 1, 1*time.Millisecond)

	// Install ONLY a read tracker (no RTT tracker).
	readTracker := metaengine.NewLatencyTracker(metaengine.WithStaleAfter(time.Minute))
	eng.SetReadTracker(readTracker)
	for range 5 {
		readTracker.Record(3 * time.Millisecond)
	}

	live := eng.LiveLatency()
	if live.HasRead != true {
		t.Errorf("HasRead = false, want true (read tracker installed)")
	}
	if live.HasRTT != false {
		t.Errorf("HasRTT = true, want false (no RTT tracker)")
	}
	if live.Fresh {
		t.Error("Fresh = true with only read tracker, want false (RTT-specific)")
	}
}

// TestEngineStats_StaleWithoutFreshRTT proves buildEngineStats marks a remote
// engine stale when it has a read tracker but no fresh RTT tracker.
func TestEngineStats_StaleWithoutFreshRTT(t *testing.T) {
	remote := newFakeRemote("remote", 1, 1*time.Millisecond)
	store := planWith(t, remote)

	// Install ONLY a read tracker.
	readTracker := metaengine.NewLatencyTracker(metaengine.WithStaleAfter(time.Minute))
	remote.SetReadTracker(readTracker)
	for range 5 {
		readTracker.Record(3 * time.Millisecond)
	}

	stats := store.GetEngineStats(context.Background())
	found := false
	for _, s := range stats {
		if s.Name == "remote" {
			if !s.Stale {
				t.Errorf("remote engine should be stale (no fresh RTT), got Stale=false")
			}
			found = true
		}
	}
	if !found {
		t.Fatal("remote engine not found in stats")
	}
}

// TestProbeOptions_TuneTracker proves WithProbeWindow/WithProbeAlpha/
// WithProbeStale configure the trackers created by ProbeEngine.
func TestProbeOptions_TuneTracker(t *testing.T) {
	eng := newFakeRemote("remote", 1, 1*time.Millisecond)
	eng.probeRTT = 5 * time.Millisecond

	// Probe with custom window, alpha, and stale settings.
	stop := metaengine.ProbeEngine(eng,
		metaengine.WithProbeInterval(5*time.Millisecond),
		metaengine.WithProbeTimeout(time.Second),
		metaengine.WithProbeJitter(0),
		metaengine.WithProbeWindow(32),
		metaengine.WithProbeAlpha(0.5),
		metaengine.WithProbeStale(time.Minute),
	)
	defer stop.Stop()

	// Wait for samples to accumulate.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		live := eng.LiveLatency()
		if live.HasRTT && live.RTT.Samples > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	live := eng.LiveLatency()
	if !live.HasRTT {
		t.Fatal("expected RTT tracker installed by ProbeEngine")
	}
	if live.RTT.Samples > 32 {
		t.Errorf("Samples = %d, want <= 32 (tuned window)", live.RTT.Samples)
	}
	if !live.Fresh {
		t.Error("expected fresh with stale=1min")
	}
}

// TestStore_Replan_PreservesQueryCount proves Replan doesn't lose queries.
func TestStore_Replan_PreservesQueryCount(t *testing.T) {
	remote := newFakeRemote("remote", 1, 1*time.Millisecond)
	store := planWith(t, remote)

	before := len(store.Plan().Queries)

	if err := store.Replan(context.Background()); err != nil {
		t.Fatalf("Replan: %v", err)
	}

	after := len(store.Plan().Queries)
	if after != before {
		t.Errorf("query count changed: before=%d after=%d", before, after)
	}
}

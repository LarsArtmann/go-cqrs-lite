package metaengine_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestGetEngineStats_ReportsLiveMeasurement proves GetEngineStats surfaces the
// live RTT snapshot (samples, EWMA) for a probed engine.
func TestGetEngineStats_ReportsLiveMeasurement(t *testing.T) {
	remote := newFakeRemote("pg", 1_000, 1*time.Millisecond)
	tracker := metaengine.NewLatencyTracker()
	remote.SetRTTTracker(tracker)
	for range 5 {
		tracker.Record(3 * time.Millisecond)
	}

	store := planWith(t, remote)

	stats := store.GetEngineStats(context.Background())
	if len(stats) != 1 {
		t.Fatalf("GetEngineStats returned %d engines, want 1", len(stats))
	}

	st := stats[0]
	if st.Name != "pg" {
		t.Errorf("Name = %q, want pg", st.Name)
	}

	if !st.HasLiveRTT {
		t.Error("HasLiveRTT = false, want true")
	}

	if st.Samples != 5 {
		t.Errorf("Samples = %d, want 5", st.Samples)
	}

	if st.MeasuredRTT.EWMA < 2*time.Millisecond || st.MeasuredRTT.EWMA > 4*time.Millisecond {
		t.Errorf("MeasuredRTT.EWMA = %s, want ~3ms", st.MeasuredRTT.EWMA)
	}

	if st.Stale {
		t.Error("Stale = true for fresh tracker, want false")
	}
}

// TestGetEngineStats_MarksRemoteStaleWithoutTracker proves a remote engine with
// no live measurement is labelled stale so the operator is not silently misled.
func TestGetEngineStats_MarksRemoteStaleWithoutTracker(t *testing.T) {
	remote := newFakeRemote("pg", 1_000, 1*time.Millisecond) // no SetRTTTracker

	store := planWith(t, remote)

	st := store.GetEngineStats(context.Background())[0]
	if !st.Profile.IsRemote() {
		t.Fatal("engine should be remote")
	}

	if !st.Stale {
		t.Error("Stale = false for remote engine with no tracker, want true")
	}

	if st.HasLiveRTT {
		t.Error("HasLiveRTT = true for remote engine with no tracker, want false")
	}
}

// TestFormatLiveLatency covers the three display branches: live, stale-prior,
// and local.
func TestFormatLiveLatency(t *testing.T) {
	t.Run("local", func(t *testing.T) {
		local := metaengine.NewMemoryEngine()
		store := planWith(t, local)
		st := store.GetEngineStats(context.Background())[0]

		got := metaengine.FormatLiveLatency(st)
		if !strings.Contains(got, "local") {
			t.Errorf("local format = %q, want to contain 'local'", got)
		}
	})

	t.Run("live", func(t *testing.T) {
		remote := newFakeRemote("pg", 1_000, 1*time.Millisecond)
		tracker := metaengine.NewLatencyTracker(metaengine.WithStaleAfter(time.Minute))
		remote.SetRTTTracker(tracker)
		for range 10 {
			tracker.Record(2 * time.Millisecond)
		}

		store := planWith(t, remote)
		st := store.GetEngineStats(context.Background())[0]

		got := metaengine.FormatLiveLatency(st)
		if !strings.Contains(got, "rtt=live") {
			t.Errorf("live format = %q, want 'rtt=live'", got)
		}

		if !strings.Contains(got, "n=10") {
			t.Errorf("live format = %q, want sample count n=10", got)
		}
	})

	t.Run("stale prior", func(t *testing.T) {
		remote := newFakeRemote("pg", 1_000, 1*time.Millisecond) // no tracker

		store := planWith(t, remote)
		st := store.GetEngineStats(context.Background())[0]

		got := metaengine.FormatLiveLatency(st)
		if !strings.Contains(got, "stale") {
			t.Errorf("stale format = %q, want 'stale'", got)
		}
	})
}

// TestDoctor_IncludesLatencySection proves the Doctor report surfaces a
// per-engine live-latency line.
func TestDoctor_IncludesLatencySection(t *testing.T) {
	remote := newFakeRemote("pg", 1_000, 1*time.Millisecond)

	store := planWith(t, remote)

	report := store.Doctor(context.Background())
	if !strings.Contains(report, "--- Latency ---") {
		t.Errorf("Doctor report missing Latency section:\n%s", report)
	}

	if !strings.Contains(report, "pg:") {
		t.Errorf("Doctor report missing engine line:\n%s", report)
	}
}

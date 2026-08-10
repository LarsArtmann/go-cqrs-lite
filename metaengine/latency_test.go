package metaengine_test

import (
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func TestLatencyTracker_RecordsAndSnapshots(t *testing.T) {
	tracker := metaengine.NewLatencyTracker()

	for _, d := range []time.Duration{
		1, 2, 3, 4, 5, 6, 7, 8, 9, 10,
	} {
		tracker.Record(d * time.Millisecond)
	}

	stats := tracker.Snapshot()

	if stats.Samples != 10 {
		t.Fatalf("Samples = %d, want 10", stats.Samples)
	}

	if got := stats.Max; got != 10*time.Millisecond {
		t.Errorf("Max = %s, want 10ms", got)
	}

	if got := stats.P50; got != 5*time.Millisecond {
		t.Errorf("P50 = %s, want 5ms", got)
	}

	if got := stats.P95; got != 9*time.Millisecond {
		t.Errorf("P95 = %s, want 9ms", got)
	}

	// Mean of 1..10 ms = 5.5ms.
	wantMean := 5500 * time.Microsecond
	if got := stats.Mean; got != wantMean {
		t.Errorf("Mean = %s, want %s", got, wantMean)
	}
}

func TestLatencyTracker_EWMAConvergesToRecentValue(t *testing.T) {
	tracker := metaengine.NewLatencyTracker(metaengine.WithTrackerAlpha(0.1))

	// Seed the EWMA at 1ms, then flood with 50ms samples. After enough samples
	// the EWMA must be dominated by the recent 50ms value.
	for range 5 {
		tracker.Record(1 * time.Millisecond)
	}

	for range 100 {
		tracker.Record(50 * time.Millisecond)
	}

	stats := tracker.Snapshot()

	if stats.EWMA < 49*time.Millisecond || stats.EWMA > 51*time.Millisecond {
		t.Errorf("EWMA = %s, want ~50ms (recent-dominant)", stats.EWMA)
	}
}

func TestLatencyTracker_WindowEvictionKeepsRecentOnly(t *testing.T) {
	tracker := metaengine.NewLatencyTracker(metaengine.WithTrackerWindow(4))
	tracker.Record(1 * time.Millisecond)
	tracker.Record(2 * time.Millisecond)
	tracker.Record(3 * time.Millisecond)
	tracker.Record(4 * time.Millisecond)
	// Window is full (4). Next record evicts the oldest (1ms).
	tracker.Record(100 * time.Millisecond)

	stats := tracker.Snapshot()

	if stats.Samples != 4 {
		t.Fatalf("Samples = %d, want 4 (windowed)", stats.Samples)
	}

	if stats.Max != 100*time.Millisecond {
		t.Errorf("Max = %s, want 100ms", stats.Max)
	}

	// The 1ms sample must be gone: min is now 2ms. Check via P50 of [2,3,4,100].
	if stats.P50 != 3*time.Millisecond {
		t.Errorf("P50 = %s, want 3ms (oldest evicted)", stats.P50)
	}
}

func TestLatencyTracker_Freshness(t *testing.T) {
	t.Run("empty tracker is not fresh", func(t *testing.T) {
		tracker := metaengine.NewLatencyTracker()
		if tracker.Fresh() {
			t.Fatal("empty tracker reported fresh")
		}

		if _, ok := tracker.Live(); ok {
			t.Fatal("empty tracker Live() reported ok")
		}
	})

	t.Run("recent sample is fresh", func(t *testing.T) {
		tracker := metaengine.NewLatencyTracker(metaengine.WithStaleAfter(time.Second))
		tracker.Record(5 * time.Millisecond)

		if !tracker.Fresh() {
			t.Fatal("tracker with recent sample reported stale")
		}

		if stats, ok := tracker.Live(); !ok {
			t.Fatal("Live() reported not-ok for fresh tracker")
		} else if stats.Samples != 1 {
			t.Errorf("Live Samples = %d, want 1", stats.Samples)
		}
	})

	t.Run("old sample goes stale", func(t *testing.T) {
		tracker := metaengine.NewLatencyTracker(metaengine.WithStaleAfter(2 * time.Millisecond))
		tracker.Record(5 * time.Millisecond)
		time.Sleep(10 * time.Millisecond)

		if tracker.Fresh() {
			t.Fatal("tracker with old sample reported fresh")
		}

		if _, ok := tracker.Live(); ok {
			t.Fatal("stale tracker Live() reported ok")
		}
	})
}

func TestLatencyTracker_StatSinkIngress(t *testing.T) {
	sink := &recordingSink{}
	tracker := metaengine.NewLatencyTracker(
		metaengine.WithTrackerSink("test-engine", sink),
	)

	tracker.Record(3 * time.Millisecond)
	tracker.Record(7 * time.Millisecond)

	sink.mu.Lock()
	defer sink.mu.Unlock()

	if len(sink.samples) != 2 {
		t.Fatalf("sink received %d samples, want 2", len(sink.samples))
	}

	if sink.samples[0].Value != 3*time.Millisecond {
		t.Errorf("first sample = %s, want 3ms", sink.samples[0].Value)
	}

	if sink.samples[0].Kind != metaengine.SampleRTT {
		t.Errorf("sample Kind = %v, want SampleRTT", sink.samples[0].Kind)
	}
}

// recordingSink collects samples for test assertions (P3 ingress path).
type recordingSink struct {
	mu      sync.Mutex
	samples []metaengine.LatencySample
}

func (r *recordingSink) ReportSample(_ string, s metaengine.LatencySample) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.samples = append(r.samples, s)
}

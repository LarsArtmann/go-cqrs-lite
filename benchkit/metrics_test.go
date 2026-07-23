package benchkit

import (
	"testing"
	"time"
)

func TestLatencyCollector_Empty(t *testing.T) {
	t.Parallel()

	lc := NewLatencyCollector(0)
	stats := lc.Stats()

	if stats.Count != 0 {
		t.Errorf("Count = %d, want 0", stats.Count)
	}
}

func TestLatencyCollector_BasicPercentiles(t *testing.T) {
	t.Parallel()

	lc := NewLatencyCollector(1000)

	for i := range 100 {
		lc.Record(time.Duration(i+1) * time.Microsecond)
	}

	stats := lc.Stats()

	if stats.Count != 100 {
		t.Errorf("Count = %d, want 100", stats.Count)
	}

	// For values 1..100µs, P50 should be around 50µs
	if stats.P50 < 40*time.Microsecond || stats.P50 > 60*time.Microsecond {
		t.Errorf("P50 = %v, want ~50µs", stats.P50)
	}

	// P100 should be 100µs
	if stats.P100 != 100*time.Microsecond {
		t.Errorf("P100 = %v, want 100µs", stats.P100)
	}

	// Mean should be ~50.5µs
	expected := time.Duration(int64(50.5 * float64(time.Microsecond)))
	if stats.Mean < expected-time.Microsecond || stats.Mean > expected+time.Microsecond {
		t.Errorf("Mean = %v, want ~%v", stats.Mean, expected)
	}

	// Percentiles should be ordered
	if stats.P50 > stats.P99 {
		t.Error("P50 > P99")
	}
}

func TestLatencyCollector_ReservoirSampling(t *testing.T) {
	t.Parallel()

	maxLen := 100
	lc := NewLatencyCollector(maxLen)

	for i := range 10_000 {
		lc.Record(time.Duration(i+1) * time.Nanosecond)
	}

	stats := lc.Stats()

	if stats.Count != 10_000 {
		t.Errorf("Count = %d, want 10000", stats.Count)
	}

	// The samples slice should be bounded by maxLen
	lc.mu.Lock()
	sampleCount := len(lc.samples)
	lc.mu.Unlock()

	if sampleCount > maxLen {
		t.Errorf("stored samples = %d, want <= %d", sampleCount, maxLen)
	}

	// P100 should be <= 10000ns (may not be exactly 10000 due to reservoir)
	if stats.P100 > 10_000*time.Nanosecond {
		t.Errorf("P100 = %v, want <= 10000ns", stats.P100)
	}

	// Mean should be approximately 5000ns
	if stats.Mean < 4_000*time.Nanosecond || stats.Mean > 6_000*time.Nanosecond {
		t.Errorf("Mean = %v, want ~5000ns", stats.Mean)
	}
}

func TestLatencyCollector_Concurrent(t *testing.T) {
	t.Parallel()

	lc := NewLatencyCollector(10_000)
	done := make(chan struct{})

	for range 10 {
		go func() {
			for range 100 {
				lc.Record(time.Microsecond)
			}

			done <- struct{}{}
		}()
	}

	for range 10 {
		<-done
	}

	stats := lc.Stats()

	if stats.Count != 1000 {
		t.Errorf("Count = %d, want 1000", stats.Count)
	}
}

func TestPercentile_SingleElement(t *testing.T) {
	t.Parallel()

	sorted := []time.Duration{42 * time.Microsecond}

	if p := percentile(sorted, 50); p != 42*time.Microsecond {
		t.Errorf("P50 = %v, want 42µs", p)
	}

	if p := percentile(sorted, 99); p != 42*time.Microsecond {
		t.Errorf("P99 = %v, want 42µs", p)
	}
}

func TestPercentile_Empty(t *testing.T) {
	t.Parallel()

	sorted := []time.Duration{}

	if p := percentile(sorted, 50); p != 0 {
		t.Errorf("P50 = %v, want 0", p)
	}
}

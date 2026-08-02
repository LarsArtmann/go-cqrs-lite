package metaengine

import (
	"testing"
	"time"
)

func TestReplicationDefaultsToNone(t *testing.T) {
	t.Parallel()

	// Every EngineProfile that doesn't set Replication defaults to ReplicationNone.
	// This is the backward-compatibility guarantee: all existing engines are local.
	p := EngineProfile{Name: "test"}
	if p.Replication != ReplicationNone {
		t.Fatalf(
			"zero-value Replication = %q, want %q (ReplicationNone)",
			p.Replication,
			ReplicationNone,
		)
	}
	if p.IsReplicated() {
		t.Fatal("zero-value EngineProfile should not be replicated")
	}
}

func TestReplicationModeIsReplicated(t *testing.T) {
	t.Parallel()

	modes := []Replication{
		ReplicationSingleLeader,
		ReplicationMultiLeader,
		ReplicationLeaderless,
	}

	for _, mode := range modes {
		p := EngineProfile{Name: "test", Replication: mode}
		if !p.IsReplicated() {
			t.Errorf("Replication %q should be replicated", mode)
		}
	}
}

func TestReplicationNoneIsZeroValue(t *testing.T) {
	t.Parallel()

	// ReplicationNone is "" so the zero value IS none — no helper needed.
	// This must never change, or every existing engine profile breaks.
	var r Replication
	if r != ReplicationNone {
		t.Fatalf("zero-value Replication = %q, want ReplicationNone (%q)", r, ReplicationNone)
	}
}

func TestEstimateCostWithNetworkRTT(t *testing.T) {
	t.Parallel()

	// NetworkRTT is additive — it doesn't scale with volume.
	// Two engines with identical compute cost but different RTT:
	// the one with higher RTT should have proportionally higher latency.
	base := estimateCost(ComplexityO1, 1000, 500.0, 0)
	withRTT := estimateCost(ComplexityO1, 1000, 500.0, 1*time.Millisecond)

	rttMs := float64(time.Millisecond.Microseconds()) / 1e3

	if withRTT.EstimatedLatencyMs != base.EstimatedLatencyMs+rttMs {
		t.Fatalf(
			"RTT not additive: base=%.4fms withRTT=%.4fms expected_delta=%.4fms",
			base.EstimatedLatencyMs, withRTT.EstimatedLatencyMs, rttMs,
		)
	}
}

func TestEstimateCostNetworkRTTDoesNotScaleWithVolume(t *testing.T) {
	t.Parallel()

	// A scan (O(N)) with RTT: the RTT component should be constant
	// regardless of whether N=100 or N=10000.
	rtt := 10 * time.Millisecond
	small := estimateCost(ComplexityON, 100, 7000.0, rtt)
	large := estimateCost(ComplexityON, 10000, 7000.0, rtt)

	rttMs := float64(rtt.Microseconds()) / 1e3

	smallWithoutRTT := small.EstimatedLatencyMs - rttMs
	largeWithoutRTT := large.EstimatedLatencyMs - rttMs

	// Without RTT, the large scan should be 100x more expensive (10000 vs 100 items)
	if smallWithoutRTT <= 0 {
		t.Fatalf("small scan compute cost should be positive, got %.4fms", smallWithoutRTT)
	}

	ratio := largeWithoutRTT / smallWithoutRTT
	if ratio < 99 || ratio > 101 {
		t.Fatalf("volume scaling broken: ratio=%.1f (expected ~100x)", ratio)
	}

	// With RTT, the ratio should be LESS than 100x because RTT is constant
	ratioWithRTT := large.EstimatedLatencyMs / small.EstimatedLatencyMs
	if ratioWithRTT >= ratio {
		t.Fatalf(
			"RTT should dampen volume scaling: ratioWithRTT=%.1f ratioWithoutRTT=%.1f",
			ratioWithRTT, ratio,
		)
	}
}

func TestLocalEngineProfileDefaults(t *testing.T) {
	t.Parallel()

	// All current production engines should have:
	// - Replication = ReplicationNone (zero value)
	// - ReplicationLag = 0
	// - NetworkRTT = 0
	profiles := []struct {
		name    string
		profile EngineProfile
	}{
		{"memory", (&memoryEngine{}).Profile()},
		{"sqlite", SQLiteEngineProfile()},
	}

	for _, tc := range profiles {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if tc.profile.Replication != ReplicationNone {
				t.Errorf(
					"%s: Replication = %q, want ReplicationNone",
					tc.name,
					tc.profile.Replication,
				)
			}

			if tc.profile.ReplicationLag != 0 {
				t.Errorf("%s: ReplicationLag = %v, want 0", tc.name, tc.profile.ReplicationLag)
			}

			if tc.profile.NetworkRTT != 0 {
				t.Errorf("%s: NetworkRTT = %v, want 0", tc.name, tc.profile.NetworkRTT)
			}
		})
	}
}

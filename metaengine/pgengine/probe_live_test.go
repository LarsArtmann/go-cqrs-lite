package pgengine_test

import (
	"context"
	"strings"
	"testing"
	"time"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// probeEvt and probeIn are minimal types for a Map query used only to create a
// Store so GetEngineStats has an engine to report on. The query is never
// executed — the test exercises the probe loop, not query routing.
type probeEvt struct {
	Category string
	Count    int64
}

type probeIn struct{}

// newProbeStore creates a Store with a single Map query backed by eng.
func newProbeStore(t *testing.T, eng metaengine.Engine) *metaengine.Store {
	t.Helper()

	store, err := metaengine.Plan(
		[]metaengine.Engine{eng},
		metaengine.Query[probeIn, map[string]int64](
			"pg_probe_live_rtt",
			metaengine.OnRecord(
				probeEvt{},
				func(_ record.Record, e probeEvt) metaengine.Delta {
					return metaengine.Delta{e.Category: e.Count}
				},
			),
		),
	)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	return store
}

// waitForLiveRTT polls GetEngineStats until the engine reports a fresh live RTT
// with at least one sample, or fails the test on timeout.
func waitForLiveRTT(
	t *testing.T,
	ctx context.Context,
	store *metaengine.Store,
	timeout time.Duration,
) metaengine.EngineStats {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		stats := store.GetEngineStats(ctx)
		if len(stats) > 0 && stats[0].HasLiveRTT && stats[0].Samples > 0 {
			return stats[0]
		}

		time.Sleep(50 * time.Millisecond)
	}

	t.Fatalf("no live RTT after %s; probe loop did not produce samples", timeout)

	return metaengine.EngineStats{}
}

// TestProbeEngine_RealPostgres_LiveRTT proves the live-latency measurement loop
// works end-to-end against a real Postgres instance — not just the
// fakeRemoteEngine in the metaengine package's unit tests.
//
// Verifies that ProbeEngine's background probes produce real RTT samples via
// PingContext, that GetEngineStats surfaces them as a fresh live measurement,
// that FormatLiveLatency renders the "rtt=live" branch, that the
// TransactMeasurer read-latency tracker is populated, and that no probe errors
// occur against a healthy server.
func TestProbeEngine_RealPostgres_LiveRTT(t *testing.T) {
	t.Parallel()

	eng := mustNewPgEngine(t)
	store := newProbeStore(t, eng)
	defer store.Close()

	ctx := context.Background()

	// Before probing: remote engine must show as stale with no live RTT.
	pre := store.GetEngineStats(ctx)[0]
	if pre.HasLiveRTT {
		t.Fatal("HasLiveRTT should be false before probing starts")
	}

	if !pre.Stale {
		t.Fatal("Stale should be true before probing (no live measurement yet)")
	}

	// Start the probe loop with a short interval for fast warmup.
	handle := metaengine.ProbeEngine(eng,
		metaengine.WithProbeInterval(15*time.Millisecond),
		metaengine.WithProbeTimeout(5*time.Second),
		metaengine.WithProbeJitter(0),
	)
	defer handle.Stop()

	stats := waitForLiveRTT(t, ctx, store, 10*time.Second)

	if !stats.HasLiveRTT {
		t.Fatal("HasLiveRTT = false after probing; probe loop did not record RTT")
	}

	if stats.Samples == 0 {
		t.Fatal("Samples = 0; probe loop recorded nothing")
	}

	if stats.Stale {
		t.Error("Stale = true; want false (fresh live measurement)")
	}

	if stats.MeasuredRTT.EWMA <= 0 {
		t.Errorf("MeasuredRTT.EWMA = %s, want > 0", stats.MeasuredRTT.EWMA)
	}

	if stats.MeasuredRTT.EWMA > 50*time.Millisecond {
		t.Errorf(
			"MeasuredRTT.EWMA = %s, want < 50ms for local PG container",
			stats.MeasuredRTT.EWMA,
		)
	}

	formatted := metaengine.FormatLiveLatency(stats)
	if !strings.Contains(formatted, "rtt=live") {
		t.Errorf("FormatLiveLatency = %q, want to contain 'rtt=live'", formatted)
	}

	if !strings.Contains(formatted, "n=") {
		t.Errorf("FormatLiveLatency = %q, want sample count 'n='", formatted)
	}

	// pgengine implements TransactMeasurer, so per-read latency is also tracked.
	if !stats.HasLiveRead {
		t.Error("HasLiveRead = false; pgengine implements TransactMeasurer")
	}

	if stats.MeasuredRead.EWMA <= 0 {
		t.Errorf("MeasuredRead.EWMA = %s, want > 0", stats.MeasuredRead.EWMA)
	}

	// No probe errors against a healthy PG.
	if failures := handle.Failures(); failures != 0 {
		t.Errorf("probe failures = %d, want 0 (healthy PG)", failures)
	}
}

// TestProbeEngine_RealPostgres_StaleAfterStop proves that once the probe loop is
// stopped, the measurement transitions to stale after the configured
// stale-after window — confirming GetEngineStats will not silently report a
// stale measurement as fresh indefinitely.
func TestProbeEngine_RealPostgres_StaleAfterStop(t *testing.T) {
	t.Parallel()

	eng := mustNewPgEngine(t)
	store := newProbeStore(t, eng)
	defer store.Close()

	ctx := context.Background()

	staleAfter := 300 * time.Millisecond
	handle := metaengine.ProbeEngine(eng,
		metaengine.WithProbeInterval(15*time.Millisecond),
		metaengine.WithProbeTimeout(5*time.Second),
		metaengine.WithProbeJitter(0),
		metaengine.WithProbeStale(staleAfter),
	)

	waitForLiveRTT(t, ctx, store, 10*time.Second)

	handle.Stop()

	// Wait beyond the stale-after window for the last sample to expire.
	time.Sleep(staleAfter + 200*time.Millisecond)

	stats := store.GetEngineStats(ctx)[0]
	if !stats.Stale {
		t.Fatalf("Stale = false after stopping probe + %s; want true", staleAfter)
	}
}

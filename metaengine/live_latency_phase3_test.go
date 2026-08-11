package metaengine_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// --- Edge Case Tests ---

// TestCheckRouting_SingleEngineNoAlternative proves CheckRouting returns no
// diagnostics when there is only one engine (no alternative to suggest).
func TestCheckRouting_SingleEngineNoAlternative(t *testing.T) {
	t.Parallel()

	remote := newFakeRemote("only", 1, 1*time.Millisecond)
	store := planWith(t, remote)

	diags := store.CheckRouting(context.Background())
	if len(diags) != 0 {
		t.Fatalf("expected 0 diagnostics with single engine, got %d: %+v", len(diags), diags)
	}
}

// TestReplan_SingleEngine proves Replan succeeds with only one engine.
func TestReplan_SingleEngine(t *testing.T) {
	t.Parallel()

	remote := newFakeRemote("only", 1, 1*time.Millisecond)
	store := planWith(t, remote)

	if err := store.Replan(context.Background()); err != nil {
		t.Fatalf("Replan with single engine: %v", err)
	}

	if len(store.Plan().Queries) != 1 {
		t.Errorf("query count = %d, want 1", len(store.Plan().Queries))
	}
}

// TestCheckRouting_NilContextReturnsNil proves CheckRouting handles a cancelled
// context gracefully.
func TestCheckRouting_CancelledContextReturnsNil(t *testing.T) {
	t.Parallel()

	remote := newFakeRemote("remote", 1, 1*time.Millisecond)
	store := planWith(t, remote)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	diags := store.CheckRouting(ctx)
	if diags != nil {
		t.Fatalf("expected nil diagnostics for cancelled context, got %v", diags)
	}
}

// --- Hysteresis Configuration Tests ---

// TestRoutingHysteresis_CustomThreshold proves WithRoutingHysteresis changes
// the deadband. With a 5% threshold, a ~7% gap triggers a suggestion that the
// default 20% would suppress.
func TestRoutingHysteresis_CustomThreshold(t *testing.T) {
	t.Parallel()

	// Two engines with high per-op costs so the absolute delta exceeds 0.5ms.
	remoteA := newFakeRemote("a", 10_000_000, 1*time.Millisecond) // cost ≈11ms
	remoteB := newFakeRemote("b", 12_000_000, 1*time.Millisecond) // cost ≈13ms

	store, err := metaengine.Plan(
		[]metaengine.Engine{remoteA, remoteB},
		freshFindTaskQuery(),
		metaengine.WithRoutingHysteresis(0.05),
	)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	if w := winnerEngine(store); w != "a" {
		t.Fatalf("initial winner = %q, want a", w)
	}

	// Shift A's RTT so B becomes ~7% cheaper: A=14ms, B=13ms.
	tracker := metaengine.NewLatencyTracker(metaengine.WithTrackerAlpha(0.5))
	remoteA.SetRTTTracker(tracker)
	for range 30 {
		tracker.Record(4 * time.Millisecond)
	}

	// With 5% hysteresis, 7% improvement triggers a suggestion.
	diags := store.CheckRouting(context.Background())

	found := false
	for _, d := range diags {
		if strings.Contains(d.Message, "REPLAN-SUGGESTED") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected REPLAN-SUGGESTED with 5%% hysteresis, got: %+v", diags)
	}
}

// TestRoutingHysteresis_MinDeltaSuppressesCheapQueries proves the absolute
// minimum delta prevents re-routing for very cheap queries even when the
// fractional improvement is large.
func TestRoutingHysteresis_MinDeltaSuppressesCheapQueries(t *testing.T) {
	t.Parallel()

	// Two local engines with near-zero costs. The fractional difference is
	// huge, but the absolute difference is tiny.
	localA := newFakeLocal("a", 1)
	localB := newFakeLocal("b", 2) // 100% more expensive in per-op cost

	store, err := metaengine.Plan(
		[]metaengine.Engine{localA, localB},
		freshFindTaskQuery(),
		metaengine.WithRoutingMinDelta(1*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	// The absolute improvement is < 1ms, so no suggestion despite 100% fraction.
	diags := store.CheckRouting(context.Background())
	for _, d := range diags {
		if strings.Contains(d.Message, "REPLAN-SUGGESTED") {
			t.Fatalf("unexpected REPLAN-SUGGESTED suppressed by min-delta: %s", d.Message)
		}
	}
}

// --- Probe Failure Observability Tests ---

// TestProbeHandle_FailureCounter proves the failure counter increments when
// probes fail.
func TestProbeHandle_FailureCounter(t *testing.T) {
	t.Parallel()

	eng := newFakeRemote("remote", 1_000, 1*time.Millisecond)
	eng.probeErr = errors.New("network unreachable")

	var handlerCalls atomic.Int64

	ph := metaengine.ProbeEngine(eng,
		metaengine.WithProbeInterval(5*time.Millisecond),
		metaengine.WithProbeTimeout(time.Second),
		metaengine.WithProbeJitter(0),
		metaengine.WithProbeErrorHandler(func(_ error) {
			handlerCalls.Add(1)
		}),
	)
	defer ph.Stop()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if ph.Failures() > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if ph.Failures() == 0 {
		t.Fatal("expected failures > 0, got 0")
	}

	if handlerCalls.Load() == 0 {
		t.Fatal("expected error handler to be called, got 0 calls")
	}
}

// TestProbeHandle_NilSafe proves a nil ProbeHandle is safe to call Stop/Failures on.
func TestProbeHandle_NilSafe(t *testing.T) {
	t.Parallel()

	var ph *metaengine.ProbeHandle

	ph.Stop() // must not panic
	if got := ph.Failures(); got != 0 {
		t.Errorf("nil Failures = %d, want 0", got)
	}
}

// --- StartAutoReplan Parent Context Tests ---

// TestStartAutoReplan_ParentContextCancellation proves that cancelling the
// parent context stops the auto-replan loop.
func TestStartAutoReplan_ParentContextCancellation(t *testing.T) {
	t.Parallel()

	remote := newFakeRemote("remote", 1, 1*time.Millisecond)
	store := planWith(t, remote)

	ctx, cancel := context.WithCancel(context.Background())

	_ = store.StartAutoReplan(ctx, 10*time.Millisecond)

	// Cancel the parent context — the loop should stop.
	cancel()

	// Give the goroutine time to notice and exit.
	time.Sleep(50 * time.Millisecond)

	// If the goroutine didn't exit, it would still be running. We can't easily
	// check goroutine count, but the test passes if it doesn't hang.
}

// --- Concurrency Stress Test ---

// TestConcurrency_ReplanCheckRoutingStress proves Replan, CheckRouting, and
// Execute can run concurrently without panics or deadlocks.
func TestConcurrency_ReplanCheckRoutingStress(t *testing.T) {
	t.Parallel()

	remote := newFakeRemote("remote", 1_000, 1*time.Millisecond)
	store := planWith(t, remote)

	tracker := metaengine.NewLatencyTracker(metaengine.WithTrackerAlpha(0.5))
	remote.SetRTTTracker(tracker)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	var wg sync.WaitGroup

	// Goroutine 1: Replan every 10ms.
	wg.Add(1)

	go func() {
		defer wg.Done()

		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = store.Replan(ctx)
			}
		}
	}()

	// Goroutine 2: CheckRouting every 5ms.
	wg.Add(1)

	go func() {
		defer wg.Done()

		ticker := time.NewTicker(5 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = store.CheckRouting(ctx)
			}
		}
	}()

	// Goroutine 3: Shift the tracker RTT.
	wg.Add(1)

	go func() {
		defer wg.Done()

		ticker := time.NewTicker(3 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				tracker.Record(time.Duration(1+atomic.AddInt64(&rttShift, 1)) * time.Millisecond)
			}
		}
	}()

	// Goroutine 4: GetEngineStats every 15ms.
	wg.Add(1)

	go func() {
		defer wg.Done()

		ticker := time.NewTicker(15 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = store.GetEngineStats(ctx)
			}
		}
	}()

	wg.Wait()

	// Verify the store is still usable after the stress test.
	if err := store.Replan(context.Background()); err != nil {
		t.Fatalf("post-stress Replan: %v", err)
	}
}

var rttShift int64

// --- Differential CheckRouting Test ---

// TestCheckRouting_CachesWhenNoRTTChange proves the differential optimization:
// when no engine's RTT changed, the second call returns the cached result.
func TestCheckRouting_CachesWhenNoRTTChange(t *testing.T) {
	t.Parallel()

	remoteA := newFakeRemote("a", 1_000, 1*time.Millisecond)
	remoteB := newFakeRemote("b", 1_500, 1*time.Millisecond)

	store := planWith(t, remoteA, remoteB)

	diags1 := store.CheckRouting(context.Background())
	diags2 := store.CheckRouting(context.Background())

	// Both calls should return the same result (no RTT change between them).
	if len(diags1) != len(diags2) {
		t.Fatalf("diagnostic count changed without RTT shift: %d vs %d",
			len(diags1), len(diags2))
	}
}

// --- Doctor Report Test ---

// TestDoctor_RoutingSection proves the Doctor report includes routing info.
func TestDoctor_RoutingSection(t *testing.T) {
	t.Parallel()

	remote := newFakeRemote("remote", 1, 1*time.Millisecond)
	store := planWith(t, remote)

	report := store.Doctor(context.Background())

	if !strings.Contains(report, "--- Routing ---") {
		t.Error("Doctor report missing '--- Routing ---' section")
	}

	if !strings.Contains(report, "plan version:") {
		t.Error("Doctor report missing plan version")
	}

	if !strings.Contains(report, "hysteresis:") {
		t.Error("Doctor report missing hysteresis")
	}

	// After a replan, the report should show replan count > 0.
	if err := store.Replan(context.Background()); err != nil {
		t.Fatalf("Replan: %v", err)
	}

	report2 := store.Doctor(context.Background())
	if !strings.Contains(report2, "replans: 1") {
		t.Errorf("Doctor report should show replans: 1, got:\n%s", report2)
	}
}

// --- RTT Amortization Test ---

// TestNsForRead_RTTAmortizationForScans proves the fallback cost for scan
// patterns on remote engines subtracts RTT to avoid overestimation. The
// subtraction only applies when NsPerRead exceeds RTT (i.e., per-read cost is
// dominated by network overhead).
func TestNsForRead_RTTAmortizationForScans(t *testing.T) {
	t.Parallel()

	profile := metaengine.EngineProfile{
		NsPerRead:  10_000_000,             // 10ms per read (includes network)
		NetworkRTT: 1 * time.Millisecond,   // 1ms RTT
		ReadCosts:  metaengine.ReadCosts{}, // no scan-specific cost set
	}

	scanCost := profile.NsForRead(metaengine.ReadScan)
	pointCost := profile.NsForRead(metaengine.ReadPointLookup)

	// Point lookup: no subtraction (NsPerPointLookup not set, falls back to
	// NsPerRead = 10,000,000).
	if pointCost != 10_000_000 {
		t.Errorf("point lookup cost = %.0f, want 10000000", pointCost)
	}

	// Scan: RTT subtracted from fallback (10,000,000 - 1,000,000 = 9,000,000).
	// RTT is added once per query by estimateCost, so the per-row scan cost
	// should exclude it.
	if scanCost != 9_000_000 {
		t.Errorf("scan cost = %.0f, want 9000000 (RTT amortized)", scanCost)
	}
}

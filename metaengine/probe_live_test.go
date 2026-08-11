package metaengine_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// fakeRemoteEngine is a test-double Engine that embeds Calibration (so it gets
// SetRTTTracker / LiveLatency promoted, exactly like pgengine/dgraphengine) and
// implements Prober with a test-controlled RTT. It proves the core mechanism:
// a live RTT shift changes Profile().NetworkRTT and therefore Plan() routing.
type fakeRemoteEngine struct {
	metaengine.Calibration

	name     string
	profile  metaengine.EngineProfile
	probeRTT time.Duration
	probeErr error
}

func (e *fakeRemoteEngine) Profile() metaengine.EngineProfile {
	p := e.profile
	p.Name = e.name
	e.ApplyCalibration(&p)

	return p
}

func (e *fakeRemoteEngine) Close() error { return nil }

func (e *fakeRemoteEngine) Probe(_ context.Context) (time.Duration, error) {
	if e.probeErr != nil {
		return 0, e.probeErr
	}

	return e.probeRTT, nil
}

// newFakeRemote builds a remote engine supporting Map (O(1)) with the given
// per-op cost and a declared RTT prior.
func newFakeRemote(name string, nsPerOp float64, priorRTT time.Duration) *fakeRemoteEngine {
	return &fakeRemoteEngine{
		name: name,
		profile: metaengine.EngineProfile{
			NsPerOp:         nsPerOp,
			Persistence:     metaengine.PersistencePersistent,
			RequiresNetwork: true,
			NetworkRTT:      priorRTT,
			Supports: map[metaengine.ADT]metaengine.Complexity{
				metaengine.ADTMap: metaengine.ComplexityO1,
			},
		},
	}
}

func freshFindTaskQuery() metaengine.QueryDecl[FindTask, FindTaskResult] {
	return metaengine.Query[FindTask, FindTaskResult](
		"find_task_live",
		metaengine.OnRecord(
			TaskCreated{},
			func(_ record.Record, e TaskCreated) (TaskID, FindTaskResult) {
				return e.ID, FindTaskResult{ID: e.ID, Title: e.Title}
			},
		),
		metaengine.OnRecord(TaskDeleted{}, metaengine.Remove[FindTaskResult]()),
	)
}

func planWith(t *testing.T, engines ...metaengine.Engine) *metaengine.Store {
	t.Helper()

	store, err := metaengine.Plan(engines, freshFindTaskQuery())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	return store
}

// TestLiveRTT_OverridesPriorInProfile proves the P1 gate: a live tracker
// replaces the declared NetworkRTT prior inside Profile().
func TestLiveRTT_OverridesPriorInProfile(t *testing.T) {
	eng := newFakeRemote("pg", 1_000, 1*time.Millisecond)

	if got := eng.Profile().NetworkRTT; got != 1*time.Millisecond {
		t.Fatalf("prior RTT = %s, want 1ms before probing", got)
	}

	// Install a live tracker and flood it with 5ms samples.
	tracker := metaengine.NewLatencyTracker(metaengine.WithTrackerAlpha(0.5))
	eng.SetRTTTracker(tracker)
	for range 30 {
		tracker.Record(5 * time.Millisecond)
	}

	if got := eng.Profile().NetworkRTT; got < 4*time.Millisecond || got > 6*time.Millisecond {
		t.Errorf("live RTT = %s, want ~5ms (prior overridden)", got)
	}

	live := eng.LiveLatency()
	if !live.HasRTT || !live.Fresh {
		t.Errorf("LiveLatency = %+v, want HasRTT+Fresh", live)
	}
}

// TestProbeEngine_FeedsProfileViaBackgroundLoop proves ProbeEngine wires a fake
// Prober's measurements into Profile() through the embedded Calibration.
func TestProbeEngine_FeedsProfileViaBackgroundLoop(t *testing.T) {
	eng := newFakeRemote("pg", 1_000, 1*time.Millisecond)
	eng.probeRTT = 8 * time.Millisecond

	stop := metaengine.ProbeEngine(eng,
		metaengine.WithProbeInterval(5*time.Millisecond),
		metaengine.WithProbeTimeout(time.Second),
		metaengine.WithProbeJitter(0),
	)
	defer stop.Stop()

	// Wait for the probe loop to record enough samples to dominate the prior.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if eng.Profile().NetworkRTT >= 6*time.Millisecond {
			break
		}

		time.Sleep(10 * time.Millisecond)
	}

	if got := eng.Profile().NetworkRTT; got < 6*time.Millisecond {
		t.Fatalf("Profile RTT = %s after probing, want >=6ms (probe loop not feeding tracker)", got)
	}
}

// TestProbeEngine_NoopForLocalEngine proves calling ProbeEngine on a local
// engine (no Prober) is safe and returns a working no-op stop.
func TestProbeEngine_NoopForLocalEngine(t *testing.T) {
	local := metaengine.NewMemoryEngine()
	stop := metaengine.ProbeEngine(local)
	stop.Stop() // must not block or panic

	if got := local.Profile().NetworkRTT; got != 0 {
		t.Errorf("local engine RTT = %s, want 0", got)
	}
}

// TestPlan_RoutingFlipsOnLiveRTTShift proves the P2 gate: a live RTT shift
// changes which engine the planner routes to. Both engines support the same
// query; the remote engine has cheaper compute but tunable RTT.
func TestPlan_RoutingFlipsOnLiveRTTShift(t *testing.T) {
	// Remote: extremely cheap compute, low prior RTT → wins initially.
	remote := newFakeRemote("remote", 1, 500*time.Microsecond)
	// Local-ish competitor: expensive compute, no network.
	local := newFakeLocal("local", 1_000_000)

	// With the prior, remote (0.5ms) beats local (1ms compute).
	store := planWith(t, local, remote)
	if winner := winnerEngine(store); winner != "remote" {
		t.Fatalf("initial winner = %q, want remote (low prior RTT)", winner)
	}

	// Shift remote's live RTT far above local's compute cost.
	tracker := metaengine.NewLatencyTracker(metaengine.WithTrackerAlpha(0.5))
	remote.SetRTTTracker(tracker)
	for range 30 {
		tracker.Record(50 * time.Millisecond)
	}

	// Re-plan: the live 50ms RTT must now lose to local's 1ms compute.
	store2 := planWith(t, local, remote)
	if winner := winnerEngine(store2); winner != "local" {
		t.Fatalf("after RTT shift winner = %q, want local", winner)
	}
}

// TestPlan_WarnsWhenRoutingOnPriorRTT proves the liveLatencyRule fires a WARN
// for a remote engine routed without a fresh live measurement.
func TestPlan_WarnsWhenRoutingOnPriorRTT(t *testing.T) {
	remote := newFakeRemote("remote", 1, 1*time.Millisecond)

	store := planWith(t, remote)

	found := false
	for _, d := range store.Plan().Diagnostics {
		if d.Level == metaengine.DiagLevelWarn && strings.Contains(d.Message, "prior") {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("expected WARN diagnostic about prior RTT; got %+v", store.Plan().Diagnostics)
	}
}

// TestPlan_NoLiveLatencyWarnForFreshTracker proves the WARN clears once the
// tracker has fresh samples (graceful degradation: warn only when not measured).
func TestPlan_NoLiveLatencyWarnForFreshTracker(t *testing.T) {
	remote := newFakeRemote("remote", 1, 1*time.Millisecond)
	tracker := metaengine.NewLatencyTracker(metaengine.WithStaleAfter(time.Minute))
	remote.SetRTTTracker(tracker)
	for range 5 {
		tracker.Record(2 * time.Millisecond)
	}

	store := planWith(t, remote)

	for _, d := range store.Plan().Diagnostics {
		if d.Level == metaengine.DiagLevelWarn && strings.Contains(d.Message, "prior") {
			t.Fatalf("unexpected prior-RTT WARN with fresh tracker: %+v", d)
		}
	}
}

// --- helpers ---

type fakeLocalEngine struct {
	metaengine.Calibration

	name    string
	nsPerOp float64
}

func newFakeLocal(name string, nsPerOp float64) *fakeLocalEngine {
	return &fakeLocalEngine{name: name, nsPerOp: nsPerOp}
}

func (e *fakeLocalEngine) Profile() metaengine.EngineProfile {
	p := metaengine.EngineProfile{
		Name:        e.name,
		NsPerOp:     e.nsPerOp,
		Persistence: metaengine.PersistenceVolatile,
		Supports: map[metaengine.ADT]metaengine.Complexity{
			metaengine.ADTMap: metaengine.ComplexityO1,
		},
	}
	e.ApplyCalibration(&p)

	return p
}

func (e *fakeLocalEngine) Close() error { return nil }

func winnerEngine(store *metaengine.Store) string {
	for _, qa := range store.Plan().Queries {
		if qa.QueryName == "find_task_live" {
			return qa.EngineName
		}
	}

	return ""
}

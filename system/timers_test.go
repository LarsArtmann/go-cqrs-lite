package system_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/scheduling/v4"
	"github.com/larsartmann/go-cqrs-lite/scheduling/engine/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// TestSystem_DeclarableTimers: the composition root owns the timer story
// (ADR-0142 T11) — TimerEngine picks the deployment-declared engine
// ("timers", falling back to the primary), an engine-backed TimerStore
// rides it, and ManageTimers gives the System the Scheduler lifecycle.
func TestSystem_DeclarableTimers(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	sys, err := system.New(ctx, system.DomainConfig{}, system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			"primary": {Driver: "memory"},
			"timers":  {Driver: "memory"},
		},
		Instances: []system.InstanceConfig{{Role: system.RoleSourceOfTruth, Engine: "primary"}},
	})
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}

	defer func() { _ = sys.Close() }()

	timerEng := sys.TimerEngine()
	if timerEng == nil {
		t.Fatal("TimerEngine must resolve the 'timers' engine")
	}

	if timerEng.Profile().Name != "memory" {
		t.Fatalf("timer engine = %s, want memory", timerEng.Profile().Name)
	}

	if !metaengine.SupportsDueClaims(timerEng) {
		t.Fatal("the timer engine must carry the DueClaimer capability")
	}

	store, err := engine.NewTimerStore[string](timerEng)
	if err != nil {
		t.Fatalf("NewTimerStore: %v", err)
	}

	mustScheduleTimer(t, store, "system-timer", time.Now().Add(-time.Second))

	var dispatched atomic.Int64

	sched := scheduling.New(
		store,
		func(_ context.Context, _ scheduling.Timer[string]) error {
			dispatched.Add(1)

			return nil
		},
		scheduling.WithPollInterval(10*time.Millisecond),
	)

	sys.ManageTimers(sched)

	if err := sys.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	waitForTimers(t, 3*time.Second, func() bool { return dispatched.Load() >= 1 })

	if err := sys.GracefulClose(context.Background()); err != nil {
		t.Fatalf("GracefulClose: %v", err)
	}

	// After graceful close the scheduler must be stopped: no further
	// dispatch growth beyond a small in-flight tail.
	settled := dispatched.Load()
	time.Sleep(80 * time.Millisecond)

	if grown := dispatched.Load() - settled; grown > 1 {
		t.Fatalf("scheduler still dispatching after GracefulClose (+%d)", grown)
	}
}

func TestSystem_TimerEngineFallsBackToPrimary(t *testing.T) {
	t.Parallel()

	sys, err := system.New(context.Background(), system.DomainConfig{}, system.DeploymentConfig{
		Engines:   map[string]system.EngineConfig{"primary": {Driver: "memory"}},
		Instances: []system.InstanceConfig{{Role: system.RoleSourceOfTruth, Engine: "primary"}},
	})
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}

	defer func() { _ = sys.Close() }()

	if sys.TimerEngine() == nil {
		t.Fatal("TimerEngine must fall back to the primary engine")
	}
}

func mustScheduleTimer(t *testing.T, s *engine.TimerStore[string], id string, fireAt time.Time) {
	t.Helper()

	if err := s.Schedule(t.Context(), scheduling.Timer[string]{
		ID:      scheduling.MustParseTimerID(id),
		FireAt:  fireAt,
		Payload: "go",
	}); err != nil {
		t.Fatalf("Schedule: %v", err)
	}
}

func waitForTimers(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}

		time.Sleep(5 * time.Millisecond)
	}
}

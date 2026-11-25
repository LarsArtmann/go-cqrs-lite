package system_test

import (
	"context"
	"errors"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

func newMemorySystem(t *testing.T) *system.System {
	t.Helper()

	sys, err := system.New(context.Background(), system.DomainConfig{}, system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{"primary": {Driver: "memory"}},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
		},
	})
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}

	return sys
}

func newProjectionSystem(t *testing.T) *system.System {
	t.Helper()

	sys, err := system.New(context.Background(), system.DomainConfig{
		Projections: []system.ProjectionDeclaration{
			system.RawQuery(taskProjectionQuery("ready_check")),
		},
	}, memoryProjectionDeployment())
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}

	return sys
}

// A system that owns no background work is ready the moment it is constructed.
func TestSystem_ReadyCheck_NoBackgroundWorkIsReadyAtConstruction(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	sys := newMemorySystem(t)
	defer func() { _ = sys.Close() }()

	if err := sys.ReadyCheck(ctx); err != nil {
		t.Fatalf("ReadyCheck on a constructed system without background work: %v", err)
	}

	if got := sys.Health(ctx); got != "ok" {
		t.Fatalf("Health = %q, want %q", got, "ok")
	}
}

// HealthCheck is a liveness probe and passes before Start; ReadyCheck is the
// readiness probe and does not (issue #139).
func TestSystem_ReadyCheck_ProjectionsNotStarted(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	sys := newProjectionSystem(t)

	if err := sys.HealthCheck(ctx); err != nil {
		t.Fatalf("HealthCheck before Start (liveness): %v", err)
	}

	if err := sys.ReadyCheck(ctx); !errors.Is(err, system.ErrSystemNotStarted) {
		t.Fatalf("ReadyCheck before Start = %v, want ErrSystemNotStarted", err)
	}

	if got, want := sys.Health(ctx), "ok projections:not-started"; got != want {
		t.Fatalf("Health before Start = %q, want %q", got, want)
	}

	if err := sys.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	if err := sys.ReadyCheck(ctx); err != nil {
		t.Fatalf("ReadyCheck after Start: %v", err)
	}

	if got, want := sys.Health(ctx), "ok projections:running"; got != want {
		t.Fatalf("Health after Start = %q, want %q", got, want)
	}

	if err := sys.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if err := sys.ReadyCheck(ctx); !errors.Is(err, system.ErrSystemStopped) {
		t.Fatalf("ReadyCheck after Close = %v, want ErrSystemStopped", err)
	}
}

// Managed schedulers run only as long as the context passed to Start: once a
// caller cancels it, timers are dead and readiness must say so.
func TestSystem_ReadyCheck_TimersStopWhenStartContextIsCancelled(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	sys := newMemorySystem(t)

	sched := &blockingScheduler{returned: make(chan struct{})}
	sys.ManageTimers(sched)

	if err := sys.ReadyCheck(ctx); !errors.Is(err, system.ErrSystemNotStarted) {
		t.Fatalf("ReadyCheck before Start = %v, want ErrSystemNotStarted", err)
	}

	if got, want := sys.Health(ctx), "ok timers:not-started"; got != want {
		t.Fatalf("Health before Start = %q, want %q", got, want)
	}

	startCtx, cancelStart := context.WithCancel(ctx)
	if err := sys.Start(startCtx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	if err := sys.ReadyCheck(ctx); err != nil {
		t.Fatalf("ReadyCheck after Start: %v", err)
	}

	if got, want := sys.Health(ctx), "ok timers:running"; got != want {
		t.Fatalf("Health after Start = %q, want %q", got, want)
	}

	cancelStart()

	select {
	case <-sched.returned:
	case <-time.After(5 * time.Second):
		t.Fatal("managed scheduler did not return after the Start context was cancelled")
	}

	deadline := time.Now().Add(5 * time.Second)
	for {
		err := sys.ReadyCheck(ctx)
		if errors.Is(err, system.ErrTimersStopped) {
			break
		}

		if time.Now().After(deadline) {
			t.Fatalf("ReadyCheck after the Start context was cancelled = %v, want ErrTimersStopped", err)
		}

		runtime.Gosched()
	}

	if got := sys.Health(ctx); !strings.Contains(got, "timers:stopped") {
		t.Fatalf("Health after the Start context was cancelled = %q, want it to contain %q", got, "timers:stopped")
	}

	if err := sys.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if err := sys.ReadyCheck(ctx); !errors.Is(err, system.ErrSystemStopped) {
		t.Fatalf("ReadyCheck after Close = %v, want ErrSystemStopped", err)
	}
}

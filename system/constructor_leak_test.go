package system_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// leakProbeEngine wraps a memory engine with a background heartbeat goroutine
// that only exits on Close. If New's error path forgets the engine, the
// goroutine outlives the test and goleak.VerifyTestMain (main_test.go) fails
// the whole suite; the closed channel additionally makes the miss readable
// per-test.
type leakProbeEngine struct {
	metaengine.Engine
	metaengine.StreamLogBackend
	metaengine.AtomicAppender
	stop   chan struct{}
	closed chan struct{}
	once   sync.Once
}

func newLeakProbeEngine() *leakProbeEngine {
	base := metaengine.NewMemoryEngine()

	streamLog, ok := base.(metaengine.StreamLogBackend)
	if !ok {
		panic("memory engine must implement StreamLogBackend")
	}

	atomic, ok := base.(metaengine.AtomicAppender)
	if !ok {
		panic("memory engine must implement AtomicAppender")
	}

	e := &leakProbeEngine{
		Engine:           base,
		StreamLogBackend: streamLog,
		AtomicAppender:   atomic,
		stop:             make(chan struct{}),
		closed:           make(chan struct{}),
	}

	go func() {
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
			case <-e.stop:
				return
			}
		}
	}()

	return e
}

func (e *leakProbeEngine) Close() error {
	e.once.Do(func() {
		close(e.stop)
		close(e.closed)
	})

	return e.Engine.Close()
}

func waitForClosed(t *testing.T, name string, closed <-chan struct{}) {
	t.Helper()

	select {
	case <-closed:
	case <-time.After(5 * time.Second):
		t.Fatalf("New error path did not close engine %q", name)
	}
}

// TestSystem_NewErrorPath_ClosesCreatedEngines pins M07: every error return
// after the first engine exists must tear down what New created. The
// projections instance references an undefined engine, so the failure happens
// well after both engines were constructed — exactly the window that used to
// leak file handles, locks, and goroutines.
func TestSystem_NewErrorPath_ClosesCreatedEngines(t *testing.T) {
	t.Parallel()

	probeA := newLeakProbeEngine()
	probeB := newLeakProbeEngine()

	metaengine.RegisterDriver("leakprobe-a-test", func(
		_ context.Context, _ metaengine.DriverConfig,
	) (metaengine.Engine, error) {
		return probeA, nil
	})

	metaengine.RegisterDriver("leakprobe-b-test", func(
		_ context.Context, _ metaengine.DriverConfig,
	) (metaengine.Engine, error) {
		return probeB, nil
	})

	_, err := system.New(
		context.Background(),
		system.DomainConfig{},
		system.DeploymentConfig{
			Engines: map[string]system.EngineConfig{
				"probe-a": {Driver: "leakprobe-a-test"},
				"probe-b": {Driver: "leakprobe-b-test"},
			},
			Instances: []system.InstanceConfig{
				{Role: system.RoleProjections, Engines: []string{"does-not-exist"}},
			},
		},
	)

	if !errors.Is(err, system.ErrUnknownEngine) {
		t.Fatalf("New error = %v, want ErrUnknownEngine", err)
	}

	waitForClosed(t, "probe-a", probeA.closed)
	waitForClosed(t, "probe-b", probeB.closed)
}

// TestSystem_NewErrorPath_ClosesEventBus pins the event-bus half of the M07
// teardown: the bus is registered for lifecycle management BEFORE the
// publisher build, so an undeclared publish target (which fails after the bus
// exists) cannot strand it. The engine-close probe proves fail() ran to
// completion (fail closes engines, then closers — the bus included); a
// stranded bus's own goroutines are additionally caught suite-wide by
// goleak.VerifyTestMain in main_test.go.
func TestSystem_NewErrorPath_ClosesEventBus(t *testing.T) {
	t.Parallel()

	closed := make(chan struct{})

	var once sync.Once

	probe := &engineCloseHookEngine{
		leakProbeEngine: newLeakProbeEngine(),
		onClose: func() {
			once.Do(func() { close(closed) })
		},
	}

	metaengine.RegisterDriver("leakprobe-bus-test", func(
		_ context.Context, _ metaengine.DriverConfig,
	) (metaengine.Engine, error) {
		return probe, nil
	})

	_, err := system.New(
		context.Background(),
		system.DomainConfig{},
		system.DeploymentConfig{
			Engines: map[string]system.EngineConfig{
				"probe": {Driver: "leakprobe-bus-test"},
			},
			Instances: []system.InstanceConfig{
				{
					Role:    system.RoleSourceOfTruth,
					Engines: []string{"probe"},
					Publish: []string{"undeclared-bus"},
				},
			},
		},
	)

	if !errors.Is(err, system.ErrUnknownPublishTarget) {
		t.Fatalf("New error = %v, want ErrUnknownPublishTarget", err)
	}

	waitForClosed(t, "probe (bus teardown runs after engine close)", closed)
}

// engineCloseHookEngine layers a Close hook over the leak probe without
// duplicating its backend wiring.
type engineCloseHookEngine struct {
	*leakProbeEngine
	onClose func()
}

func (e *engineCloseHookEngine) Close() error {
	err := e.leakProbeEngine.Close()
	e.onClose()

	return err //nolint:wrapcheck // passthrough by design
}

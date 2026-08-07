package system_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

func TestSystem_HealthCheck_Healthy(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	sys, err := system.New(ctx, system.DomainConfig{}, system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{"primary": {Driver: "memory"}},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
		},
	})
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}
	defer sys.Close()

	if err := sys.HealthCheck(ctx); err != nil {
		t.Fatalf("HealthCheck on healthy system: %v", err)
	}
}

func TestSystem_HealthCheck_Stopped(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	sys, _ := system.New(ctx, system.DomainConfig{}, system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{"primary": {Driver: "memory"}},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
		},
	})

	if err := sys.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	err := sys.HealthCheck(ctx)
	if err == nil {
		t.Fatal("expected error from HealthCheck on stopped system")
	}

	if !errors.Is(err, system.ErrSystemStopped) {
		t.Fatalf("expected ErrSystemStopped, got: %v", err)
	}
}

func TestSystem_GracefulClose(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	sys, err := system.New(ctx, system.DomainConfig{}, system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{"primary": {Driver: "memory"}},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
		},
	})
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}

	gCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := sys.GracefulClose(gCtx); err != nil {
		t.Fatalf("GracefulClose: %v", err)
	}

	// Double GracefulClose should be nil (idempotent via Close).
	if err := sys.GracefulClose(gCtx); err != nil {
		t.Fatalf("double GracefulClose: %v", err)
	}
}

func TestSystem_GracefulClose_ContextExpired(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	sys, _ := system.New(ctx, system.DomainConfig{}, system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{"primary": {Driver: "memory"}},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
		},
	})

	// Use an already-cancelled context to trigger the timeout path.
	cancelCtx, cancel := context.WithCancel(ctx)
	cancel()

	err := sys.GracefulClose(cancelCtx)
	if err == nil {
		t.Fatal("expected error from GracefulClose with cancelled context")
	}
}

func TestSystem_ResetProjection_NoHost(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// No projection instance configured → no projHost.
	sys, _ := system.New(ctx, system.DomainConfig{}, system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{"primary": {Driver: "memory"}},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
		},
	})
	defer sys.Close()

	err := sys.ResetProjection(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error from ResetProjection with no projection host")
	}

	if !errors.Is(err, system.ErrNoProjectionHost) {
		t.Fatalf("expected ErrNoProjectionHost, got: %v", err)
	}
}

// recordingCheckpointStore is a minimal CheckpointStore that records calls
// for test assertions.
type recordingCheckpointStore struct {
	saved   map[string]event.Checkpoint
	saveCnt int
}

func (s *recordingCheckpointStore) Save(
	_ context.Context,
	projection string,
	cp event.Checkpoint,
) error {
	if s.saved == nil {
		s.saved = make(map[string]event.Checkpoint)
	}

	s.saved[projection] = cp
	s.saveCnt++

	return nil
}

func (s *recordingCheckpointStore) Load(
	_ context.Context,
	projection string,
) (event.Checkpoint, error) {
	return s.saved[projection], nil
}

func (s *recordingCheckpointStore) Close() error { return nil }

// taskProjectionQuery returns a metaengine query declaration for a task view
// projection. Used by multiple hardening tests.
func taskProjectionQuery(collection string) any {
	return metaengine.Query[FindTask, TaskView](collection,
		metaengine.OnTyped("task.created", TaskCreated{}, func(e TaskCreated) (string, TaskView) {
			return e.Title, TaskView{Title: e.Title, Status: "pending"}
		}),
	)
}

// taskDomainConfig returns a DomainConfig with a task.create command and
// the given projection + checkpoint store.
func taskDomainConfig(
	projection any,
	cpStore event.CheckpointStore,
	extraOpts ...projectionhost.HostOption,
) system.DomainConfig {
	opts := []projectionhost.HostOption{
		projectionhost.WithBackoff(10*time.Millisecond, 100*time.Millisecond),
	}
	opts = append(opts, extraOpts...)

	return system.DomainConfig{
		Commands: func(sys *system.System) {
			system.RegisterDecider(sys, "Task", TaskDecider)

			system.RegisterCommand[*command.BasicCommand, TaskState](sys, "task.create",
				func(ctx context.Context, cmd *command.BasicCommand) system.Op[TaskState] {
					return system.Execute(ctx, cmd.StreamID(), "Task",
						func(state TaskState, ver event.Version) ([]event.Event, error) {
							if state.Exists {
								return nil, errors.New("task already exists")
							}

							return []event.Event{mustEvent(event.New("task.created",
								cmd.StreamID(), "Task", ver+1,
								TaskCreated{Title: "hardening-task", At: time.Now()},
								event.WithCodec(codec.JSONCodec{})))}, nil
						})
				})
		},
		Projections:           []any{projection},
		ProjectionDecoder:     projectionDecoder,
		ProjectionHostOptions: opts,
		CheckpointStore:       cpStore,
	}
}

func memoryProjectionDeployment() system.DeploymentConfig {
	return system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{"primary": {Driver: "memory"}},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
			{Role: system.RoleProjections, Engine: "primary"},
		},
	}
}

// waitForProjectionProcessed polls the projection host until at least one
// worker has processed >= minProcessed events with zero errors, or the
// deadline expires.
func waitForProjectionProcessed(t *testing.T, sys *system.System, minProcessed int) bool {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)

	for time.Now().Before(deadline) {
		for _, s := range sys.ProjectionHost().Status() {
			if s.Processed >= int64(minProcessed) && s.Errors == 0 {
				return true
			}
		}

		time.Sleep(50 * time.Millisecond)
	}

	return false
}

func TestSystem_CustomCheckpointStore(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cpStore := &recordingCheckpointStore{}

	sys, err := system.New(ctx, taskDomainConfig(taskProjectionQuery("cp_test"), cpStore),
		memoryProjectionDeployment())
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}
	defer sys.Close()

	// Produce an event before starting projections — host will replay from journal.
	if err := sys.CommandDispatcher().
		Dispatch(ctx, newCmd("task.create", id.NewStreamID())); err != nil {
		t.Fatalf("dispatch create: %v", err)
	}

	if err := sys.Start(ctx); err != nil {
		t.Fatalf("system.Start: %v", err)
	}

	if !waitForProjectionProcessed(t, sys, 1) {
		for _, s := range sys.ProjectionHost().Status() {
			t.Fatalf("projection %q: processed=%d errors=%d", s.Name, s.Processed, s.Errors)
		}
	}

	// The custom checkpoint store must have been used.
	if cpStore.saveCnt == 0 {
		t.Fatal("expected recordingCheckpointStore.saveCnt > 0, got 0 — custom store was not used")
	}
}

func TestSystem_HealthCheck_FailedProjection(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cpStore := &recordingCheckpointStore{}

	// A decoder that always errors, causing the projection handler to fail.
	failingDecoder := func(_ string, _ []byte) (any, error) {
		return nil, errors.New("intentional decoder failure")
	}

	domain := system.DomainConfig{
		Commands: func(sys *system.System) {
			system.RegisterDecider(sys, "Task", TaskDecider)

			system.RegisterCommand[*command.BasicCommand, TaskState](sys, "task.create",
				func(ctx context.Context, cmd *command.BasicCommand) system.Op[TaskState] {
					return system.Execute(ctx, cmd.StreamID(), "Task",
						func(state TaskState, ver event.Version) ([]event.Event, error) {
							return []event.Event{mustEvent(event.New("task.created",
								cmd.StreamID(), "Task", ver+1,
								TaskCreated{Title: "fail-proj", At: time.Now()},
								event.WithCodec(codec.JSONCodec{})))}, nil
						})
				})
		},
		Projections:       []any{taskProjectionQuery("fail_proj")},
		ProjectionDecoder: failingDecoder,
		ProjectionHostOptions: []projectionhost.HostOption{
			projectionhost.WithMaxRestarts(1),
			projectionhost.WithBackoff(1*time.Millisecond, 5*time.Millisecond),
		},
		CheckpointStore: cpStore,
	}

	sys, err := system.New(ctx, domain, memoryProjectionDeployment())
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}
	defer sys.Close()

	// Produce an event that will cause the projection to fail.
	if err := sys.CommandDispatcher().
		Dispatch(ctx, newCmd("task.create", id.NewStreamID())); err != nil {
		t.Fatalf("dispatch create: %v", err)
	}

	if err := sys.Start(ctx); err != nil {
		t.Fatalf("system.Start: %v", err)
	}

	// Wait for the worker to reach WorkerFailed state.
	deadline := time.Now().Add(10 * time.Second)

	var failed bool

	for time.Now().Before(deadline) {
		for _, s := range sys.ProjectionHost().Status() {
			if s.Status == projectionhost.WorkerFailed {
				failed = true
				break
			}
		}

		if failed {
			break
		}

		time.Sleep(50 * time.Millisecond)
	}

	if !failed {
		for _, s := range sys.ProjectionHost().Status() {
			t.Logf("worker %q: status=%s processed=%d errors=%d lastError=%s",
				s.Name, s.Status, s.Processed, s.Errors, s.LastError)
		}

		t.Fatal("projection worker did not reach WorkerFailed state")
	}

	// HealthCheck should return an error naming the failed projection.
	err = sys.HealthCheck(ctx)
	if err == nil {
		t.Fatal("expected HealthCheck to return error for failed projection")
	}
}

func TestSystem_ResetProjection_Positive(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cpStore := &recordingCheckpointStore{}

	sys, err := system.New(ctx, taskDomainConfig(taskProjectionQuery("reset_test"), cpStore),
		memoryProjectionDeployment())
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}
	defer sys.Close()

	// Produce an event and let the projection process it.
	if err := sys.CommandDispatcher().
		Dispatch(ctx, newCmd("task.create", id.NewStreamID())); err != nil {
		t.Fatalf("dispatch create: %v", err)
	}

	if err := sys.Start(ctx); err != nil {
		t.Fatalf("system.Start: %v", err)
	}

	if !waitForProjectionProcessed(t, sys, 1) {
		for _, s := range sys.ProjectionHost().Status() {
			t.Fatalf("projection %q: processed=%d errors=%d", s.Name, s.Processed, s.Errors)
		}
	}

	// Record the checkpoint count before reset.
	saveCntBeforeReset := cpStore.saveCnt
	if saveCntBeforeReset == 0 {
		t.Fatal("expected checkpoint saves before reset, got 0")
	}

	// Stop the projection host before resetting (Reset requires host to be stopped).
	if err := sys.ProjectionHost().Stop(); err != nil {
		t.Fatalf("stop projection host: %v", err)
	}

	// Reset the projection — should clear the checkpoint.
	if err := sys.ResetProjection(ctx, "projections"); err != nil {
		t.Fatalf("ResetProjection: %v", err)
	}

	// The reset should have saved a zero-value checkpoint, incrementing saveCnt.
	if cpStore.saveCnt <= saveCntBeforeReset {
		t.Fatalf("expected saveCnt > %d after reset, got %d", saveCntBeforeReset, cpStore.saveCnt)
	}

	// The last saved checkpoint for "projections" should be the zero value (cleared).
	lastCp := cpStore.saved["projections"]
	if !lastCp.IsZero() {
		t.Fatalf("expected zero-value checkpoint after reset, got %v", lastCp)
	}
}

func TestSystem_GracefulClose_SlowShutdown(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	sys, err := system.New(ctx, taskDomainConfig(taskProjectionQuery("slow_shutdown"), nil),
		memoryProjectionDeployment())
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}

	// Produce events and start processing so Close has real work to drain.
	for range 3 {
		if err := sys.CommandDispatcher().
			Dispatch(ctx, newCmd("task.create", id.NewStreamID())); err != nil {
			t.Fatalf("dispatch create: %v", err)
		}
	}

	if err := sys.Start(ctx); err != nil {
		t.Fatalf("system.Start: %v", err)
	}

	// Wait for at least one event to be processed.
	if !waitForProjectionProcessed(t, sys, 1) {
		for _, s := range sys.ProjectionHost().Status() {
			t.Fatalf("projection %q: processed=%d errors=%d", s.Name, s.Processed, s.Errors)
		}
	}

	// GracefulClose should complete within the context (generous timeout).
	gCtx, gCancel := context.WithTimeout(ctx, 10*time.Second)
	defer gCancel()

	if err := sys.GracefulClose(gCtx); err != nil {
		t.Fatalf("GracefulClose with slow shutdown: %v", err)
	}
}

func TestSystem_HealthCheck_SQLite(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	sys, err := system.New(ctx, system.DomainConfig{}, system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			"primary": {
				Driver:  "sqlite",
				DSN:     "file:" + t.Name() + "?mode=memory&cache=shared",
				Pragmas: []string{"journal_mode=wal"},
			},
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
		},
	})
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}
	defer sys.Close()

	// SQLite engine implements metaengine.HealthChecker via db.PingContext.
	// HealthCheck should succeed when the DB is reachable.
	if err := sys.HealthCheck(ctx); err != nil {
		t.Fatalf("HealthCheck on SQLite system: %v", err)
	}
}

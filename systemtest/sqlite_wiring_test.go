package systemtest_test

import (
	"context"
	"errors"
	"runtime"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-codec"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// Real-sqlite-driver wiring tests moved from system/ in the Feedback-#4
// split: they require the sqlite driver registration that only the
// systemtest module provides.

// TestSystem_RoleWiring_DedicatedSnapshots verifies a dedicated snapshots
// instance binds the snapshot store from its own engine (SQLite implements
// SnapshotBackend; the memory source-of-truth does not).
func TestSystem_RoleWiring_DedicatedSnapshots(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	sys, err := system.New(ctx, system.DomainConfig{}, system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			"primary": {Driver: "memory"},
			"snaps":   {Driver: "sqlite"},
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
			{Role: system.RoleSnapshots, Engine: "snaps"},
		},
	})
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}

	defer sys.Close()

	snapStore := sys.SnapshotStore()
	if snapStore == nil {
		t.Fatal("dedicated snapshots instance must bind SnapshotStore")
	}

	streamID := id.NewStreamID()

	snap := snapshot.Snapshot{
		StreamID:   streamID,
		StreamType: "Task",
		Version:    event.Version(3),
		State:      []byte("state"),
		CreatedAt:  time.Now(),
	}

	if err := snapStore.Save(ctx, snap); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := snapStore.Load(ctx, id.NewStreamRef("Task", streamID))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded == nil || string(loaded.State) != "state" || loaded.Version != event.Version(3) {
		t.Fatalf("snapshot = %+v, want (state, version 3)", loaded)
	}
}

func TestSystem_RegisteredDriversIncludesMemoryAndSQLite(t *testing.T) {
	t.Parallel()

	drivers := metaengine.RegisteredDrivers()

	if !slices.Contains(drivers, "memory") {
		t.Fatal("memory driver not registered")
	}

	if !slices.Contains(drivers, "sqlite") {
		t.Fatal("sqlite driver not registered")
	}
}

func TestSystem_HealthCheck_SQLite(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	sys, err := system.New(ctx, system.DomainConfig{}, system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			"primary": {
				Driver:  "sqlite",
				DSN:     sqliteTestDSN(t),
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

// TestSystem_ResetProjection_RestartAndReplay verifies that a projection can be
// reset and replayed from scratch after a restart. It uses SQLite's
// shared-cache in-memory DSN pattern (file:<name>?mode=memory&cache=shared)
// so that two System instances (phase 1: produce, phase 2: replay) share the
// same in-memory database. The shared cache ensures data written by sys1 is
// visible to sys2 without touching disk.
func TestSystem_ResetProjection_RestartAndReplay(t *testing.T) {
	// NOT parallel: this test has two sequential projection-wait phases and
	// a system close/reopen cycle. Running it concurrently with
	// TestSystem_HealthCheck_FailedProjection (which has tight retry-loop
	// timing) causes CPU contention that can make the projection-wait
	// budget expire spuriously on busy machines.

	// Load-scaled outer deadline with real headroom: TWO sequential
	// waitForProjectionProcessed budgets (loadScaledDeadline(15s) each) plus
	// a close/reopen cycle with snapshot load must all fit inside it. The
	// 2026-09-08 fix (30s) matched one inner budget — but 15s+15s already
	// consumes the whole thing at factor 1, leaving phase 2's replay-from-
	// zero to starve under full-suite contention.
	// Load-scaled outer deadline: must exceed BOTH waitForProjectionProcessed
	// budgets (phase 1 + phase 2, 45s base each) — the 2026-09-08 fix already
	// starved once when the outer ctx (30s) matched a single inner budget.
	// 270s = 3x the combined inner budget at factor 1.
	ctx, cancel := context.WithDeadline(context.Background(), loadScaledDeadline(270*time.Second))
	defer cancel()

	cpStore := &recordingCheckpointStore{}
	dsn := sqliteFileDSN(t)

	deployment := system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			"primary": {Driver: "sqlite", DSN: dsn, Pragmas: []string{"journal_mode=wal"}},
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
			{Role: system.RoleProjections, Engine: "primary"},
		},
	}

	// Phase 1: produce an event, process it, then stop and reset.
	sys1, err := system.New(ctx, taskDomainConfig(taskProjectionQuery("replay_test"), cpStore),
		deployment)
	if err != nil {
		t.Fatalf("system.New (phase 1): %v", err)
	}

	if err := sys1.CommandDispatcher().
		Dispatch(ctx, newCmd("task.create", id.NewStreamID())); err != nil {
		t.Fatalf("dispatch create: %v", err)
	}

	if err := sys1.Start(ctx); err != nil {
		t.Fatalf("system.Start (phase 1): %v", err)
	}

	if !waitForProjectionProcessed(t, sys1, 1) {
		for _, s := range sys1.ProjectionHost().Status() {
			t.Fatalf("projection %q: processed=%d errors=%d", s.Name, s.Processed, s.Errors)
		}
	}

	// Verify the projection has data before reset.
	view, err := metaengine.ExecuteTyped[FindTask, TaskView](
		ctx, sys1.MetaEngine(), FindTask{ID: "hardening-task"})
	if err != nil {
		t.Fatalf("ExecuteTyped before reset: %v", err)
	}

	if view.Title != "hardening-task" {
		t.Fatalf("expected task %q before reset, got %q", "hardening-task", view.Title)
	}

	// Stop, reset, close. The SQLite DB persists in-memory via shared cache.
	if err := sys1.ProjectionHost().Stop(); err != nil {
		t.Fatalf("stop projection host: %v", err)
	}

	if err := sys1.ResetProjection(ctx, "projections"); err != nil {
		t.Fatalf("ResetProjection: %v", err)
	}

	// Checkpoint should be zero-value after reset.
	lastCp := cpStore.checkpoint("projections")
	if !lastCp.IsZero() {
		t.Fatalf("expected zero-value checkpoint after reset, got %v", lastCp)
	}

	if err := sys1.Close(); err != nil {
		t.Fatalf("Close (phase 1): %v", err)
	}

	// Phase 2: new system with same SQLite DSN (events persist) and fresh
	// checkpoint store (no checkpoint = replay from zero).
	sys2, err := system.New(ctx, taskDomainConfig(taskProjectionQuery("replay_test"), nil),
		deployment)
	if err != nil {
		t.Fatalf("system.New (phase 2): %v", err)
	}
	defer sys2.Close()

	if err := sys2.Start(ctx); err != nil {
		t.Fatalf("system.Start (phase 2): %v", err)
	}

	// Wait for the projection to replay from the journal.
	if !waitForProjectionProcessed(t, sys2, 1) {
		if js, ok := sys2.EventStore().(event.SeekableJournal); ok {
			journal, readErr := js.ReadFrom(ctx, id.EventID{}, 10)
			if readErr != nil {
				t.Logf("diagnostic: journal read error from sys2: %v", readErr)
			} else {
				t.Logf("diagnostic: journal holds %d event(s) from sys2's view", len(journal))
			}
		}

		for _, s := range sys2.ProjectionHost().Status() {
			t.Fatalf(
				"projection %q after replay: status=%s processed=%d errors=%d restarts=%d checkpoint=%q lastError=%q",
				s.Name,
				s.Status,
				s.Processed,
				s.Errors,
				s.Restarts,
				s.Checkpoint,
				s.LastError,
			)
		}
	}

	// Verify the replayed projection has the same data.
	view2, err := metaengine.ExecuteTyped[FindTask, TaskView](
		ctx, sys2.MetaEngine(), FindTask{ID: "hardening-task"})
	if err != nil {
		t.Fatalf("ExecuteTyped after replay: %v", err)
	}

	if view2.Title != "hardening-task" {
		t.Fatalf("expected replayed projection to have task %q, got %q",
			"hardening-task", view2.Title)
	}
}

// recordingCheckpointStore twin of system/system_hardening_test.go's
// helper — test fixtures may not cross the module boundary.

//art-dupl:accept test-fixture twin of system/system_hardening_test.go helper

type recordingCheckpointStore struct {
	mu      sync.Mutex
	saved   map[string]event.Checkpoint
	saveCnt int
}

func (s *recordingCheckpointStore) Save(
	_ context.Context,
	projection string,
	cp event.Checkpoint,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

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
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.saved[projection], nil
}

func (s *recordingCheckpointStore) Close() error { return nil }

func (s *recordingCheckpointStore) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.saveCnt
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
		Projections:           []system.ProjectionDeclaration{system.RawQuery(projection)},
		ProjectionDecoder:     projectionDecoder,
		ProjectionHostOptions: opts,
		CheckpointStore:       cpStore,
	}
}

// taskProjectionQuery twin of system/system_hardening_test.go helper.

//art-dupl:accept test-fixture twin of system/system_hardening_test.go helper

// taskProjectionQuery returns a metaengine query declaration for a task view
// projection. Used by multiple hardening tests.
func taskProjectionQuery(collection string) any {
	return metaengine.Query[FindTask, TaskView](
		collection,
		metaengine.OnRecordTyped(
			"task.created",
			TaskCreated{},
			func(_ record.Record, e TaskCreated) (string, TaskView) {
				return e.Title, TaskView{Title: e.Title, Status: "pending"}
			},
		),
	)
}

// waitForProjectionProcessed twin of system/system_hardening_test.go helper.

//art-dupl:accept test-fixture twin of system/system_hardening_test.go helper

// waitForProjectionProcessed polls the projection host until at least one
// worker has processed >= minProcessed events with zero errors, or the
// deadline expires.
//
// Base is 45s (raised from 15s, 2026-09-13): under full-suite contention the
// replay-from-zero phase 2 of TestSystem_ResetProjection_RestartAndReplay
// starved past 15s — load1/cores underestimates disk + cross-process
// contention on high-core machines (factor floors at 1), so the raw budget
// carries the headroom.
func waitForProjectionProcessed(t *testing.T, sys *system.System, minProcessed int) bool {
	t.Helper()

	deadline := loadScaledDeadline(45 * time.Second)

	for time.Now().Before(deadline) {
		for _, s := range sys.ProjectionHost().Status() {
			if s.Processed >= int64(minProcessed) && s.Errors == 0 {
				return true
			}
		}

		time.Sleep(50 * time.Millisecond)
	}

	// Starvation crime scene: every composed-run failure so far reported
	// only processed=0 errors=0, never the worker's goroutine state — the
	// root cause (blocked vs exited-empty vs never-started) was
	// undiscoverable after the fact. Dump all stacks while the failure is
	// live so the next composed-run incident pins the blocking site.
	buf := make([]byte, 1<<20)

	n := runtime.Stack(buf, true)

	t.Logf("projection wait expired (deadline %s, load factor %.2f); goroutine dump follows",
		deadline.Format(time.RFC3339), currentLoadFactor())
	t.Logf("%s", buf[:n])

	return false
}

// checkpoint returns the last saved checkpoint for a projection (race-safe).
func (s *recordingCheckpointStore) checkpoint(projection string) event.Checkpoint {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.saved[projection]
}

package system_test

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// ── SQLite-through-System integration tests ──

func sqliteDeployment(t *testing.T) system.DeploymentConfig {
	t.Helper()

	return system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			"primary": {
				Driver:  "sqlite",
				DSN:     fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()),
				Pragmas: []string{"journal_mode=wal"},
			},
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
		},
	}
}

func TestSystem_SQLiteFullCQRSRoundtrip(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	domain := system.DomainConfig{
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
								TaskCreated{Title: "sqlite task", At: time.Now()}))}, nil
						})
				})

			system.RegisterCommand[*command.BasicCommand, TaskState](sys, "task.complete",
				func(ctx context.Context, cmd *command.BasicCommand) system.Op[TaskState] {
					return system.Execute(ctx, cmd.StreamID(), "Task",
						func(state TaskState, ver event.Version) ([]event.Event, error) {
							if !state.Exists {
								return nil, errors.New("task not found")
							}

							return []event.Event{mustEvent(event.New("task.completed",
								cmd.StreamID(), "Task", ver+1,
								TaskCompleted{At: time.Now()}))}, nil
						})
				})
		},
	}

	sys, err := system.New(ctx, domain, sqliteDeployment(t))
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}
	defer sys.Close()

	// Create a task.
	taskStreamID := id.NewStreamID()
	createCmd := newCmd("task.create", taskStreamID)
	if err := sys.CommandDispatcher().Dispatch(ctx, createCmd); err != nil {
		t.Fatalf("dispatch create: %v", err)
	}

	// Verify event persisted via Load.
	ref := id.NewStreamRef("Task", taskStreamID)
	events, err := sys.EventStore().Load(ctx, ref)
	if err != nil {
		t.Fatalf("load events: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	if events[0].Type() != "task.created" {
		t.Fatalf("expected task.created, got %s", events[0].Type())
	}

	var p TaskCreated
	if err := json.Unmarshal(events[0].Payload(), &p); err != nil {
		// Payload may be CBOR-encoded (default codec) — try auto-decode.
		decoded, derr := event.DecodePayloadAuto[TaskCreated](events[0])
		if derr != nil {
			t.Fatalf("decode payload: %v", derr)
		}
		p = decoded
	}
	if p.Title != "sqlite task" {
		t.Fatalf("expected title %q, got %q", "sqlite task", p.Title)
	}

	// Complete the task.
	completeCmd := newCmd("task.complete", taskStreamID)
	if err := sys.CommandDispatcher().Dispatch(ctx, completeCmd); err != nil {
		t.Fatalf("dispatch complete: %v", err)
	}

	events, _ = sys.EventStore().Load(ctx, ref)
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
}

func TestSystem_SQLiteOptimisticConcurrency(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	domain := system.DomainConfig{
		Commands: func(sys *system.System) {
			system.RegisterDecider(sys, "Task", TaskDecider)

			system.RegisterCommand[*command.BasicCommand, TaskState](sys, "task.create",
				func(ctx context.Context, cmd *command.BasicCommand) system.Op[TaskState] {
					return system.Execute(ctx, cmd.StreamID(), "Task",
						func(state TaskState, ver event.Version) ([]event.Event, error) {
							return []event.Event{mustEvent(event.New("task.created",
								cmd.StreamID(), "Task", ver+1,
								TaskCreated{Title: "conc", At: time.Now()}))}, nil
						})
				})
		},
	}

	sys, err := system.New(ctx, domain, sqliteDeployment(t))
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}
	defer sys.Close()

	// Direct Save with wrong version should fail.
	ref := id.NewStreamRef("Task", id.NewStreamID())
	err = sys.EventStore().Save(ctx, ref,
		[]event.Event{mustEvent(event.New("task.created", ref.ID, "Task", 1,
			TaskCreated{Title: "x", At: time.Now()}))},
		event.Version(0))
	if err != nil {
		t.Fatalf("first Save should succeed: %v", err)
	}

	// Second Save with same expected version (0) should conflict.
	err = sys.EventStore().Save(ctx, ref,
		[]event.Event{mustEvent(event.New("task.completed", ref.ID, "Task", 2,
			TaskCompleted{At: time.Now()}))},
		event.Version(0))
	if err == nil {
		t.Fatal("expected version conflict on second Save")
	}

	if !errors.Is(err, event.ErrVersionConflict) {
		t.Fatalf("expected ErrVersionConflict, got %v", err)
	}
}

func TestSystem_SQLiteJournal(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	domain := system.DomainConfig{
		Commands: func(sys *system.System) {
			system.RegisterDecider(sys, "Task", TaskDecider)

			system.RegisterCommand[*command.BasicCommand, TaskState](sys, "task.create",
				func(ctx context.Context, cmd *command.BasicCommand) system.Op[TaskState] {
					return system.Execute(ctx, cmd.StreamID(), "Task",
						func(state TaskState, ver event.Version) ([]event.Event, error) {
							return []event.Event{mustEvent(event.New("task.created",
								cmd.StreamID(), "Task", ver+1,
								TaskCreated{Title: "journal", At: time.Now()}))}, nil
						})
				})
		},
	}

	sys, err := system.New(ctx, domain, sqliteDeployment(t))
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}
	defer sys.Close()

	for i := range 3 {
		streamID := id.NewStreamID()
		cmd := newCmd("task.create", streamID)
		if err := sys.CommandDispatcher().Dispatch(ctx, cmd); err != nil {
			t.Fatalf("dispatch %d: %v", i, err)
		}
	}

	journal := sys.EventStore().(event.Journal)
	all, err := journal.ReadAll(ctx)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	if len(all) != 3 {
		t.Fatalf("expected 3 events in journal, got %d", len(all))
	}

	sj := sys.EventStore().(event.SeekableJournal)
	from, err := sj.ReadFrom(ctx, all[0].ID(), 10)
	if err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}

	if len(from) != 2 {
		t.Fatalf("expected 2 events after first, got %d", len(from))
	}
}

func TestSystem_SQLitePersistence(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test.db")

	domain := system.DomainConfig{
		Commands: func(sys *system.System) {
			system.RegisterDecider(sys, "Task", TaskDecider)

			system.RegisterCommand[*command.BasicCommand, TaskState](sys, "task.create",
				func(ctx context.Context, cmd *command.BasicCommand) system.Op[TaskState] {
					return system.Execute(ctx, cmd.StreamID(), "Task",
						func(state TaskState, ver event.Version) ([]event.Event, error) {
							return []event.Event{mustEvent(event.New("task.created",
								cmd.StreamID(), "Task", ver+1,
								TaskCreated{Title: "persist", At: time.Now()}))}, nil
						})
				})
		},
	}

	deployment := system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			"primary": {Driver: "sqlite", DSN: dbPath},
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
		},
	}

	// First system: write events.
	sys1, err := system.New(ctx, domain, deployment)
	if err != nil {
		t.Fatalf("system.New (first): %v", err)
	}

	taskStreamID := id.NewStreamID()
	if err := sys1.CommandDispatcher().Dispatch(ctx, newCmd("task.create", taskStreamID)); err != nil {
		t.Fatalf("dispatch: %v", err)
	}

	if err := sys1.Close(); err != nil {
		t.Fatalf("Close first: %v", err)
	}

	// Second system: same DSN, verify events survived restart.
	sys2, err := system.New(ctx, domain, deployment)
	if err != nil {
		t.Fatalf("system.New (second): %v", err)
	}
	defer sys2.Close()

	ref := id.NewStreamRef("Task", taskStreamID)
	events, err := sys2.EventStore().Load(ctx, ref)
	if err != nil {
		t.Fatalf("load after restart: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event after restart, got %d", len(events))
	}

	if events[0].Type() != "task.created" {
		t.Fatalf("expected task.created, got %s", events[0].Type())
	}

	var p TaskCreated
	if err := json.Unmarshal(events[0].Payload(), &p); err != nil {
		decoded, derr := event.DecodePayloadAuto[TaskCreated](events[0])
		if derr != nil {
			t.Fatalf("decode payload: %v", derr)
		}
		p = decoded
	}
	if p.Title != "persist" {
		t.Fatalf("expected title %q, got %q", "persist", p.Title)
	}
}

func TestSystem_SQLiteDriverRegistered(t *testing.T) {
	t.Parallel()

	drivers := system.RegisteredDrivers()
	found := false

	for _, d := range drivers {
		if d == "sqlite" {
			found = true
		}
	}

	if !found {
		t.Fatal("sqlite driver not registered")
	}
}

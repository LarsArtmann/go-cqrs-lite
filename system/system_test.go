package system_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// ── Domain types ──

type TaskStreamID = id.Of[id.StreamMarker]

type CreateTaskCmd struct {
	cmdID  id.CommandID
	taskID id.StreamID
	title  string
}

func (c *CreateTaskCmd) Type() command.Type   { return "task.create" }
func (c *CreateTaskCmd) StreamID() id.StreamID { return c.taskID }
func (c *CreateTaskCmd) ID() id.CommandID      { return c.cmdID }

type CompleteTaskCmd struct {
	cmdID  id.CommandID
	taskID id.StreamID
}

func (c *CompleteTaskCmd) Type() command.Type   { return "task.complete" }
func (c *CompleteTaskCmd) StreamID() id.StreamID { return c.taskID }
func (c *CompleteTaskCmd) ID() id.CommandID      { return c.cmdID }

type TaskCreated struct {
	Title string
	At    time.Time
}

type TaskCompleted struct {
	At time.Time
}

type TaskState struct {
	Title  string
	Status string
	Exists bool
}

func applyTask(state TaskState, evt event.Event) (TaskState, error) {
	switch evt.Type() {
	case "task.created":
		var p TaskCreated
		_ = json.Unmarshal(evt.Payload(), &p)
		state.Title = p.Title
		state.Status = "pending"
		state.Exists = true
	case "task.completed":
		state.Status = "completed"
	}
	return state, nil
}

var TaskDecider = decider.Decider[TaskState]{
	Initial: TaskState{},
	Apply:   applyTask,
}

func mustEvent(evt event.Event, err error) event.Event {
	if err != nil {
		panic(err)
	}
	return evt
}

func newCreateCmd(title string) *CreateTaskCmd {
	return &CreateTaskCmd{
		cmdID:  id.New[id.CommandID](),
		taskID: id.New[id.StreamID](),
		title:  title,
	}
}

func newCompleteCmd(taskID id.StreamID) *CompleteTaskCmd {
	return &CompleteTaskCmd{
		cmdID:  id.New[id.CommandID](),
		taskID: taskID,
	}
}

// ── Tests ──

func TestSystem_FullCQRSRoundtrip(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	domain := system.DomainConfig{
		Commands: func(sys *system.System) {
			system.RegisterDecider(sys, "Task", TaskDecider)

			system.RegisterCommand[*CreateTaskCmd, TaskState](sys, "task.create",
				func(ctx context.Context, cmd *CreateTaskCmd) system.Op[TaskState] {
					return system.Execute(ctx, cmd.StreamID(), "Task",
						func(state TaskState, ver event.Version) ([]event.Event, error) {
							if state.Exists {
								return nil, fmt.Errorf("task already exists")
							}
							return []event.Event{mustEvent(event.New("task.created",
								cmd.StreamID(), "Task", ver+1,
								TaskCreated{Title: cmd.title, At: time.Now()}))}, nil
						})
				})

			system.RegisterCommand[*CompleteTaskCmd, TaskState](sys, "task.complete",
				func(ctx context.Context, cmd *CompleteTaskCmd) system.Op[TaskState] {
					return system.Execute(ctx, cmd.StreamID(), "Task",
						func(state TaskState, ver event.Version) ([]event.Event, error) {
							if !state.Exists {
								return nil, fmt.Errorf("task not found")
							}
							return []event.Event{mustEvent(event.New("task.completed",
								cmd.StreamID(), "Task", ver+1,
								TaskCompleted{At: time.Now()}))}, nil
						})
				})
		},
	}

	sys, err := system.New(ctx, domain, system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{"primary": {Driver: "memory"}},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
		},
	})
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}
	defer sys.Close()

	// Create a task
	createCmd := newCreateCmd("Write tests")
	if err := sys.CommandDispatcher().Dispatch(ctx, createCmd); err != nil {
		t.Fatalf("dispatch create: %v", err)
	}

	// Verify events
	ref := id.NewStreamRef("Task", createCmd.StreamID())
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

	// Complete the task
	completeCmd := newCompleteCmd(createCmd.StreamID())
	if err := sys.CommandDispatcher().Dispatch(ctx, completeCmd); err != nil {
		t.Fatalf("dispatch complete: %v", err)
	}

	events, _ = sys.EventStore().Load(ctx, ref)
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	// Optimistic concurrency conflict
	err = sys.EventStore().Save(ctx, ref,
		[]event.Event{mustEvent(event.New("task.completed", createCmd.StreamID(), "Task", 99,
			TaskCompleted{At: time.Now()}))},
		event.Version(0))
	if err == nil {
		t.Fatal("expected version conflict, got nil")
	}
}

func TestSystem_Journal(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	domain := system.DomainConfig{
		Commands: func(sys *system.System) {
			system.RegisterDecider(sys, "Task", TaskDecider)
			system.RegisterCommand[*CreateTaskCmd, TaskState](sys, "task.create",
				func(ctx context.Context, cmd *CreateTaskCmd) system.Op[TaskState] {
					return system.Execute(ctx, cmd.StreamID(), "Task",
						func(state TaskState, ver event.Version) ([]event.Event, error) {
							return []event.Event{mustEvent(event.New("task.created",
								cmd.StreamID(), "Task", ver+1,
								TaskCreated{Title: cmd.title, At: time.Now()}))}, nil
						})
				})
		},
	}

	sys, _ := system.New(ctx, domain, system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{"primary": {Driver: "memory"}},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
		},
	})
	defer sys.Close()

	for i := range 3 {
		cmd := newCreateCmd(fmt.Sprintf("Task %d", i))
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

func TestSystem_Close(t *testing.T) {
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
	if err := sys.Close(); err != nil {
		t.Fatalf("double Close: %v", err)
	}
}

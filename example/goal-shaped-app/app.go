// The developer's wiring: ONE Evolution (folds declared once per result
// type), two read shapes (a point Lookup and a filtered QuerySet), and
// typed command/query handlers. Nothing here names an engine, a DSN, a
// schema, or a limit — that is cqrs.yaml's job, and the same wiring runs
// unchanged on every driver the operator picks.
package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

const (
	streamType      = "Task"
	tasksCollection = "tasks"
	openCollection  = "open_tasks"

	evtTaskCreated   = event.Type("task.created")
	evtTaskCompleted = event.Type("task.completed")
	evtTaskDeleted   = event.Type("task.deleted")

	cmdCreateTask   = command.Type("task.create")
	cmdCompleteTask = command.Type("task.complete")
	cmdDeleteTask   = command.Type("task.delete")

	qryGetTask   = query.Type("task.get")
	qryOpenTasks = query.Type("task.open")

	settleDeadline = 5 * time.Second
	settlePoll     = 20 * time.Millisecond
)

var (
	errTaskExists = errors.New("task already exists")
	errTaskDone   = errors.New("task already done")
	errTaskGone   = errors.New("task not found")
)

// TaskState is the write-side invariant state (replay, never stored).
type TaskState struct {
	Exists bool
	Done   bool
}

func applyTask(s TaskState, evt event.Event) (TaskState, error) {
	switch evt.Type() {
	case evtTaskCreated:
		s.Exists = true
	case evtTaskCompleted:
		s.Done = true
	}

	return s, nil
}

// Domain declares everything the developer owns. The Evolution is the only
// place folds exist; Lookup and QuerySet both inherit them by result type.
func Domain() system.DomainConfig {
	created := system.OnEvolution(
		system.Evolve[TaskView](tasksCollection),
		string(evtTaskCreated), TaskCreated{},
	)
	completed := system.OnEvolution(
		created,
		string(evtTaskCompleted), TaskCompleted{},
		func(_ TaskCompleted, v *TaskView) { v.Status = StatusDone },
	)
	deleted := system.OnEvolution(completed, string(evtTaskDeleted), TaskDeleted{})

	return system.DomainConfig{
		Evolutions: []system.EvolutionSpec{deleted.Done()},
		Projections: []system.ProjectionDeclaration{
			system.Lookup[TaskView](tasksCollection).Done(),
			system.QuerySet[TaskView](openCollection).Filterable("status").Done(),
		},
		Commands: registerCommands,
		Queries:  registerQueries,
	}
}

func registerCommands(sys *system.System) {
	deciderErr := system.RegisterDecider(sys, streamType, decider.Decider[TaskState]{
		Initial: TaskState{},
		Apply:   applyTask,
	})
	must(deciderErr)

	createErr := system.RegisterCommand[CreateTaskCmd, TaskState](sys, cmdCreateTask,
		func(ctx context.Context, cmd CreateTaskCmd) system.Op[TaskState] {
			return system.Execute(ctx, cmd.StreamID(), streamType,
				func(s TaskState, v event.Version) ([]event.Event, error) {
					return decideCreate(cmd, s, v)
				})
		})
	must(createErr)

	completeErr := system.RegisterCommand[CompleteTaskCmd, TaskState](sys, cmdCompleteTask,
		func(ctx context.Context, cmd CompleteTaskCmd) system.Op[TaskState] {
			return system.Execute(ctx, cmd.StreamID(), streamType,
				func(s TaskState, v event.Version) ([]event.Event, error) {
					return decideComplete(cmd, s, v)
				})
		})
	must(completeErr)

	deleteErr := system.RegisterCommand[DeleteTaskCmd, TaskState](sys, cmdDeleteTask,
		func(ctx context.Context, cmd DeleteTaskCmd) system.Op[TaskState] {
			return system.Execute(ctx, cmd.StreamID(), streamType,
				func(s TaskState, v event.Version) ([]event.Event, error) {
					return decideDelete(cmd, s, v)
				})
		})
	must(deleteErr)
}

func decideCreate(cmd CreateTaskCmd, s TaskState, v event.Version) ([]event.Event, error) {
	if s.Exists {
		return nil, errTaskExists
	}

	evt, err := event.New(evtTaskCreated, cmd.StreamID(), streamType, v.Increment(),
		TaskCreated{ID: cmd.StreamID().String(), Title: cmd.Title, Priority: cmd.Priority})
	if err != nil {
		return nil, fmt.Errorf("task.created: %w", err)
	}

	return []event.Event{evt}, nil
}

func decideComplete(cmd CompleteTaskCmd, s TaskState, v event.Version) ([]event.Event, error) {
	if !s.Exists {
		return nil, errTaskGone
	}

	if s.Done {
		return nil, errTaskDone
	}

	evt, err := event.New(evtTaskCompleted, cmd.StreamID(), streamType, v.Increment(),
		TaskCompleted{ID: cmd.StreamID().String()})
	if err != nil {
		return nil, fmt.Errorf("task.completed: %w", err)
	}

	return []event.Event{evt}, nil
}

func decideDelete(cmd DeleteTaskCmd, s TaskState, v event.Version) ([]event.Event, error) {
	if !s.Exists {
		return nil, errTaskGone
	}

	evt, err := event.New(evtTaskDeleted, cmd.StreamID(), streamType, v.Increment(),
		TaskDeleted{ID: cmd.StreamID().String()})
	if err != nil {
		return nil, fmt.Errorf("task.deleted: %w", err)
	}

	return []event.Event{evt}, nil
}

func registerQueries(sys *system.System) {
	byID := metaengine.NewReader[TaskView](sys.MetaEngine(), tasksCollection)
	open := metaengine.NewReader[TaskView](sys.MetaEngine(), openCollection)

	getErr := system.RegisterQuery[GetTask, TaskView](sys, string(qryGetTask),
		func(ctx context.Context, q GetTask) (TaskView, error) {
			view, found, err := byID.Get(ctx, q.ID)
			if err != nil {
				return TaskView{}, err
			}

			if !found {
				return TaskView{}, errTaskGone
			}

			return view, nil
		})
	must(getErr)

	openErr := system.RegisterQuery[OpenTasks, []TaskView](sys, string(qryOpenTasks),
		func(ctx context.Context, _ OpenTasks) ([]TaskView, error) {
			return open.Scan(ctx, metaengine.WithFilter("status", metaengine.FilterEq, StatusOpen))
		})
	must(openErr)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

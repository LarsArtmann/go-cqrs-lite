// The five-minute story: two tasks in, one completed, one deleted, read
// back through the typed query bus. It runs identically on every engine
// the operator picks — run() in main.go composes the system from cqrs.yaml.
package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

var errStoryNotDone = errors.New("story: task never reached done in the read model")

// runStory boots the projection host, exercises every declared fold
// (create, complete, delete), and returns the surviving task's view.
func runStory(ctx context.Context, sys *system.System) (TaskView, error) {
	if err := sys.Start(ctx); err != nil {
		return TaskView{}, fmt.Errorf("start system: %w", err)
	}

	doneID, goneID := id.NewStreamID(), id.NewStreamID()

	steps := []struct {
		name string
		run  func() error
	}{
		{"create (Ship The Goal demo)", func() error { return createTask(ctx, sys, doneID, "Ship The Goal demo", 2) }},
		{"create (Break the build)", func() error { return createTask(ctx, sys, goneID, "Break the build", 5) }},
		{"complete the first", func() error { return dispatch(ctx, sys, cmdCompleteTask, doneID) }},
		{"delete the second", func() error { return dispatch(ctx, sys, cmdDeleteTask, goneID) }},
	}

	for _, step := range steps {
		if err := step.run(); err != nil {
			return TaskView{}, fmt.Errorf("story %s: %w", step.name, err)
		}
	}

	view, err := awaitDone(ctx, sys, doneID.String())
	if err != nil {
		return TaskView{}, err
	}

	openBasic, err := query.New(qryOpenTasks)
	if err != nil {
		return TaskView{}, fmt.Errorf("query.New: %w", err)
	}

	open, err := system.DispatchQuery[OpenTasks, []TaskView](ctx, sys, OpenTasks{BasicQuery: openBasic})
	if err != nil {
		return TaskView{}, fmt.Errorf("task.open: %w", err)
	}

	fmt.Printf("open tasks now: %d (completed is filtered, deleted is gone)\n", len(open))

	return view, nil
}

func createTask(ctx context.Context, sys *system.System, taskID id.StreamID, title string, priority int) error {
	basic, err := command.New(cmdCreateTask, taskID)
	if err != nil {
		return fmt.Errorf("command.New: %w", err)
	}

	return sys.CommandDispatcher().Dispatch(ctx,
		CreateTaskCmd{BasicCommand: basic, Title: title, Priority: priority})
}

func dispatch(ctx context.Context, sys *system.System, kind command.Type, taskID id.StreamID) error {
	basic, err := command.New(kind, taskID)
	if err != nil {
		return fmt.Errorf("command.New: %w", err)
	}

	var cmd command.Command = basic
	switch kind {
	case cmdCompleteTask:
		cmd = CompleteTaskCmd{BasicCommand: basic}
	case cmdDeleteTask:
		cmd = DeleteTaskCmd{BasicCommand: basic}
	}

	return sys.CommandDispatcher().Dispatch(ctx, cmd)
}

// awaitGone polls the task.get query until the deleted task's view has
// been removed from the read model (the tombstone folded everywhere).
func awaitGone(ctx context.Context, sys *system.System, taskID string) error {
	basic, err := query.New(qryGetTask)
	if err != nil {
		return fmt.Errorf("query.New: %w", err)
	}

	for deadline := time.Now().Add(settleDeadline); ; {
		_, qerr := system.DispatchQuery[GetTask, TaskView](ctx, sys, GetTask{BasicQuery: basic, ID: taskID})
		if errors.Is(qerr, errTaskGone) {
			return nil
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("story: deleted task still readable: %w", qerr)
		}

		time.Sleep(settlePoll)
	}
}

// awaitDone polls the task.get query until the projection host has folded
// the completed fact into the read model.
func awaitDone(ctx context.Context, sys *system.System, taskID string) (TaskView, error) {
	basic, err := query.New(qryGetTask)
	if err != nil {
		return TaskView{}, fmt.Errorf("query.New: %w", err)
	}

	for deadline := time.Now().Add(settleDeadline); ; {
		view, qerr := system.DispatchQuery[GetTask, TaskView](ctx, sys, GetTask{BasicQuery: basic, ID: taskID})
		if qerr == nil && view.Status == StatusDone {
			return view, nil
		}

		if time.Now().After(deadline) {
			return TaskView{}, fmt.Errorf("%w: %v", errStoryNotDone, qerr)
		}

		time.Sleep(settlePoll)
	}
}

package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// The five-minute story: two tasks in, one completed, one deleted, read
// back through the typed query bus. It runs identically on every engine
// the operator picks — run() in main.go composes the system from cqrs.yaml.

var (
	errStoryNotDone  = errors.New("story: task never reached done in the read model")
	errNoConstructor = errors.New("story: no typed constructor for command")
)

// runStory boots the projection host, exercises every declared fold
// (create, complete, delete), and returns the surviving task's view.
func runStory(ctx context.Context, sys *system.System) (TaskView, error) {
	if err := sys.Start(ctx); err != nil {
		return TaskView{}, fmt.Errorf("start system: %w", err)
	}

	doneID, goneID := id.NewStreamID(), id.NewStreamID()

	if err := runSteps(ctx, sys, doneID, goneID); err != nil {
		return TaskView{}, err
	}

	view, err := awaitDone(ctx, sys, doneID)
	if err != nil {
		return TaskView{}, err
	}

	openQuery, err := NewOpenTasks(OpenTasks{})
	if err != nil {
		return TaskView{}, err
	}

	open, err := system.DispatchQuery[OpenTasks, []TaskView](ctx, sys, openQuery)
	if err != nil {
		return TaskView{}, fmt.Errorf("task.open: %w", err)
	}

	fmt.Printf("open tasks now: %d (completed is filtered, deleted is gone)\n", len(open))

	return view, nil
}

// runSteps dispatches the four commands whose facts the read models fold.
func runSteps(ctx context.Context, sys *system.System, doneID, goneID id.StreamID) error {
	steps := []struct {
		name string
		run  func() error
	}{
		{"create (Ship The Goal demo)", func() error {
			return createTask(ctx, sys, doneID, "Ship The Goal demo", PriorityNormal)
		}},
		{"create (Break the build)", func() error {
			return createTask(ctx, sys, goneID, "Break the build", PriorityHigh)
		}},
		{"complete the first", func() error { return dispatch(ctx, sys, cmdCompleteTask, doneID) }},
		{"delete the second", func() error { return dispatch(ctx, sys, cmdDeleteTask, goneID) }},
	}

	for _, step := range steps {
		if err := step.run(); err != nil {
			return fmt.Errorf("story %s: %w", step.name, err)
		}
	}

	return nil
}

func createTask(
	ctx context.Context,
	sys *system.System,
	taskID id.StreamID,
	title string,
	priority int,
) error {
	cmd, err := NewCreateTaskCmd(taskID, CreateTaskCmd{Title: title, Priority: priority})
	if err != nil {
		return err
	}

	return sys.CommandDispatcher().Dispatch(ctx, cmd)
}

// dispatch builds every command through the constructor cqrs-gen derives, so
// a wire name can never be paired with the wrong payload type.
func dispatch(
	ctx context.Context,
	sys *system.System,
	kind command.Type,
	taskID id.StreamID,
) error {
	var (
		cmd command.Command
		err error
	)

	switch kind {
	case cmdCreateTask:
		cmd, err = NewCreateTaskCmd(taskID, CreateTaskCmd{})
	case cmdCompleteTask:
		cmd, err = NewCompleteTaskCmd(taskID, CompleteTaskCmd{})
	case cmdDeleteTask:
		cmd, err = NewDeleteTaskCmd(taskID, DeleteTaskCmd{})
	default:
		return fmt.Errorf("%w %s", errNoConstructor, kind)
	}

	if err != nil {
		return err
	}

	return sys.CommandDispatcher().Dispatch(ctx, cmd)
}

// awaitGone polls the task.get query until the deleted task's view has
// been removed from the read model (the tombstone folded everywhere).
func awaitGone(ctx context.Context, sys *system.System, taskID id.StreamID) error {
	for deadline := time.Now().Add(settleDeadline); ; {
		get, err := NewGetTask(GetTask{ID: taskID})
		if err != nil {
			return err
		}

		_, qerr := system.DispatchQuery[GetTask, TaskView](ctx, sys, get)
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
func awaitDone(ctx context.Context, sys *system.System, taskID id.StreamID) (TaskView, error) {
	for deadline := time.Now().Add(settleDeadline); ; {
		get, err := NewGetTask(GetTask{ID: taskID})
		if err != nil {
			return TaskView{}, err
		}

		view, qerr := system.DispatchQuery[GetTask, TaskView](ctx, sys, get)
		if qerr == nil && view.Status == StatusDone {
			return view, nil
		}

		if time.Now().After(deadline) {
			return TaskView{}, fmt.Errorf("%w: %w", errStoryNotDone, qerr)
		}

		time.Sleep(settlePoll)
	}
}

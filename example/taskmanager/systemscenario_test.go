package main

import (
	"context"
	"fmt"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/systemscenario/v4"
)

// adoptTaskmanager boots the production facade (NewServer + Start) and wraps
// it with the systemscenario harness via Adopt — the entry point for
// facades that own system.New themselves. The Server's t.Cleanup owns the
// lifecycle; Adopt deliberately registers no shutdown of its own.
func adoptTaskmanager(t *testing.T) (*Server, *systemscenario.Scenario) {
	t.Helper()

	srv, err := NewServer(DefaultConfig(), nil)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	t.Cleanup(func() { _ = srv.Stop() })

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	srv.Start(ctx)

	return srv, systemscenario.Adopt(t, context.Background(), srv.Sys)
}

// taskView returns a probe reading the TaskReader read model for taskID.
func taskView(srv *Server, taskID id.StreamID) func() (any, error) {
	return func() (any, error) {
		view, _, err := srv.TaskReader.Get(context.Background(), taskID.String())

		return view, err
	}
}

// taskViewAbsent returns a probe reporting whether the view still exists —
// TaskReader.Get returns found=false (not an error) for a deleted task, so
// absence must be carried as the probe's value, not its error.
func taskViewAbsent(srv *Server, taskID id.StreamID) func() (any, error) {
	return func() (any, error) {
		_, found, err := srv.TaskReader.Get(context.Background(), taskID.String())
		if err != nil {
			return nil, err
		}

		return found, nil
	}
}

// createdView checks title, priority, and the deriver's auto-assignment in
// one poll: requiring AssigneeID here is load-bearing — the deriver's
// task.assign rides the durable workqueue, and dispatching task.start
// before it commits hits an optimistic-concurrency conflict.
func createdView(view TaskView) error {
	if view.Title != "Scenario Task" {
		return fmt.Errorf("Title: want %q, got %q", "Scenario Task", view.Title)
	}

	if view.Priority != PriorityHigh {
		return fmt.Errorf("Priority: want %q, got %q", PriorityHigh, view.Priority)
	}

	if view.AssigneeID != defaultAssignee.Get() {
		return fmt.Errorf("AssigneeID: want %q, got %q", defaultAssignee, view.AssigneeID)
	}

	return nil
}

func statusIs(want Status) func(any) error {
	return func(got any) error {
		view, ok := got.(TaskView)
		if !ok {
			return fmt.Errorf("view: got %T, want TaskView", got)
		}

		if view.Status != want {
			return fmt.Errorf("Status: want %q, got %q", want, view.Status)
		}

		return nil
	}
}

// assertView adapts a typed TaskView check into a ThenQueryFunc check.
func assertView(check func(TaskView) error) func(any) error {
	return func(got any) error {
		view, ok := got.(TaskView)
		if !ok {
			return fmt.Errorf("view: got %T, want TaskView", got)
		}

		return check(view)
	}
}

// assertAbsent passes only when the probe reported the view gone.
func assertAbsent(got any) error {
	if found, ok := got.(bool); !ok || found {
		return fmt.Errorf("task view still present: %#v", got)
	}

	return nil
}

// TestScenario_FullLifecycle replays the full CQRS pipeline through the
// systemscenario harness: create → deriver auto-assign → start → complete →
// delete, asserting the read model after each step via polling Then*.
func TestScenario_FullLifecycle(t *testing.T) {
	t.Parallel()

	srv, sc := adoptTaskmanager(t)

	taskID := id.NewStreamID()

	sc.Given().
		When(CreateTaskCmd{
			BasicCommand: Must(command.New(cmdCreateTask, taskID)),
			Title:        "Scenario Task",
			Priority:     PriorityHigh,
		}).
		ThenSuccess().
		ThenQueryFunc(taskView(srv, taskID), assertView(createdView)).
		Command(StartTaskCmd{BasicCommand: Must(command.New(cmdStartTask, taskID))}).
		ThenQueryFunc(taskView(srv, taskID), statusIs(StatusActive)).
		Command(CompleteTaskCmd{BasicCommand: Must(command.New(cmdCompleteTask, taskID))}).
		ThenQueryFunc(taskView(srv, taskID), statusIs(StatusCompleted)).
		Command(DeleteTaskCmd{BasicCommand: Must(command.New(cmdDeleteTask, taskID))}).
		ThenQueryFunc(taskViewAbsent(srv, taskID), assertAbsent)
}

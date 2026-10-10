package main

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
	"github.com/larsartmann/go-cqrs-lite/systemscenario/v4"
)

// newCmd builds a basic command for kind on taskID, failing the test on
// construction errors.
func newCmd(t *testing.T, kind command.Type, taskID id.StreamID) *command.BasicCommand {
	t.Helper()

	basic, err := command.New(kind, taskID)
	if err != nil {
		t.Fatalf("command.New(%s): %v", kind, err)
	}

	return basic
}

// goalDoneView checks the surviving task reached the done state the story
// pins: completed, correct title, priority folded through.
func goalDoneView(view TaskView) error {
	if view.Status != StatusDone {
		return fmt.Errorf("Status: want %q, got %q", StatusDone, view.Status)
	}

	if view.Title != "Ship The Goal demo" {
		return fmt.Errorf("Title: want %q, got %q", "Ship The Goal demo", view.Title)
	}

	if view.Priority != PriorityNormal {
		return fmt.Errorf("Priority: want %d, got %d", PriorityNormal, view.Priority)
	}

	return nil
}

// wantGone passes only when the probe reported the query erroring with
// errTaskGone — deletion is a domain event; the lookup reports not-found
// instead of resurrecting state.
func wantGone(got any) error {
	if gone, ok := got.(bool); !ok || !gone {
		return fmt.Errorf("deleted task still readable: %#v", got)
	}

	return nil
}

// wantNoOpenTasks passes when the OpenTasks query reports an empty slice:
// the completed task is filtered and the deleted task is gone.
func wantNoOpenTasks(got any) error {
	open, ok := got.([]TaskView)
	if !ok {
		return fmt.Errorf("open tasks: got %T, want []TaskView", got)
	}

	if len(open) != 0 {
		return fmt.Errorf("open tasks: want none, got %d (%+v)", len(open), open)
	}

	return nil
}

// TestScenario_GoalStory replays runStory through the systemscenario harness
// over the production boot (cqrs.yaml → LoadConfig → system.New): the two
// creates seed Given, complete/delete are When acts, and the read models are
// asserted through the same typed query bus the binary serves.
func TestScenario_GoalStory(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	sys := boot(t, ctx, sqliteConfig(t))

	if err := sys.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	sc := systemscenario.Adopt(t, ctx, sys)

	doneID, goneID := id.NewStreamID(), id.NewStreamID()

	getBasic, err := query.New(qryGetTask)
	if err != nil {
		t.Fatalf("query.New: %v", err)
	}

	openBasic, err := query.New(qryOpenTasks)
	if err != nil {
		t.Fatalf("query.New: %v", err)
	}

	doneView := func() (any, error) {
		return system.DispatchQuery[GetTask, TaskView](ctx, sys,
			GetTask{BasicQuery: getBasic, ID: doneID.String()})
	}

	goneView := func() (any, error) {
		view, qerr := system.DispatchQuery[GetTask, TaskView](ctx, sys,
			GetTask{BasicQuery: getBasic, ID: goneID.String()})
		if errors.Is(qerr, errTaskGone) {
			return true, nil
		}

		if qerr != nil {
			return nil, qerr
		}

		return false, fmt.Errorf("deleted task still readable: %+v", view)
	}

	openView := func() (any, error) {
		return system.DispatchQuery[OpenTasks, []TaskView](ctx, sys,
			OpenTasks{BasicQuery: openBasic})
	}

	sc.Given().
		Command(
			CreateTaskCmd{
				BasicCommand: newCmd(t, cmdCreateTask, doneID),
				Title:        "Ship The Goal demo",
				Priority:     PriorityNormal,
			},
			CreateTaskCmd{
				BasicCommand: newCmd(t, cmdCreateTask, goneID),
				Title:        "Break the build",
				Priority:     PriorityHigh,
			},
		).
		When(CompleteTaskCmd{BasicCommand: newCmd(t, cmdCompleteTask, doneID)}).
		ThenSuccess().
		Command(DeleteTaskCmd{BasicCommand: newCmd(t, cmdDeleteTask, goneID)}).
		ThenQueryFunc(doneView, func(got any) error {
			view, ok := got.(TaskView)
			if !ok {
				return fmt.Errorf("view: got %T, want TaskView", got)
			}

			return goalDoneView(view)
		}).
		ThenQueryFunc(goneView, wantGone).
		ThenQueryFunc(openView, wantNoOpenTasks)
}

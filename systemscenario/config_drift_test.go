package systemscenario_test

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// TestSystem_ConfigDriftDemo is the fixture-from-production-config proof
// (ADR-0153): the SAME taskDomain() value boots a raw production-style
// system AND the harness; both observe the identical journal outcome. If the
// harness ever re-declared wiring, this test would be the first drift
// detector.
func TestSystem_ConfigDriftDemo(t *testing.T) {
	t.Parallel()

	ref := id.NewStreamRef("Task", id.NewStreamID())

	// Production-style boot: system.New + Start + dispatch + journal read.
	rawTypes := bootRawAndDispatch(t, ref)

	// Harness boot: same DomainConfig, same DeploymentConfig.
	sc, href, _ := newTaskScenario(t)

	sc.Given(
		sc.Event("task.created", href, TaskCreated{ID: href.ID.String(), Status: "pending"}),
	).When(newTaskCmd("task.complete", href.ID)).
		Then("task.updated")

	want := []event.Type{"task.created", "task.updated"}
	if len(rawTypes) != len(want) {
		t.Fatalf("raw boot: want journal %v, got %v", want, rawTypes)
	}

	for i := range want {
		if rawTypes[i] != want[i] {
			t.Fatalf("raw boot: want journal %v, got %v", want, rawTypes)
		}
	}
}

// bootRawAndDispatch boots system.New directly (no harness), runs the
// create→complete flow, and returns the raw journal's event types.
func bootRawAndDispatch(t *testing.T, ref id.StreamRef) []event.Type {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sys, err := system.New(ctx, taskDomain(), memoryDeployment())
	if err != nil {
		t.Fatalf("raw boot: system.New: %v", err)
	}

	defer func() { _ = sys.Close() }()

	if err := sys.Start(ctx); err != nil {
		t.Fatalf("raw boot: Start: %v", err)
	}

	for _, cmdType := range []command.Type{"task.create", "task.complete"} {
		if err := sys.CommandDispatcher().Dispatch(ctx, newTaskCmd(cmdType, ref.ID)); err != nil {
			t.Fatalf("raw boot: dispatch %s: %v", cmdType, err)
		}
	}

	events, err := sys.EventStore().Load(ctx, ref)
	if err != nil {
		t.Fatalf("raw boot: load stream: %v", err)
	}

	types := make([]event.Type, len(events))
	for i, evt := range events {
		types[i] = evt.Type()
	}

	return types
}

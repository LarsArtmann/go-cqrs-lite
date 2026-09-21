package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// docs_compile_test.go — compile-gate for the README's API fences (the
// getting-started pattern): if the system API drifts, this test breaks in CI
// instead of the README silently lying.

// memoryDeployment composes the demo on the memory driver (no config file).
func memoryDeployment() system.DeploymentConfig {
	return system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			"primary": {Driver: "memory"},
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
			{Role: system.RoleProjections, Engine: "primary"},
		},
	}
}

// TestDocs_ReadmeEvolutionFence compiles and runs the README's Evolution +
// inherited read-shapes fence verbatim (against the memory driver), so the
// Goal-in-one-fence snippet cannot rot.
func TestDocs_ReadmeEvolutionFence(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// BEGIN README fence (example/goal-shaped-app/README.md) — keep in sync.
	tasks := system.OnEvolution(
		system.OnEvolution(
			system.Evolve[TaskView]("tasks"),
			"task.created", TaskCreated{},
		),
		"task.updated", TaskUpdated{},
	)
	deleted := system.OnEvolution(tasks, "task.deleted", TaskDeleted{})

	domain := system.DomainConfig{
		Events:     []event.Type{evtTaskCreated, evtTaskUpdated, evtTaskDeleted},
		Evolutions: []system.EvolutionSpec{deleted.Done()},
		Projections: []system.ProjectionDeclaration{
			system.Lookup[TaskView]("tasks").Done(),
			system.QuerySet[TaskView]("open_tasks").Filterable("status").Done(),
		},
	}
	// END README fence

	sys, err := system.New(ctx, domain, memoryDeployment())
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}

	defer sys.Close()
}

// TestDocs_CoeffectGate_DanglingSubscriptionFailsLoud demos the coeffect gate
// the Events declaration arms: a projection subscribing to a type that is not
// in the declared event universe fails system.New with a hard error (the
// classic cause is a typo'd event string).
func TestDocs_CoeffectGate_DanglingSubscriptionFailsLoud(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tasks := system.OnEvolution(
		system.Evolve[TaskView]("tasks"),
		"task.created", TaskCreated{},
	)
	dangling := system.OnEvolution(tasks, "task.deleated", TaskDeleted{}) // typo'd

	domain := system.DomainConfig{
		Events:     []event.Type{evtTaskCreated}, // "task.deleated" is NOT declared
		Evolutions: []system.EvolutionSpec{dangling.Done()},
		Projections: []system.ProjectionDeclaration{
			system.Lookup[TaskView]("tasks").Done(),
		},
	}

	_, err := system.New(ctx, domain, memoryDeployment())
	if err == nil {
		t.Fatal("expected system.New to reject the dangling event subscription")
	}

	if !strings.Contains(err.Error(), "task.deleated") &&
		!errors.Is(err, system.ErrDanglingEventSubscription) {
		t.Fatalf("error should name the undeclared type, got: %v", err)
	}
}

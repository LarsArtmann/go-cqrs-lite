package system_test

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// ── Evolution test types ──

type EvoCreated struct {
	ID     string
	Title  string
	Status string
}

type EvoStarted struct {
	ID string
}

type EvoCompleted struct {
	ID string
}

type EvoDeleted struct {
	ID string
}

type EvoView struct {
	ID     string
	Title  string
	Status string
}

// TestSystem_Evolution_Convention verifies that a Lookup without its own
// samples inherits folds from the matching Evolution by result type.
func TestSystem_Evolution_Convention(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	domain := system.DomainConfig{
		Evolutions: []system.EvolutionSpec{
			system.OnEvolution(
				system.OnEvolution(
					system.Evolve[EvoView]("evo_tasks"),
					"evo.created", EvoCreated{},
				),
				"evo.deleted", EvoDeleted{},
			).Done(),
		},
		Projections: []system.ProjectionDeclaration{
			// No .On() calls — inherits from Evolve[EvoView]
			system.Lookup[EvoView]("evo_lookup").Done(),
		},
	}

	deployment := system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			"primary": {Driver: "memory"},
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
			{Role: system.RoleProjections, Engine: "primary"},
		},
	}

	sys, err := system.New(ctx, domain, deployment)
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}

	defer sys.Close()

	store := sys.MetaEngine()
	_ = store.Apply(ctx, "evo.created", EvoCreated{ID: "e1", Title: "Task", Status: "open"})

	v, err := system.Get[EvoView](ctx, sys, "evo_lookup", "e1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if v.Title != "Task" || v.Status != "open" {
		t.Fatalf("unexpected view: %+v", v)
	}
}

// TestSystem_Evolution_ExplicitFold verifies explicit fold functions on Evolutions.
func TestSystem_Evolution_ExplicitFold(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	domain := system.DomainConfig{
		Evolutions: []system.EvolutionSpec{
			system.OnEvolution(
				system.OnEvolution(
					system.OnEvolution(
						system.Evolve[EvoView]("evo_tasks2"),
						"evo.created", EvoCreated{},
					),
					"evo.started", EvoStarted{},
					func(e EvoStarted, v *EvoView) { v.Status = "active" },
				),
				"evo.completed", EvoCompleted{},
				func(e EvoCompleted, v *EvoView) { v.Status = "done" },
			).Done(),
		},
		Projections: []system.ProjectionDeclaration{
			system.Lookup[EvoView]("evo_lookup2").Done(),
		},
	}

	deployment := system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			"primary": {Driver: "memory"},
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
			{Role: system.RoleProjections, Engine: "primary"},
		},
	}

	sys, err := system.New(ctx, domain, deployment)
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}

	defer sys.Close()

	store := sys.MetaEngine()
	_ = store.Apply(ctx, "evo.created", EvoCreated{ID: "e2", Title: "Task2", Status: "open"})
	_ = store.Apply(ctx, "evo.started", EvoStarted{ID: "e2"})
	_ = store.Apply(ctx, "evo.completed", EvoCompleted{ID: "e2"})

	v, err := system.Get[EvoView](ctx, sys, "evo_lookup2", "e2")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if v.Status != "done" {
		t.Fatalf("expected status 'done', got %q", v.Status)
	}

	if v.Title != "Task2" {
		t.Fatalf("expected title 'Task2', got %q", v.Title)
	}
}

// TestSystem_Evolution_QuerySet verifies a QuerySet inherits folds from an Evolution.
func TestSystem_Evolution_QuerySet(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	domain := system.DomainConfig{
		Evolutions: []system.EvolutionSpec{
			system.OnEvolution(
				system.Evolve[EvoView]("evo_tasks3"),
				"evo.created", EvoCreated{},
			).Done(),
		},
		Projections: []system.ProjectionDeclaration{
			system.QuerySet[EvoView]("evo_set").
				Filterable("status").
				Done(),
		},
	}

	deployment := system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			"primary": {Driver: "memory"},
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
			{Role: system.RoleProjections, Engine: "primary"},
		},
	}

	sys, err := system.New(ctx, domain, deployment)
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}

	defer sys.Close()

	store := sys.MetaEngine()
	_ = store.Apply(ctx, "evo.created", EvoCreated{ID: "1", Title: "A", Status: "open"})
	_ = store.Apply(ctx, "evo.created", EvoCreated{ID: "2", Title: "B", Status: "done"})

	results, err := system.Find[EvoView](ctx, sys, "evo_set",
		system.Where("status", "open"),
	)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}

	if len(results) != 1 || results[0].Title != "A" {
		t.Fatalf("expected 1 result with title 'A', got %+v", results)
	}
}

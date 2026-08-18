package system_test

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// ── Option test types ──

type OptCreated struct {
	Other string
	Title string
}

type OptStarted struct {
	Other string
}

type OptView struct {
	Other  string
	Title  string
	Status string
}

// TestSystem_Evolution_CustomKeyField verifies EvolveKey: the projection is
// keyed by "Other" instead of the default "ID". Internal() is exercised in
// the same builder to lock its current semantics (recorded, not enforced).
func TestSystem_Evolution_CustomKeyField(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	domain := system.DomainConfig{
		Evolutions: []system.EvolutionSpec{
			system.OnEvolution(
				system.Evolve[OptView](
					"opt_tasks",
					system.EvolveKey("Other"),
					system.Internal(),
				),
				"opt.created", OptCreated{},
			).Done(),
		},
		Projections: []system.ProjectionDeclaration{
			system.Lookup[OptView]("opt_lookup").Done(),
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
	mustApply(t, store, "opt.created", OptCreated{Other: "k1", Title: "Keyed"})

	v, err := system.Get[OptView](ctx, sys, "opt_lookup", "k1")
	if err != nil {
		t.Fatalf("Get by custom key: %v", err)
	}

	if v.Title != "Keyed" {
		t.Fatalf("expected title 'Keyed', got %q", v.Title)
	}
}

// TestSystem_Evolution_ExplicitFoldReifiesJSON drives the explicit-fold
// reify path through a SQLite engine: stored state decodes back as
// map[string]any, so the fold must re-marshal it into the result type
// (reifyTo's JSON branch) before mutating.
func TestSystem_Evolution_ExplicitFoldReifiesJSON(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	domain := system.DomainConfig{
		Evolutions: []system.EvolutionSpec{
			system.OnEvolution(
				system.OnEvolution(
					system.Evolve[OptView]("opt_tasks2", system.EvolveKey("Other")),
					"opt.created", OptCreated{},
				),
				"opt.started", OptStarted{},
				func(e OptStarted, v *OptView) { v.Status = "active" },
			).Done(),
		},
		Projections: []system.ProjectionDeclaration{
			system.Lookup[OptView]("opt_lookup2").Done(),
		},
	}

	deployment := system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			"primary": {Driver: "sqlite"},
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
	mustApply(t, store, "opt.created", OptCreated{Other: "k2", Title: "SqliteKeyed"})
	mustApply(t, store, "opt.started", OptStarted{Other: "k2"})

	v, err := system.Get[OptView](ctx, sys, "opt_lookup2", "k2")
	if err != nil {
		t.Fatalf("Get after JSON reify: %v", err)
	}

	if v.Status != "active" {
		t.Fatalf("expected status 'active' (explicit fold over reified state), got %q", v.Status)
	}

	if v.Title != "SqliteKeyed" {
		t.Fatalf("reified state lost Title: got %q, want %q", v.Title, "SqliteKeyed")
	}
}

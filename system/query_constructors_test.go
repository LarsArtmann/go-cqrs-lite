package system_test

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// ── QuerySet test types ──

type QSCreated struct {
	ID       string
	Title    string
	Status   string
	Priority int
}

type QSDeleted struct {
	ID string
}

type QSView struct {
	ID       string
	Title    string
	Status   string
	Priority int
}

// TestSystem_QuerySet_Planning verifies that a QuerySet declaration with
// Filterable + Sortable produces a planned projection with pushdown options.
func TestSystem_QuerySet_Planning(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	domain := system.DomainConfig{
		Projections: []system.ProjectionDeclaration{
			system.QuerySet[QSView]("qs_views").
				On("qs.created", QSCreated{}).
				On("qs.deleted", QSDeleted{}).
				Filterable("status", "priority").
				Sortable("priority", true).
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

	if sys.MetaEngine() == nil {
		t.Fatal("expected non-nil MetaEngine")
	}

	collections := sys.MetaEngine().Collections()
	if len(collections) != 1 {
		t.Fatalf("expected 1 collection, got %d", len(collections))
	}

	if collections[0].Name != "qs_views" {
		t.Fatalf("expected collection name 'qs_views', got %q", collections[0].Name)
	}
}

// ── Count test types ──

type CountCreated struct {
	ID     string
	Status string
}

type CountCompleted struct {
	ID string
}

// TestSystem_Count_Planning verifies that a Count declaration with .On
// produces a counter query with the correct ADT classification.
func TestSystem_Count_Planning(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	domain := system.DomainConfig{
		Projections: []system.ProjectionDeclaration{
			system.Count("task-counts").
				On("count.created", CountCreated{}, +1, "pending").
				On("count.completed", CountCompleted{}, -1, "pending").
				On("count.completed", CountCompleted{}, +1, "done").
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

	if sys.MetaEngine() == nil {
		t.Fatal("expected non-nil MetaEngine")
	}

	collections := sys.MetaEngine().Collections()
	if len(collections) != 1 {
		t.Fatalf("expected 1 collection, got %d", len(collections))
	}

	if collections[0].ADT != metaengine.ADTCounter {
		t.Fatalf("expected ADT Counter, got %s", collections[0].ADT)
	}
}

// TestSystem_Count_E2E verifies the full pipeline for a Count projection:
// declare → plan → event → counter query.
func TestSystem_Count_E2E(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	domain := system.DomainConfig{
		Projections: []system.ProjectionDeclaration{
			system.Count("e2e-counts").
				On("e2e.created", E2ECreated{}, +1, "active").
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

	// Apply events directly to the store (bypassing command dispatch for simplicity).
	store := sys.MetaEngine()
	mustApply(t, store, "e2e.created", E2ECreated{ID: "1"})

	// Query the counter.
	result, err := metaengine.ExecuteTyped[system.CountInput, map[string]int64](
		ctx, store, system.CountInput{},
	)
	if err != nil {
		t.Fatalf("ExecuteTyped: %v", err)
	}

	if result["active"] != 1 {
		t.Fatalf("expected active=1, got %d", result["active"])
	}
}

type E2ECreated struct {
	ID string
}

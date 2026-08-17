package system_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// ── Runtime API test types ──

type RuntimeCreated struct {
	ID       string
	Title    string
	Status   string
	Priority int
}

type RuntimeUpdated struct {
	ID     string
	Status string
}

type RuntimeDeleted struct {
	ID string
}

type RuntimeView struct {
	ID       string
	Title    string
	Status   string
	Priority int
}

// TestSystem_Runtime_Get verifies system.Get[R] returns the projected value
// after an event flows through the full pipeline.
func TestSystem_Runtime_Get(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	domain := system.DomainConfig{
		Projections: []system.ProjectionDeclaration{
			system.Lookup[RuntimeView]("rt_lookup").
				On("rt.created", RuntimeCreated{}).
				On("rt.updated", RuntimeUpdated{}).
				On("rt.deleted", RuntimeDeleted{}).
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

	// Apply events directly to the metaengine store.
	store := sys.MetaEngine()
	mustApply(t, store, "rt.created", RuntimeCreated{
		ID: "rt-1", Title: "Test", Status: "open", Priority: 5,
	})

	// Use system.Get to read it.
	v, err := system.Get[RuntimeView](ctx, sys, "rt_lookup", "rt-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if v.Title != "Test" || v.Status != "open" {
		t.Fatalf("unexpected view: %+v", v)
	}

	// Update and verify.
	mustApply(t, store, "rt.updated", RuntimeUpdated{ID: "rt-1", Status: "done"})
	v, err = system.Get[RuntimeView](ctx, sys, "rt_lookup", "rt-1")
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}

	if v.Status != "done" {
		t.Fatalf("expected status 'done', got %q", v.Status)
	}

	// Not found.
	_, err = system.Get[RuntimeView](ctx, sys, "rt_lookup", "nonexistent")
	if !errors.Is(err, system.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// TestSystem_Runtime_Find verifies system.Find[R] with filters and sorting.
func TestSystem_Runtime_Find(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	domain := system.DomainConfig{
		Projections: []system.ProjectionDeclaration{
			system.QuerySet[RuntimeView]("rt_set").
				On("rt.created", RuntimeCreated{}).
				Filterable("status").
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

	store := sys.MetaEngine()

	mustApply(t, store, "rt.created", RuntimeCreated{ID: "1", Title: "A", Status: "open", Priority: 1})
	mustApply(t, store, "rt.created", RuntimeCreated{ID: "2", Title: "B", Status: "done", Priority: 3})
	mustApply(t, store, "rt.created", RuntimeCreated{ID: "3", Title: "C", Status: "open", Priority: 5})

	// All results.
	all, err := system.Find[RuntimeView](ctx, sys, "rt_set")
	if err != nil {
		t.Fatalf("Find all: %v", err)
	}

	if len(all) != 3 {
		t.Fatalf("expected 3 results, got %d", len(all))
	}

	// Filter by status.
	open, err := system.Find[RuntimeView](ctx, sys, "rt_set",
		system.Where("status", "open"),
	)
	if err != nil {
		t.Fatalf("Find filtered: %v", err)
	}

	if len(open) != 2 {
		t.Fatalf("expected 2 open results, got %d", len(open))
	}

	// Limit.
	limited, err := system.Find[RuntimeView](ctx, sys, "rt_set",
		system.Limit(2),
	)
	if err != nil {
		t.Fatalf("Find limited: %v", err)
	}

	if len(limited) != 2 {
		t.Fatalf("expected 2 results with limit, got %d", len(limited))
	}
}

// TestSystem_Runtime_GetCount verifies system.GetCount on a Count projection.
func TestSystem_Runtime_GetCount(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	domain := system.DomainConfig{
		Projections: []system.ProjectionDeclaration{
			system.Count("rt_counts").
				On("rt.created", RuntimeCreated{}, +1, "active").
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
	mustApply(t, store, "rt.created", RuntimeCreated{ID: "1"})
	mustApply(t, store, "rt.created", RuntimeCreated{ID: "2"})

	counts, err := system.GetCount(ctx, sys, "rt_counts")
	if err != nil {
		t.Fatalf("GetCount: %v", err)
	}

	if counts["active"] != 2 {
		t.Fatalf("expected active=2, got %d", counts["active"])
	}
}

// TestSystem_Runtime_Get_NoProjections verifies Get returns an error when
// no projections are configured.
func TestSystem_Runtime_Get_NoProjections(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	_, err := system.New(ctx, system.DomainConfig{}, system.DeploymentConfig{
		Engines:   map[string]system.EngineConfig{"primary": {Driver: "memory"}},
		Instances: []system.InstanceConfig{{Role: system.RoleSourceOfTruth, Engine: "primary"}},
	})
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}

	// Use a minimal system with no projections.
	sys2, _ := system.New(ctx, system.DomainConfig{}, system.DeploymentConfig{
		Engines:   map[string]system.EngineConfig{"primary": {Driver: "memory"}},
		Instances: []system.InstanceConfig{{Role: system.RoleSourceOfTruth, Engine: "primary"}},
	})
	defer sys2.Close()

	_, err = system.Get[RuntimeView](ctx, sys2, "nonexistent", "x")
	if !errors.Is(err, system.ErrNoProjections) {
		t.Fatalf("expected ErrNoProjections, got %v", err)
	}
}

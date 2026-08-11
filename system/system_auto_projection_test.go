package system_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// ── Auto-projection domain types ──

type AutoProjCreated struct {
	ID     string
	Title  string
	Status string
}

type AutoProjUpdated struct {
	ID     string
	Title  string
	Status string
}

type AutoProjDeleted struct {
	ID string
}

type AutoProjView struct {
	ID     string
	Title  string
	Status string
}

// TestSystem_AutoProjection_MemoryEngine validates the full auto-projection
// pipeline: View[V,K](name).From(events...) → system.New() → command dispatch →
// projection processing → typed query.
func TestSystem_AutoProjection_MemoryEngine(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	domain := system.DomainConfig{
		Commands: func(sys *system.System) {
			system.RegisterDecider(sys, "AutoProj", TaskDecider)

			system.RegisterCommand[*command.BasicCommand, TaskState](sys, "autoproj.create",
				func(ctx context.Context, cmd *command.BasicCommand) system.Op[TaskState] {
					return system.Execute(ctx, cmd.StreamID(), "AutoProj",
						func(state TaskState, ver event.Version) ([]event.Event, error) {
							if state.Exists {
								return nil, errors.New("already exists")
							}

							return []event.Event{mustEvent(event.New(
								"autoproj.created",
								cmd.StreamID(),
								"AutoProj",
								ver+1,
								AutoProjCreated{
									ID:     "auto-1",
									Title:  "Auto-projection test",
									Status: "open",
								},
							))}, nil
						})
				})
		},
		Projections: []system.ProjectionDeclaration{
			system.Lookup[AutoProjView]("auto_views").
				On("autoproj.created", AutoProjCreated{}).
				On("autoproj.updated", AutoProjUpdated{}).
				On("autoproj.deleted", AutoProjDeleted{}).
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

	// Dispatch command BEFORE starting projections — host replays from journal.
	if err := sys.CommandDispatcher().
		Dispatch(ctx, newCmd("autoproj.create", id.NewStreamID())); err != nil {
		t.Fatalf("dispatch create: %v", err)
	}

	if err := sys.Start(ctx); err != nil {
		t.Fatalf("system.Start: %v", err)
	}

	// Wait for projection to process.
	deadline := time.Now().Add(5 * time.Second)

	for time.Now().Before(deadline) {
		for _, s := range sys.ProjectionHost().Status() {
			if s.Processed >= 1 && s.Errors == 0 {
				// Query the auto-projected view.
				result, err := metaengine.ExecuteTyped[system.LookupInput[string], AutoProjView](
					ctx, sys.MetaEngine(), system.LookupInput[string]{ID: "auto-1"},
				)
				if err != nil {
					t.Fatalf("ExecuteTyped: %v", err)
				}

				if result.Title != "Auto-projection test" {
					t.Fatalf("expected title 'Auto-projection test', got %q", result.Title)
				}

				if result.Status != "open" {
					t.Fatalf("expected status 'open', got %q", result.Status)
				}

				return // success
			}
		}

		time.Sleep(50 * time.Millisecond)
	}

	for _, s := range sys.ProjectionHost().Status() {
		if s.Errors > 0 {
			t.Fatalf("projection %q has %d errors", s.Name, s.Errors)
		}
	}

	t.Fatal("projection did not process event within timeout")
}

// TestSystem_AutoProjection_BackwardCompat verifies that raw metaengine.QueryDecl
// values still work alongside ProjectionSpec values (no breaking change).
func TestSystem_AutoProjection_BackwardCompat(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Mix: raw QueryDecl (wrapped) + auto-projection ProjectionSpec
	rawQuery := metaengine.Query[FindTask, TaskView](
		"raw_views",
		metaengine.OnRecordTyped(
			"raw.event",
			TaskCreated{},
			func(_ record.Record, e TaskCreated) (string, TaskView) {
				return e.Title, TaskView{Title: e.Title, Status: "manual"}
			},
		),
	)

	autoProj := system.Lookup[AutoProjView]("auto_views_bc").
		On("autoproj.created", AutoProjCreated{}).
		Done()

	domain := system.DomainConfig{
		Projections: []system.ProjectionDeclaration{
			system.RawQuery(rawQuery),
			autoProj,
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

	// Both projections should be planned.
	if sys.MetaEngine() == nil {
		t.Fatal("expected non-nil MetaEngine")
	}

	// Verify both collections exist.
	collections := sys.MetaEngine().Collections()
	if len(collections) < 2 {
		t.Fatalf("expected at least 2 collections, got %d", len(collections))
	}
}

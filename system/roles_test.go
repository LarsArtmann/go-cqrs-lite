package system_test

import (
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// TestSystem_RoleWiring_DedicatedCommandsQueries verifies that dedicated
// commands/queries instances bind their audit stores — including two roles
// sharing one engine (collections are namespaced, so role separation does not
// require engine separation).
func TestSystem_RoleWiring_DedicatedCommandsQueries(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	sys, err := system.New(ctx, system.DomainConfig{}, system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			"primary": {Driver: "memory"},
			"audit":   {Driver: "memory"},
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
			{Role: system.RoleCommands, Engine: "audit"},
			{Role: system.RoleQueries, Engine: "audit"},
		},
	})
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}

	defer sys.Close()

	if sys.CommandStore() == nil {
		t.Fatal("dedicated commands instance must bind CommandStore")
	}

	if sys.QueryStore() == nil {
		t.Fatal("dedicated queries instance must bind QueryStore")
	}

	// Round-trip one command through the dedicated store.
	ref := command.NewStreamRef(id.NewStreamID(), "Task")
	pc, err := command.NewPersistedCommand("task.create", ref, []byte(`{"title":"x"}`))
	if err != nil {
		t.Fatalf("NewPersistedCommand: %v", err)
	}

	if err := sys.CommandStore().Save(ctx, ref, pc); err != nil {
		t.Fatalf("CommandStore.Save: %v", err)
	}

	loaded, err := sys.CommandStore().Load(ctx, ref)
	if err != nil {
		t.Fatalf("CommandStore.Load: %v", err)
	}

	if len(loaded) != 1 || loaded[0].Type() != "task.create" {
		t.Fatalf("loaded commands = %+v, want one task.create", loaded)
	}
}

// TestSystem_RoleWiring_DedicatedSnapshots verifies a dedicated snapshots
// instance binds the snapshot store from its own engine (SQLite implements
// SnapshotBackend; the memory source-of-truth does not).
func TestSystem_RoleWiring_DedicatedSnapshots(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	sys, err := system.New(ctx, system.DomainConfig{}, system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			"primary": {Driver: "memory"},
			"snaps":   {Driver: "sqlite"},
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
			{Role: system.RoleSnapshots, Engine: "snaps"},
		},
	})
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}

	defer sys.Close()

	snapStore := sys.SnapshotStore()
	if snapStore == nil {
		t.Fatal("dedicated snapshots instance must bind SnapshotStore")
	}

	streamID := id.NewStreamID()

	if err := snapStore.SaveSnapshot(ctx, streamID, "Task", 3, []byte("state")); err != nil {
		t.Fatalf("SaveSnapshot: %v", err)
	}

	data, version, err := snapStore.LoadSnapshot(ctx, streamID, "Task")
	if err != nil {
		t.Fatalf("LoadSnapshot: %v", err)
	}

	if string(data) != "state" || version != 3 {
		t.Fatalf("snapshot = (%q, %d), want (state, 3)", data, version)
	}
}

// TestSystem_RoleWiring_DuplicateRole fails construction on two instances
// claiming the same dedicated role: the System holds one store per role.
func TestSystem_RoleWiring_DuplicateRole(t *testing.T) {
	t.Parallel()

	_, err := system.New(context.Background(), system.DomainConfig{}, system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			"primary": {Driver: "memory"},
			"other":   {Driver: "memory"},
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
			{Role: system.RoleCommands, Engine: "primary"},
			{Role: system.RoleCommands, Engine: "other"},
		},
	})
	if !errors.Is(err, system.ErrDuplicateInstanceRole) {
		t.Fatalf("duplicate commands role error = %v, want ErrDuplicateInstanceRole", err)
	}
}

// TestSystem_RoleWiring_SnapshotsBackendMissing fails construction when a
// dedicated snapshots instance names an engine without SnapshotBackend.
func TestSystem_RoleWiring_SnapshotsBackendMissing(t *testing.T) {
	t.Parallel()

	_, err := system.New(context.Background(), system.DomainConfig{}, system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{"primary": {Driver: "memory"}},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
			{Role: system.RoleSnapshots, Engine: "primary"},
		},
	})
	if !errors.Is(err, system.ErrNotSnapshotBackend) {
		t.Fatalf("snapshots on memory engine error = %v, want ErrNotSnapshotBackend", err)
	}
}

// TestSystem_RoleWiring_UnknownEngine verifies dedicated instances resolve
// their engine through the same validation path as source-of-truth.
func TestSystem_RoleWiring_UnknownEngine(t *testing.T) {
	t.Parallel()

	_, err := system.New(context.Background(), system.DomainConfig{}, system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{"primary": {Driver: "memory"}},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
			{Role: system.RoleCommands, Engine: "ghost"},
		},
	})
	if !errors.Is(err, system.ErrUnknownEngine) {
		t.Fatalf("commands on unknown engine error = %v, want ErrUnknownEngine", err)
	}
}

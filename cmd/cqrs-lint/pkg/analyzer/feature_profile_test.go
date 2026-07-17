package analyzer

import (
	"testing"
)

func TestDetectFeatures_LocalCLI(t *testing.T) {
	t.Parallel()

	ctx := BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import "github.com/larsartmann/go-cqrs-lite/event/v4"

func save(store event.Store) {
	store.Save(nil, ref, events)
}
`,
	})

	fp := DetectFeatures(ctx)

	if fp.HasServer {
		t.Errorf("local CLI should not have server, got HasServer=true")
	}
	if fp.CommandFlow != CommandFlowReadOnly {
		t.Errorf("local CLI without dispatcher should be read-only, got %s", fp.CommandFlow)
	}
	if fp.Store != StoreNone {
		t.Errorf("no store import should give StoreNone, got %s", fp.Store)
	}
	if fp.Tracing != TracingOff {
		t.Errorf("no otel should give TracingOff, got %s", fp.Tracing)
	}
}

func TestDetectFeatures_CommandFlow(t *testing.T) {
	t.Parallel()

	ctx := BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup() {
	d := dispatcher.NewDispatcher()
	d.Use(middleware)
	d.Dispatch(ctx, cmd)
}
`,
	})

	fp := DetectFeatures(ctx)

	if fp.CommandFlow != CommandFlowCommands {
		t.Errorf("Dispatch() calls should give CommandFlowCommands, got %s", fp.CommandFlow)
	}
}

func TestDetectFeatures_SyncDispatcher(t *testing.T) {
	t.Parallel()

	ctx := BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup() {
	d := dispatcher.NewDispatcher()
	d.Use(middleware)
}
`,
	})

	fp := DetectFeatures(ctx)

	if fp.CommandFlow != CommandFlowSync {
		t.Errorf(
			"dispatcher without Dispatch() should give CommandFlowSync, got %s",
			fp.CommandFlow,
		)
	}
}

func TestDetectFeatures_SoftDelete(t *testing.T) {
	t.Parallel()

	ctx := BuildContextFromSource(t, map[string]string{
		"events.go": `package main

func emitDelete() {
	event.New("user.deleted", id, "User", 1, payload)
}
`,
	})

	fp := DetectFeatures(ctx)

	if !fp.HasSoftDelete {
		t.Error("event name 'user.deleted' should detect HasSoftDelete=true")
	}
}

func TestDetectFeatures_NoSoftDelete(t *testing.T) {
	t.Parallel()

	ctx := BuildContextFromSource(t, map[string]string{
		"events.go": `package main

func emitCreate() {
	New("user.created", id, "User", 1, payload)
}
`,
	})

	fp := DetectFeatures(ctx)

	if fp.HasSoftDelete {
		t.Error("event name 'user.created' should not detect HasSoftDelete")
	}
}

func TestResolveFeatureProfile_ConfigOverridesDetect(t *testing.T) {
	t.Parallel()

	detected := FeatureProfile{
		Store:       StoreSQLite,
		CommandFlow: CommandFlowCommands,
		HasServer:   false,
		Tracing:     TracingOff,
	}

	postgres := StorePostgres
	cfg := ConfigFeatures{
		Store: &postgres,
	}

	resolved := ResolveFeatureProfile(cfg, PresetNone, detected)

	if resolved.Store != StorePostgres {
		t.Errorf("config store override should win, got %s", resolved.Store)
	}
	if resolved.CommandFlow != CommandFlowCommands {
		t.Errorf("non-overridden field should stay detected, got %s", resolved.CommandFlow)
	}
}

func TestResolveFeatureProfile_PresetThenOverride(t *testing.T) {
	t.Parallel()

	detected := FeatureProfile{
		Store:       StoreSQLite,
		CommandFlow: CommandFlowCommands,
		HasServer:   true,
		Tracing:     TracingOn,
	}

	// Preset local-cli sets Server=false, Tracing=off.
	// Explicit override sets Server=true.
	serverTrue := true
	cfg := ConfigFeatures{
		Server: &serverTrue,
	}

	resolved := ResolveFeatureProfile(cfg, PresetLocalCLI, detected)

	if !resolved.HasServer {
		t.Error("explicit Features.Server should override preset local-cli's Server=false")
	}
	if resolved.Tracing != TracingOff {
		t.Errorf("preset local-cli Tracing=off should apply, got %s", resolved.Tracing)
	}
}

func TestResolveFeatureProfile_PresetLocalCLI(t *testing.T) {
	t.Parallel()

	detected := FeatureProfile{
		HasServer: true,
		Tracing:   TracingOn,
	}

	resolved := ResolveFeatureProfile(ConfigFeatures{}, PresetLocalCLI, detected)

	if resolved.HasServer {
		t.Error("preset local-cli should force HasServer=false")
	}
	if resolved.Tracing != TracingOff {
		t.Errorf("preset local-cli should force Tracing=off, got %s", resolved.Tracing)
	}
}

func TestResolveFeatureProfile_PresetLibrary(t *testing.T) {
	t.Parallel()

	detected := FeatureProfile{
		CommandFlow: CommandFlowCommands,
		Snapshot:    SnapshotOn,
	}

	resolved := ResolveFeatureProfile(ConfigFeatures{}, PresetLibrary, detected)

	if resolved.CommandFlow != CommandFlowReadOnly {
		t.Errorf("preset library should force CommandFlow=read-only, got %s", resolved.CommandFlow)
	}
	if resolved.Snapshot != SnapshotOff {
		t.Errorf("preset library should force Snapshot=off, got %s", resolved.Snapshot)
	}
}

func TestFeatureProfile_String(t *testing.T) {
	t.Parallel()

	fp := FeatureProfile{
		Store:       StoreSQLite,
		CommandFlow: CommandFlowCommands,
		HasServer:   true,
		Tracing:     TracingOn,
	}

	s := fp.String()
	if s == "" {
		t.Error("String() should not be empty")
	}
}

func TestResolvePreset_Unknown(t *testing.T) {
	t.Parallel()

	cf := ResolvePreset("nonexistent")
	if cf.Server != nil || cf.Tracing != nil {
		t.Error("unknown preset should return empty ConfigFeatures")
	}
}

func TestResolvePreset_None(t *testing.T) {
	t.Parallel()

	cf := ResolvePreset(PresetNone)
	if cf.Server != nil || cf.Tracing != nil {
		t.Error("PresetNone should return empty ConfigFeatures")
	}
}

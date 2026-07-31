package analyzer

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
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

// TestToConfigFeatures_OmitsUnknownFields is the regression test for the
// doctor trailing-comma bug. When tracing and snapshot are both unknown, the
// old hand-formatted JSON emitted a trailing comma after "soft-delete". The
// fix builds ConfigFeatures via ToConfigFeatures and serializes with
// encoding/json, which can never produce a trailing comma.
func TestToConfigFeatures_OmitsUnknownFields(t *testing.T) {
	t.Parallel()

	fp := FeatureProfile{
		HasServer:     false,
		HasSoftDelete: false,
	}

	cf := fp.ToConfigFeatures()

	if cf.Store != nil {
		t.Errorf("unknown Store should be omitted, got %v", *cf.Store)
	}
	if cf.CommandFlow != nil {
		t.Errorf("unknown CommandFlow should be omitted, got %v", *cf.CommandFlow)
	}
	if cf.Tracing != nil {
		t.Errorf("unknown Tracing should be omitted, got %v", *cf.Tracing)
	}
	if cf.Snapshot != nil {
		t.Errorf("unknown Snapshot should be omitted, got %v", *cf.Snapshot)
	}
	if cf.Server == nil || *cf.Server != false {
		t.Error("server is a meaningful bool and should always be included")
	}
	if cf.SoftDelete == nil || *cf.SoftDelete != false {
		t.Error("soft-delete is a meaningful bool and should always be included")
	}
}

func TestToConfigFeatures_IncludesKnownFields(t *testing.T) {
	t.Parallel()

	fp := FeatureProfile{
		Store:         StorePostgres,
		CommandFlow:   CommandFlowCommands,
		HasServer:     true,
		HasSoftDelete: true,
		Tracing:       TracingOn,
		Snapshot:      SnapshotOn,
	}

	cf := fp.ToConfigFeatures()

	if cf.Store == nil || *cf.Store != StorePostgres {
		t.Error("detected Store should be included")
	}
	if cf.CommandFlow == nil || *cf.CommandFlow != CommandFlowCommands {
		t.Error("detected CommandFlow should be included")
	}
	if cf.Tracing == nil || *cf.Tracing != TracingOn {
		t.Error("detected Tracing should be included")
	}
	if cf.Snapshot == nil || *cf.Snapshot != SnapshotOn {
		t.Error("detected Snapshot should be included")
	}
}

// TestToConfigFeatures_JSONAlwaysValid guards the doctor output across every
// realistic profile shape: the marshaled suggestion must parse back as JSON and
// must never contain a trailing comma.
func TestToConfigFeatures_JSONAlwaysValid(t *testing.T) {
	t.Parallel()

	profiles := map[string]FeatureProfile{
		"all-unknown":           {},
		"server-only":           {HasServer: true, HasSoftDelete: false},
		"tracing-only-unknown":  {HasServer: false, HasSoftDelete: false, Snapshot: SnapshotOn},
		"snapshot-only-unknown": {HasServer: false, HasSoftDelete: false, Tracing: TracingOn},
		"fully-detected": {
			Store: StoreSQLite, CommandFlow: CommandFlowCommands,
			HasServer: true, HasSoftDelete: true, Tracing: TracingOn, Snapshot: SnapshotOn,
		},
		"store-none": {Store: StoreNone, HasServer: false, HasSoftDelete: false},
	}

	for name, fp := range profiles {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			raw, err := json.Marshal(
				map[string]ConfigFeatures{
					"features": fp.ToConfigFeatures(),
				},
				jsontext.WithIndentPrefix(""),
				jsontext.WithIndent("  "),
			)
			if err != nil {
				t.Fatalf("marshal failed: %v", err)
			}

			if strings.Contains(string(raw), ",\n  }") || strings.Contains(string(raw), ",\n}") {
				t.Errorf("JSON contains a trailing comma:\n%s", raw)
			}

			var back map[string]ConfigFeatures
			if err := json.Unmarshal(raw, &back); err != nil {
				t.Fatalf("doctor output is not valid JSON (%v):\n%s", err, raw)
			}
		})
	}
}

// --- DetectFeatures precision tests ---

func TestDetectFeatures_HTTPServer(t *testing.T) {
	t.Parallel()

	ctx := BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import "net/http"

func serve() {
	http.ListenAndServe(":8080", nil)
}
`,
	})

	fp := DetectFeatures(ctx)
	if !fp.HasServer {
		t.Error("http.ListenAndServe should detect HasServer=true")
	}
}

func TestDetectFeatures_GRPCServer(t *testing.T) {
	t.Parallel()

	ctx := BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func serve() {
	srv := grpc.NewServer()
	_ = srv
}
`,
	})

	fp := DetectFeatures(ctx)
	if !fp.HasServer {
		t.Error("grpc.NewServer should detect HasServer=true")
	}
}

func TestDetectFeatures_Tracing(t *testing.T) {
	t.Parallel()

	ctx := BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func setup(bus *Bus) {
	bus.Use(middleware.EventTracing(tracer))
}
`,
	})
	ctx.Packages = []*packages.Package{
		{
			PkgPath: "example.com/app",
			Imports: map[string]*packages.Package{
				"go.opentelemetry.io/otel": {PkgPath: "go.opentelemetry.io/otel"},
			},
		},
	}

	fp := DetectFeatures(ctx)
	if fp.Tracing != TracingOn {
		t.Errorf(
			"otel import + EventTracing middleware should detect Tracing=on, got %s",
			fp.Tracing,
		)
	}
}

func TestDetectFeatures_Snapshot(t *testing.T) {
	t.Parallel()

	ctx := BuildContextFromSource(t, map[string]string{
		"repo.go": `package main

func setup() {
	repo := decider.NewRepository(store, bus, d, decider.WithSnapshotStore(snap))
	_ = repo
}
`,
	})

	fp := DetectFeatures(ctx)
	if fp.Snapshot != SnapshotOn {
		t.Errorf("WithSnapshotStore call should detect Snapshot=on, got %s", fp.Snapshot)
	}
}

func TestDetectFeatures_MySQLStore(t *testing.T) {
	t.Parallel()

	ctx := BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func setup() {
	_ = mysql.New(dsn)
}
`,
	})
	ctx.Packages = []*packages.Package{
		{
			PkgPath: "example.com/app",
			Imports: map[string]*packages.Package{
				"github.com/larsartmann/go-cqrs-lite/stack/mysql/v4": {
					PkgPath: "github.com/larsartmann/go-cqrs-lite/stack/mysql/v4",
				},
			},
		},
	}

	fp := DetectFeatures(ctx)
	if fp.Store != StoreMySQL {
		t.Errorf("stack/mysql import should detect StoreMySQL, got %s", fp.Store)
	}
}

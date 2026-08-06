package analyzer

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"slices"
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

func TestPresetLibrary_DisablesAdoptionAndSecurityFalsePositives(t *testing.T) {
	t.Parallel()

	// A library/SDK defines types and infrastructure but cannot force its
	// consumers to adopt catalog docs, encryption, signing, or relational
	// projections. These rules are inherent false-positives for library code
	// and must be in the preset's disable list so consumers don't have to
	// suppress them one-by-one.
	def := PresetDefinitions[PresetLibrary]

	wantDisabled := []string{
		"E003", "E016", // domain-package mixing
		"F002", "F006", "F010", "F011", // adoption coaching (consumer's job)
		"F015", "F022", // metaengine coaching (consumer's deployment choice)
		"S002", "S003", // security middleware (consumer wires it)
	}
	for _, rule := range wantDisabled {
		if !slices.Contains(def.Rules.Disable, rule) {
			t.Errorf("library preset should disable %s (inherent library false-positive)", rule)
		}
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

func TestDetectFeatures_GinImportDetectsServer(t *testing.T) {
	t.Parallel()

	ctx := BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import "github.com/gin-gonic/gin"

func setup() {
	engine := gin.New()
	_ = engine
}
`,
	})

	fp := DetectFeatures(ctx)
	if !fp.HasServer {
		t.Error("gin-gonic/gin import should detect HasServer=true")
	}
}

func TestDetectFeatures_HttpServerListenAndServe(t *testing.T) {
	t.Parallel()

	ctx := BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import "net/http"

func serve() {
	srv := &http.Server{Addr: ":8080", Handler: nil}
	_ = srv.ListenAndServe()
}
`,
	})

	fp := DetectFeatures(ctx)
	if !fp.HasServer {
		t.Error("srv.ListenAndServe() method call should detect HasServer=true")
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

func TestDetectFeatures_StorageNewSQLiteEventStore(t *testing.T) {
	t.Parallel()

	ctx := BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup(db *sql.DB) {
	store, _ := storage.NewSQLiteEventStore(db)
	_ = store
}
`,
	})
	ctx.Packages = []*packages.Package{
		{
			PkgPath: "example.com/app",
			Imports: map[string]*packages.Package{
				"github.com/larsartmann/go-cqrs-lite/storage/v4": {
					PkgPath: "github.com/larsartmann/go-cqrs-lite/storage/v4",
				},
			},
		},
	}

	fp := DetectFeatures(ctx)
	if fp.Store != StoreSQLite {
		t.Errorf(
			"storage.NewSQLiteEventStore should refine StoreCustom→StoreSQLite, got %s",
			fp.Store,
		)
	}
}

func TestDetectFeatures_ServerLocalListenAndServeOnly(t *testing.T) {
	t.Parallel()

	ctx := BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func main() {
	http.ListenAndServe(":8080", nil)
}
`,
	})

	fp := DetectFeatures(ctx)
	if !fp.HasServer {
		t.Fatal("ListenAndServe should detect HasServer")
	}
	if !fp.ServerLocal {
		t.Error("ListenAndServe without TLS/Shutdown/health should detect ServerLocal")
	}
}

func TestDetectFeatures_NotServerLocalWithShutdown(t *testing.T) {
	t.Parallel()

	ctx := BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func main() {
	srv := &http.Server{}
	srv.ListenAndServe()
	srv.Shutdown(nil)
}
`,
	})

	fp := DetectFeatures(ctx)
	if !fp.HasServer {
		t.Fatal("ListenAndServe should detect HasServer")
	}
	if fp.ServerLocal {
		t.Error("Server with Shutdown should NOT be Server-local")
	}
}

func TestDetectFeatures_NotServerLocalWithHealthRoute(t *testing.T) {
	t.Parallel()

	ctx := BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func main() {
	http.HandleFunc("/healthz", handler)
	http.ListenAndServe(":8080", nil)
}
`,
	})

	fp := DetectFeatures(ctx)
	if !fp.HasServer {
		t.Fatal("ListenAndServe should detect HasServer")
	}
	if fp.ServerLocal {
		t.Error("Server with /healthz route should NOT be server-local")
	}
}

// --- Phase 2: ConfigFeatures override tests ---

func TestResolveFeatureProfile_TransportOverride(t *testing.T) {
	t.Parallel()

	detected := FeatureProfile{
		HasTransport: true,
	}

	cfg := ConfigFeatures{
		Transport: new(false),
	}

	resolved := ResolveFeatureProfile(cfg, PresetNone, detected)
	if resolved.HasTransport {
		t.Error("ConfigFeatures.Transport=false should override detected HasTransport=true")
	}
}

func TestResolveFeatureProfile_TransportOverrideTrue(t *testing.T) {
	t.Parallel()

	detected := FeatureProfile{
		HasTransport: false,
	}

	cfg := ConfigFeatures{
		Transport: new(true),
	}

	resolved := ResolveFeatureProfile(cfg, PresetNone, detected)
	if !resolved.HasTransport {
		t.Error("ConfigFeatures.Transport=true should override detected HasTransport=false")
	}
}

func TestResolveFeatureProfile_ServerLocalOverride(t *testing.T) {
	t.Parallel()

	detected := FeatureProfile{
		ServerLocal: false,
	}

	cfg := ConfigFeatures{
		ServerLocal: new(true),
	}

	resolved := ResolveFeatureProfile(cfg, PresetNone, detected)
	if !resolved.ServerLocal {
		t.Error("ConfigFeatures.ServerLocal=true should override detected ServerLocal=false")
	}
}

func TestResolveFeatureProfile_ServerLocalOverrideFalse(t *testing.T) {
	t.Parallel()

	detected := FeatureProfile{
		ServerLocal: true,
	}

	cfg := ConfigFeatures{
		ServerLocal: new(false),
	}

	resolved := ResolveFeatureProfile(cfg, PresetNone, detected)
	if resolved.ServerLocal {
		t.Error("ConfigFeatures.ServerLocal=false should override detected ServerLocal=true")
	}
}

func TestToConfigFeatures_RoundTrip_Transport(t *testing.T) {
	t.Parallel()

	original := FeatureProfile{
		HasTransport: true,
		HasServer:    true,
	}

	cf := original.ToConfigFeatures()
	resolved := ResolveFeatureProfile(cf, PresetNone, FeatureProfile{})

	if !resolved.HasTransport {
		t.Error("round-trip detect→config→resolve should preserve HasTransport=true")
	}
}

func TestToConfigFeatures_RoundTrip_ServerLocal(t *testing.T) {
	t.Parallel()

	original := FeatureProfile{
		ServerLocal: true,
		HasServer:   true,
	}

	cf := original.ToConfigFeatures()
	resolved := ResolveFeatureProfile(cf, PresetNone, FeatureProfile{})

	if !resolved.ServerLocal {
		t.Error("round-trip detect→config→resolve should preserve ServerLocal=true")
	}
}

// --- Phase 2: TLS detection precision tests ---

func TestDetectFeatures_TLSListenDetectsTLS(t *testing.T) {
	t.Parallel()

	ctx := BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func main() {
	_ = tls.Listen("tcp", ":443", config)
}
`,
	})

	fp := DetectFeatures(ctx)
	if !fp.HasServer {
		t.Fatal("tls.Listen should detect HasServer")
	}
	if fp.ServerLocal {
		t.Error("tls.Listen should detect TLS → not ServerLocal")
	}
}

func TestDetectFeatures_NetListenNotTLS(t *testing.T) {
	t.Parallel()

	ctx := BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func main() {
	lis, _ := net.Listen("tcp", ":8080")
	_ = lis
}
`,
	})

	fp := DetectFeatures(ctx)
	if !fp.HasServer {
		t.Fatal("net.Listen should detect HasServer")
	}
	if !fp.ServerLocal {
		t.Error("net.Listen without TLS/Shutdown/health should be ServerLocal")
	}
}

func TestDetectFeatures_ListenAndServeTLSDetectsTLS(t *testing.T) {
	t.Parallel()

	ctx := BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func main() {
	http.ListenAndServeTLS(":443", "cert.pem", "key.pem", nil)
}
`,
	})

	fp := DetectFeatures(ctx)
	if !fp.HasServer {
		t.Fatal("ListenAndServeTLS should detect HasServer")
	}
	if fp.ServerLocal {
		t.Error("ListenAndServeTLS should detect TLS → not ServerLocal")
	}
}

func TestPresetDefinitions_AllPresetsHaveFeatures(t *testing.T) {
	t.Parallel()

	for name, def := range PresetDefinitions {
		t.Run(string(name), func(t *testing.T) {
			t.Parallel()

			if def.Features.Server == nil && def.Features.CommandFlow == nil &&
				def.Features.Tracing == nil && def.Features.Snapshot == nil &&
				def.Features.SoftDelete == nil && def.Features.Store == nil &&
				def.Features.Domain == nil && def.Features.Transport == nil {
				t.Errorf("preset %q has no feature overrides at all", name)
			}
		})
	}
}

func TestResolvePresetDefinition_BackwardCompatWithResolvePreset(t *testing.T) {
	t.Parallel()

	for _, name := range ValidPresetNames() {
		def := ResolvePresetDefinition(ConfigPreset(name))
		features := ResolvePreset(ConfigPreset(name))

		if def.Features.Server != features.Server {
			t.Errorf(
				"preset %q: ResolvePresetDefinition().Features.Server != ResolvePreset().Server",
				name,
			)
		}
	}
}

func TestIsKnownPreset(t *testing.T) {
	t.Parallel()

	if !IsKnownPreset(PresetNone) {
		t.Error("PresetNone should be known")
	}
	if !IsKnownPreset(PresetLocalCLI) {
		t.Error("PresetLocalCLI should be known")
	}
	if IsKnownPreset(ConfigPreset("server")) {
		t.Error("'server' should NOT be known (stale init preset)")
	}
	if IsKnownPreset(ConfigPreset("full-stack")) {
		t.Error("'full-stack' should NOT be known (stale init preset)")
	}
}

func TestValidPresetNames_ContainsAllFourPresets(t *testing.T) {
	t.Parallel()

	names := ValidPresetNames()
	expected := []string{"library", "local-cli", "production", "read-only"}
	if len(names) != len(expected) {
		t.Fatalf("expected %d preset names, got %d: %v", len(expected), len(names), names)
	}
	for i, want := range expected {
		if names[i] != want {
			t.Errorf("ValidPresetNames()[%d] = %q, want %q", i, names[i], want)
		}
	}
}

// --- AllXxxKinds coverage meta-tests ---
// These tests prevent drift: if a new constant is added to a Kind const block
// but the corresponding All*Kinds() function isn't updated, the explain
// command will silently miss the new value. Each test hardcodes every known
// constant (same source as the const block) and asserts each non-Unknown
// constant appears in the All*Kinds() result.

func TestAllStoreKindsCoversEveryConstant(t *testing.T) {
	t.Parallel()

	allConstants := []StoreKind{
		StoreUnknown, StoreSQLite, StorePostgres, StoreMySQL,
		StorePebble, StoreMemory, StoreTurso, StoreDuckDB,
		StoreCustom, StoreNone,
	}

	got := AllStoreKinds()
	gotSet := make(map[StoreKind]bool, len(got))
	for _, k := range got {
		gotSet[k] = true
	}

	for _, c := range allConstants {
		if c == StoreUnknown {
			continue
		}
		if !gotSet[c] {
			t.Errorf("StoreKind constant %q is missing from AllStoreKinds()", c)
		}
	}

	for _, k := range got {
		if k == StoreUnknown {
			t.Error("AllStoreKinds() should not include StoreUnknown")
		}
	}
}

func TestAllCommandFlowKindsCoversEveryConstant(t *testing.T) {
	t.Parallel()

	allConstants := []CommandFlowKind{
		CommandFlowUnknown, CommandFlowReadOnly, CommandFlowSync, CommandFlowCommands,
	}

	got := AllCommandFlowKinds()
	gotSet := make(map[CommandFlowKind]bool, len(got))
	for _, k := range got {
		gotSet[k] = true
	}

	for _, c := range allConstants {
		if c == CommandFlowUnknown {
			continue
		}
		if !gotSet[c] {
			t.Errorf("CommandFlowKind constant %q is missing from AllCommandFlowKinds()", c)
		}
	}
}

func TestAllTracingKindsCoversEveryConstant(t *testing.T) {
	t.Parallel()

	allConstants := []TracingKind{TracingUnknown, TracingOff, TracingOn}

	got := AllTracingKinds()
	gotSet := make(map[TracingKind]bool, len(got))
	for _, k := range got {
		gotSet[k] = true
	}

	for _, c := range allConstants {
		if c == TracingUnknown {
			continue
		}
		if !gotSet[c] {
			t.Errorf("TracingKind constant %q is missing from AllTracingKinds()", c)
		}
	}
}

func TestAllSnapshotKindsCoversEveryConstant(t *testing.T) {
	t.Parallel()

	allConstants := []SnapshotKind{SnapshotUnknown, SnapshotOff, SnapshotOn}

	got := AllSnapshotKinds()
	gotSet := make(map[SnapshotKind]bool, len(got))
	for _, k := range got {
		gotSet[k] = true
	}

	for _, c := range allConstants {
		if c == SnapshotUnknown {
			continue
		}
		if !gotSet[c] {
			t.Errorf("SnapshotKind constant %q is missing from AllSnapshotKinds()", c)
		}
	}
}

func TestAllDomainKindsCoversEveryConstant(t *testing.T) {
	t.Parallel()

	allConstants := []DomainKind{
		DomainUnknown, DomainFinancial, DomainInternal, DomainSecurity,
	}

	got := AllDomainKinds()
	gotSet := make(map[DomainKind]bool, len(got))
	for _, k := range got {
		gotSet[k] = true
	}

	for _, c := range allConstants {
		if c == DomainUnknown {
			continue
		}
		if !gotSet[c] {
			t.Errorf("DomainKind constant %q is missing from AllDomainKinds()", c)
		}
	}
}

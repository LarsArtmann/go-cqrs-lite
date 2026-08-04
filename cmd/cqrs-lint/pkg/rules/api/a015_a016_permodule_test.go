package api

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// TestA015_PerModuleProfileSuppressesNonServerModule verifies that A015
// evaluates HasServer per-module: a global mutable cache in a library module
// (no server) must NOT fire when an example sub-module has a server.
func TestA015_PerModuleProfileSuppressesNonServerModule(t *testing.T) {
	t.Parallel()

	source := `package test

var globalCache = make(map[string]string)

func update(key, val string) {
	globalCache[key] = val
}
`

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"/repo/lib/state.go": source,
	})

	ctx.FeatureProfiles = map[string]analyzer.FeatureProfile{
		"/repo/lib":          {HasServer: false},
		"/repo/examples/app": {HasServer: true},
	}
	ctx.FeatureProfile = analyzer.FeatureProfile{HasServer: true}

	det := NewA015Detector(ctx)
	findings, err := det.Detect(context.Background())
	if err != nil {
		t.Fatalf("A015 detect: %v", err)
	}

	if len(findings) != 0 {
		t.Errorf("expected 0 findings for library module (no server), got %d", len(findings))
	}
}

// TestA015_PerModuleProfileFiresForServerModule verifies that A015 still
// fires when the global mutable IS in a module that has a server.
func TestA015_PerModuleProfileFiresForServerModule(t *testing.T) {
	t.Parallel()

	source := `package test

var globalCache = make(map[string]string)

func update(key, val string) {
	globalCache[key] = val
}
`

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"/repo/examples/app/state.go": source,
	})

	ctx.FeatureProfiles = map[string]analyzer.FeatureProfile{
		"/repo/lib":          {HasServer: false},
		"/repo/examples/app": {HasServer: true},
	}
	ctx.FeatureProfile = analyzer.FeatureProfile{HasServer: true}

	det := NewA015Detector(ctx)
	findings, err := det.Detect(context.Background())
	if err != nil {
		t.Fatalf("A015 detect: %v", err)
	}

	if len(findings) != 1 {
		t.Errorf("expected 1 finding for server module, got %d", len(findings))
	}
}

// TestA015_SingleModuleFallback verifies backward compatibility: a
// single-module project falls back to the primary profile.
func TestA015_SingleModuleFallback(t *testing.T) {
	t.Parallel()

	source := `package test

var globalCache = make(map[string]string)

func update(key, val string) {
	globalCache[key] = val
}
`

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"state.go": source,
	})

	ctx.FeatureProfile = analyzer.FeatureProfile{HasServer: true}

	det := NewA015Detector(ctx)
	findings, err := det.Detect(context.Background())
	if err != nil {
		t.Fatalf("A015 detect: %v", err)
	}

	if len(findings) != 1 {
		t.Errorf("expected 1 finding for single-module server project, got %d", len(findings))
	}
}

// TestA016_PerModuleProfileSuppressesReadOnlyModule verifies that A016
// evaluates CommandFlow per-module: a dispatcher in a read-only library module
// must NOT fire when an example sub-module has command flow.
func TestA016_PerModuleProfileSuppressesReadOnlyModule(t *testing.T) {
	t.Parallel()

	source := `package test

func setup() {
	d := dispatcher.NewDispatcher()
	d.Use(loggingMiddleware)
	d.Dispatch(ctx, cmd)
}
`

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"/repo/lib/setup.go": source,
	})

	ctx.FeatureProfiles = map[string]analyzer.FeatureProfile{
		"/repo/lib":          {CommandFlow: analyzer.CommandFlowReadOnly},
		"/repo/examples/app": {CommandFlow: analyzer.CommandFlowCommands},
	}
	ctx.FeatureProfile = analyzer.FeatureProfile{CommandFlow: analyzer.CommandFlowCommands}

	det := NewA016Detector(ctx)
	findings, err := det.Detect(context.Background())
	if err != nil {
		t.Fatalf("A016 detect: %v", err)
	}

	if len(findings) != 0 {
		t.Errorf("expected 0 findings for read-only module, got %d", len(findings))
	}
}

// TestA016_PerModuleProfileFiresForCommandFlowModule verifies that A016 still
// fires when the dispatcher IS in a module with command flow.
func TestA016_PerModuleProfileFiresForCommandFlowModule(t *testing.T) {
	t.Parallel()

	source := `package test

func setup() {
	d := dispatcher.NewDispatcher()
	d.Use(loggingMiddleware)
	d.Dispatch(ctx, cmd)
}
`

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"/repo/examples/app/setup.go": source,
	})

	ctx.FeatureProfiles = map[string]analyzer.FeatureProfile{
		"/repo/lib":          {CommandFlow: analyzer.CommandFlowReadOnly},
		"/repo/examples/app": {CommandFlow: analyzer.CommandFlowCommands},
	}
	ctx.FeatureProfile = analyzer.FeatureProfile{CommandFlow: analyzer.CommandFlowCommands}

	det := NewA016Detector(ctx)
	findings, err := det.Detect(context.Background())
	if err != nil {
		t.Fatalf("A016 detect: %v", err)
	}

	if len(findings) != 1 {
		t.Errorf("expected 1 finding for command-flow module, got %d", len(findings))
	}
}

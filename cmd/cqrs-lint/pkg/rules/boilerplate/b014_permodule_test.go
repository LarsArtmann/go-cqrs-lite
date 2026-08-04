package boilerplate

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// TestB014_PerModuleProfileSuppressesNonServerModule verifies that B014
// evaluates HasServer per-module: a bus.Use() call in a library module
// (no server) must NOT fire when an example sub-module has a server.
func TestB014_PerModuleProfileSuppressesNonServerModule(t *testing.T) {
	t.Parallel()

	source := `package test

func setup(bus *EventBus) {
	bus.Use(loggingMiddleware)
	bus.UsePublish(retryMiddleware)
}
`

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"/repo/lib/setup.go": source,
	})

	ctx.FeatureProfiles = map[string]analyzer.FeatureProfile{
		"/repo/lib":          {HasServer: false},
		"/repo/examples/app": {HasServer: true},
	}
	ctx.FeatureProfile = analyzer.FeatureProfile{HasServer: true}

	det := NewB014Detector(ctx)
	findings, err := det.Detect(context.Background())
	if err != nil {
		t.Fatalf("B014 detect: %v", err)
	}

	if len(findings) != 0 {
		t.Errorf("expected 0 findings for library module (no server), got %d", len(findings))
	}
}

// TestB014_PerModuleProfileFiresForServerModule verifies that B014 still
// fires when the middleware setup IS in a module that has a server.
func TestB014_PerModuleProfileFiresForServerModule(t *testing.T) {
	t.Parallel()

	source := `package test

func setup(bus *EventBus) {
	bus.Use(loggingMiddleware)
	bus.UsePublish(retryMiddleware)
}
`

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"/repo/examples/app/setup.go": source,
	})

	ctx.FeatureProfiles = map[string]analyzer.FeatureProfile{
		"/repo/lib":          {HasServer: false},
		"/repo/examples/app": {HasServer: true},
	}
	ctx.FeatureProfile = analyzer.FeatureProfile{HasServer: true}

	det := NewB014Detector(ctx)
	findings, err := det.Detect(context.Background())
	if err != nil {
		t.Fatalf("B014 detect: %v", err)
	}

	if len(findings) != 1 {
		t.Errorf("expected 1 finding for server module, got %d", len(findings))
	}
}

// TestB014_SingleModuleFallback verifies backward compatibility: a
// single-module project falls back to the primary profile.
func TestB014_SingleModuleFallback(t *testing.T) {
	t.Parallel()

	source := `package test

func setup(bus *EventBus) {
	bus.Use(loggingMiddleware)
	bus.UsePublish(retryMiddleware)
}
`

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": source,
	})

	ctx.FeatureProfile = analyzer.FeatureProfile{HasServer: true}

	det := NewB014Detector(ctx)
	findings, err := det.Detect(context.Background())
	if err != nil {
		t.Fatalf("B014 detect: %v", err)
	}

	if len(findings) != 1 {
		t.Errorf("expected 1 finding for single-module server project, got %d", len(findings))
	}
}

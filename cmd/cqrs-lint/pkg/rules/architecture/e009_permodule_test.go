package architecture

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// TestE009_PerModuleProfileFiresForNonTransportModule verifies that E009
// evaluates HasTransport per-module: a module with command+query but no
// transport must fire even when another sub-module HAS transport.
// With the old primary-profile behavior, the example's transport would
// suppress the library module's finding.
func TestE009_PerModuleProfileFiresForNonTransportModule(t *testing.T) {
	t.Parallel()

	source := `package test

import (
	"github.com/larsartmann/go-cqrs-lite/command"
	"github.com/larsartmann/go-cqrs-lite/query"
)

func setup() {
	_ = command.BasicCommand{}
	_ = query.PaginatedResult[any]{}
}
`

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"/repo/lib/handler.go": source,
	})

	// Set ModuleDir so E009 can group files by module.
	for i := range ctx.GoFiles {
		ctx.GoFiles[i].ModuleDir = "/repo/lib"
	}

	ctx.FeatureProfiles = map[string]analyzer.FeatureProfile{
		"/repo/lib":          {HasTransport: false},
		"/repo/examples/app": {HasTransport: true},
	}
	ctx.FeatureProfile = analyzer.FeatureProfile{HasTransport: true}

	det := NewE009Detector(ctx)
	findings, err := det.Detect(context.Background())
	if err != nil {
		t.Fatalf("E009 detect: %v", err)
	}

	if len(findings) != 1 {
		t.Errorf("expected 1 finding for library module (no transport), got %d", len(findings))
		for _, f := range findings {
			t.Logf("  finding: %s", f.Message)
		}
	}
}

// TestE009_PerModuleProfileSuppressesTransportModule verifies that E009 does
// NOT fire for a module that HAS transport.
func TestE009_PerModuleProfileSuppressesTransportModule(t *testing.T) {
	t.Parallel()

	source := `package test

import (
	"github.com/larsartmann/go-cqrs-lite/command"
	"github.com/larsartmann/go-cqrs-lite/query"
)

func setup() {
	_ = command.BasicCommand{}
	_ = query.PaginatedResult[any]{}
}
`

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"/repo/examples/app/handler.go": source,
	})

	for i := range ctx.GoFiles {
		ctx.GoFiles[i].ModuleDir = "/repo/examples/app"
	}

	ctx.FeatureProfiles = map[string]analyzer.FeatureProfile{
		"/repo/lib":          {HasTransport: false},
		"/repo/examples/app": {HasTransport: true},
	}
	ctx.FeatureProfile = analyzer.FeatureProfile{HasTransport: true}

	det := NewE009Detector(ctx)
	findings, err := det.Detect(context.Background())
	if err != nil {
		t.Fatalf("E009 detect: %v", err)
	}

	if len(findings) != 0 {
		t.Errorf("expected 0 findings for transport module, got %d", len(findings))
	}
}

// TestE009_SingleModuleFallback verifies backward compatibility: a
// single-module project falls back to the primary profile.
func TestE009_SingleModuleFallback(t *testing.T) {
	t.Parallel()

	source := `package test

import (
	"github.com/larsartmann/go-cqrs-lite/command"
	"github.com/larsartmann/go-cqrs-lite/query"
)

func setup() {
	_ = command.BasicCommand{}
	_ = query.PaginatedResult[any]{}
}
`

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"handler.go": source,
	})

	// Single-module: no FeatureProfiles set, no transport.
	ctx.FeatureProfile = analyzer.FeatureProfile{HasTransport: false}

	det := NewE009Detector(ctx)
	findings, err := det.Detect(context.Background())
	if err != nil {
		t.Fatalf("E009 detect: %v", err)
	}

	if len(findings) != 1 {
		t.Errorf("expected 1 finding for single-module project without transport, got %d", len(findings))
	}
}

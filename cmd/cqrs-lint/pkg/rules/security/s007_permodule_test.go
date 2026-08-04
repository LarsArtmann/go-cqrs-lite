package security

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// TestS007_PerModuleProfileSuppressesNonServerModule verifies that S007
// evaluates HasServer per-module: an in-memory session store in a library
// module (no server) must NOT fire when an example sub-module has a server.
// With the old primary-profile behavior, the example's server would leak
// into the library module and produce a false positive.
func TestS007_PerModuleProfileSuppressesNonServerModule(t *testing.T) {
	t.Parallel()

	source := `package test

type InMemorySessionStore struct{}

func NewInMemorySessionStore() *InMemorySessionStore { return &InMemorySessionStore{} }
`

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"/repo/lib/auth.go": source,
	})

	// Simulate a multi-module workspace: the library has no server,
	// but an example sub-module does.
	ctx.FeatureProfiles = map[string]analyzer.FeatureProfile{
		"/repo/lib":          {HasServer: false},
		"/repo/examples/app": {HasServer: true},
	}
	// Primary profile reflects the example app (the "most significant" module).
	ctx.FeatureProfile = analyzer.FeatureProfile{HasServer: true}

	det := NewS007Detector(ctx)
	findings, err := det.Detect(context.Background())
	if err != nil {
		t.Fatalf("S007 detect: %v", err)
	}

	if len(findings) != 0 {
		t.Errorf("expected 0 findings for library module (no server), got %d", len(findings))
		for _, f := range findings {
			t.Logf("  unexpected: %s", f.Message)
		}
	}
}

// TestS007_PerModuleProfileFiresForServerModule verifies that S007 still
// fires when the session store IS in a module that has a server.
func TestS007_PerModuleProfileFiresForServerModule(t *testing.T) {
	t.Parallel()

	source := `package test

type InMemorySessionStore struct{}

func NewInMemorySessionStore() *InMemorySessionStore { return &InMemorySessionStore{} }
`

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"/repo/examples/app/auth.go": source,
	})

	ctx.FeatureProfiles = map[string]analyzer.FeatureProfile{
		"/repo/lib":          {HasServer: false},
		"/repo/examples/app": {HasServer: true},
	}
	ctx.FeatureProfile = analyzer.FeatureProfile{HasServer: true}

	det := NewS007Detector(ctx)
	findings, err := det.Detect(context.Background())
	if err != nil {
		t.Fatalf("S007 detect: %v", err)
	}

	if len(findings) != 1 {
		t.Errorf("expected 1 finding for server module, got %d", len(findings))
		for _, f := range findings {
			t.Logf("  finding: %s", f.Message)
		}
	}
}

// TestS007_SingleModuleFallbackStillWorks verifies that a single-module
// project (no FeatureProfiles) falls back to the primary profile — backward
// compatible with the pre-per-module behavior.
func TestS007_SingleModuleFallbackStillWorks(t *testing.T) {
	t.Parallel()

	source := `package test

type InMemorySessionStore struct{}

func NewInMemorySessionStore() *InMemorySessionStore { return &InMemorySessionStore{} }
`

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"auth.go": source,
	})

	// Single-module: no FeatureProfiles set.
	ctx.FeatureProfile = analyzer.FeatureProfile{HasServer: true}

	det := NewS007Detector(ctx)
	findings, err := det.Detect(context.Background())
	if err != nil {
		t.Fatalf("S007 detect: %v", err)
	}

	if len(findings) != 1 {
		t.Errorf("expected 1 finding for single-module server project, got %d", len(findings))
	}
}

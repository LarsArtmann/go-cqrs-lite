package resilience

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// TestB029_PerModuleSuppressesLibraryModule verifies that B029 (missing retry
// middleware) evaluates HasServer per-module: a bus in a library module (no
// server) must NOT fire when an example sub-module has a server. With the old
// primary-profile behavior, the example's server leaked into the library.
func TestB029_PerModuleSuppressesLibraryModule(t *testing.T) {
	t.Parallel()

	libSrc := `package lib
func init() {
	eventBus := struct{}{}
	eventBus.Use()
}
`

	exampleSrc := `package main
import "net/http"
func main() {
	commandBus := struct{}{}
	commandBus.Use()
	_ = http.ListenAndServe(":8080", nil)
}
`

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"/repo/lib/bus.go":           libSrc,
		"/repo/examples/app/main.go": exampleSrc,
	})

	ctx.FeatureProfiles = map[string]analyzer.FeatureProfile{
		"/repo/lib":          {HasServer: false},
		"/repo/examples/app": {HasServer: true},
	}
	ctx.FeatureProfile = analyzer.FeatureProfile{HasServer: true}

	det := NewB029Detector(ctx)
	findings, err := det.Detect(context.Background())
	if err != nil {
		t.Fatalf("B029 detect: %v", err)
	}

	// B029 should fire exactly once — for the example module's commandBus
	// (server module, no retry middleware). The library's eventBus (no server)
	// must not fire.
	if len(findings) != 1 {
		t.Errorf("expected 1 finding (example module only), got %d", len(findings))
		for _, f := range findings {
			t.Logf("  finding: %s @ %s", f.Message, f.Position.File)
		}
	}
}

// TestB029_PerModuleFiresForServerModule verifies that B029 still fires when
// the bus IS in a module that has a server.
func TestB029_PerModuleFiresForServerModule(t *testing.T) {
	t.Parallel()

	src := `package main
import "net/http"
func main() {
	commandBus := struct{}{}
	commandBus.Use()
	_ = http.ListenAndServe(":8080", nil)
}
`

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"/repo/examples/app/main.go": src,
	})

	ctx.FeatureProfiles = map[string]analyzer.FeatureProfile{
		"/repo/lib":          {HasServer: false},
		"/repo/examples/app": {HasServer: true},
	}
	ctx.FeatureProfile = analyzer.FeatureProfile{HasServer: true}

	det := NewB029Detector(ctx)
	findings, err := det.Detect(context.Background())
	if err != nil {
		t.Fatalf("B029 detect: %v", err)
	}

	if len(findings) != 1 {
		t.Errorf("expected 1 finding for server module, got %d", len(findings))
	}
}

// TestB031_PerModuleSuppressesLibraryModule verifies that B031 (missing DLQ
// config) evaluates HasServer per-module: a projectionhost.New() call in a
// library module (no server) must NOT fire when an example has a server.
func TestB031_PerModuleSuppressesLibraryModule(t *testing.T) {
	t.Parallel()

	libSrc := `package lib
import "github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
func newProjection() { projectionhost.New() }
`

	exampleSrc := `package main
import "net/http"
import "github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
func main() {
	projectionhost.New()
	_ = http.ListenAndServe(":8080", nil)
}
`

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"/repo/lib/proj.go":          libSrc,
		"/repo/examples/app/main.go": exampleSrc,
	})

	ctx.FeatureProfiles = map[string]analyzer.FeatureProfile{
		"/repo/lib":          {HasServer: false},
		"/repo/examples/app": {HasServer: true},
	}
	ctx.FeatureProfile = analyzer.FeatureProfile{HasServer: true}

	det := NewB031Detector(ctx)
	findings, err := det.Detect(context.Background())
	if err != nil {
		t.Fatalf("B031 detect: %v", err)
	}

	// B031 should fire exactly once — for the example module's projectionhost.New
	// (server module, no DLQ). The library's projectionhost.New (no server)
	// must not fire.
	if len(findings) != 1 {
		t.Errorf("expected 1 finding (example module only), got %d", len(findings))
		for _, f := range findings {
			t.Logf("  finding: %s @ %s", f.Message, f.Position.File)
		}
	}
}

// TestB029_SingleModuleFallbackStillWorks verifies that a single-module project
// (no FeatureProfiles) falls back to the primary profile.
func TestB029_SingleModuleFallbackStillWorks(t *testing.T) {
	t.Parallel()

	src := `package main
import "net/http"
func main() {
	commandBus := struct{}{}
	commandBus.Use()
	_ = http.ListenAndServe(":8080", nil)
}
`

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": src,
	})

	ctx.FeatureProfile = analyzer.FeatureProfile{HasServer: true}

	det := NewB029Detector(ctx)
	findings, err := det.Detect(context.Background())
	if err != nil {
		t.Fatalf("B029 detect: %v", err)
	}

	if len(findings) != 1 {
		t.Errorf("expected 1 finding for single-module server project, got %d", len(findings))
	}
}

// TestB030_PerModuleSuppressesLibraryModule verifies that B030 (missing
// circuit breaker) evaluates HasServer per-module: a bus in a library module
// (no server) must NOT fire when an example sub-module has a server.
func TestB030_PerModuleSuppressesLibraryModule(t *testing.T) {
	t.Parallel()

	libSrc := `package lib
func init() {
	eventBus := struct{}{}
	eventBus.Use()
}
`

	exampleSrc := `package main
import "net/http"
func main() {
	commandBus := struct{}{}
	commandBus.Use()
	_ = http.ListenAndServe(":8080", nil)
}
`

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"/repo/lib/bus.go":           libSrc,
		"/repo/examples/app/main.go": exampleSrc,
	})

	ctx.FeatureProfiles = map[string]analyzer.FeatureProfile{
		"/repo/lib":          {HasServer: false},
		"/repo/examples/app": {HasServer: true},
	}
	ctx.FeatureProfile = analyzer.FeatureProfile{HasServer: true}

	det := NewB030Detector(ctx)
	findings, err := det.Detect(context.Background())
	if err != nil {
		t.Fatalf("B030 detect: %v", err)
	}

	// B030 should fire exactly once — for the example module's commandBus
	// (server module, no circuit breaker). The library's eventBus (no server)
	// must not fire.
	if len(findings) != 1 {
		t.Errorf("expected 1 finding (example module only), got %d", len(findings))
		for _, f := range findings {
			t.Logf("  finding: %s @ %s", f.Message, f.Position.File)
		}
	}
}

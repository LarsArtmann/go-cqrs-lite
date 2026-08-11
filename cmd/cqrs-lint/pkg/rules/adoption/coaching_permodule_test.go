package adoption

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// TestF003_PerModuleSuppressesLibraryModule verifies that F003 (no OTel
// tracing) evaluates HasServer per-module: a library module without a server
// must NOT be coached when an example sub-module has a server. With the old
// primary-profile behavior, the example's server leaked into the library.
func TestF003_PerModuleSuppressesLibraryModule(t *testing.T) {
	t.Parallel()

	libSrc := `package lib
import _ "github.com/larsartmann/go-cqrs-lite/event/v4"
type UserCreated struct{ Name string }
`

	exampleSrc := `package main
import "net/http"
func main() { _ = http.ListenAndServe(":8080", nil) }
`

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"/repo/lib/events.go":        libSrc,
		"/repo/examples/app/main.go": exampleSrc,
	})

	ctx.FeatureProfiles = map[string]analyzer.FeatureProfile{
		"/repo/lib":          {HasServer: false},
		"/repo/examples/app": {HasServer: true},
	}
	ctx.FeatureProfile = analyzer.FeatureProfile{HasServer: false}

	det := NewF003Detector(ctx)
	findings, err := det.Detect(context.Background())
	if err != nil {
		t.Fatalf("F003 detect: %v", err)
	}

	// F003 should fire exactly once — for the example module (server, no OTel).
	// The library module (no server) must not be coached.
	if len(findings) != 1 {
		t.Errorf("expected 1 finding (example module only), got %d", len(findings))
		for _, f := range findings {
			t.Logf("  finding: %s @ %s", f.Message, f.Position.File)
		}
	}
}

// TestF013_PerModuleSuppressesLibraryModule verifies that F013 (no transport
// module) evaluates HasServer per-module: a library module must NOT be coached
// when an example sub-module has manual HTTP handlers.
func TestF013_PerModuleSuppressesLibraryModule(t *testing.T) {
	t.Parallel()

	libSrc := `package lib
import _ "github.com/larsartmann/go-cqrs-lite/event/v4"
type UserCreated struct{ Name string }
`

	exampleSrc := `package main
import "net/http"
func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {})
	_ = http.ListenAndServe(":8080", nil)
}
`

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"/repo/lib/events.go":        libSrc,
		"/repo/examples/app/main.go": exampleSrc,
	})

	ctx.FeatureProfiles = map[string]analyzer.FeatureProfile{
		"/repo/lib":          {HasServer: false},
		"/repo/examples/app": {HasServer: true},
	}
	ctx.FeatureProfile = analyzer.FeatureProfile{HasServer: false}

	det := NewF013Detector(ctx)
	findings, err := det.Detect(context.Background())
	if err != nil {
		t.Fatalf("F013 detect: %v", err)
	}

	// F013 should fire exactly once — for the example module (server, manual
	// handlers, no transport). The library (no server) must not be coached.
	if len(findings) != 1 {
		t.Errorf("expected 1 finding (example module only), got %d", len(findings))
		for _, f := range findings {
			t.Logf("  finding: %s @ %s", f.Message, f.Position.File)
		}
	}
}

// TestF022_PerModuleStoreIsolation verifies that F022 (manual sort, no
// metaengine) evaluates Store per-module: a library module with StoreNone
// must NOT fire even if an example module has a SQL store with manual sorts.
func TestF022_PerModuleStoreIsolation(t *testing.T) {
	t.Parallel()

	libSrc := `package lib
import "sort"
type Item struct{ Name string }
func sortItems(items []Item) { sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name }) }
`

	exampleSrc := `package main
import "sort"
func main() {
	items := []string{"b", "a"}
	sort.Slice(items, func(i, j int) bool { return items[i] < items[j] })
}
`

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"/repo/lib/sort.go":          libSrc,
		"/repo/examples/app/main.go": exampleSrc,
	})

	ctx.FeatureProfiles = map[string]analyzer.FeatureProfile{
		"/repo/lib":          {Store: analyzer.StoreNone},
		"/repo/examples/app": {Store: analyzer.StoreSQLite},
	}
	ctx.FeatureProfile = analyzer.FeatureProfile{Store: analyzer.StoreNone}

	det := NewF022Detector(ctx)
	findings, err := det.Detect(context.Background())
	if err != nil {
		t.Fatalf("F022 detect: %v", err)
	}

	// F022 should fire exactly once — for the example module (SQLite + manual
	// sort). The library (StoreNone) must not be coached.
	if len(findings) != 1 {
		t.Errorf("expected 1 finding (example module only), got %d", len(findings))
		for _, f := range findings {
			t.Logf("  finding: %s @ %s", f.Message, f.Position.File)
		}
	}
}

// TestF003_SingleModuleFallbackStillWorks verifies that a single-module project
// (no FeatureProfiles) falls back to the primary profile.
func TestF003_SingleModuleFallbackStillWorks(t *testing.T) {
	t.Parallel()

	src := `package main
import "net/http"
func main() { _ = http.ListenAndServe(":8080", nil) }
`

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": src,
	})

	ctx.FeatureProfile = analyzer.FeatureProfile{HasServer: true}

	det := NewF003Detector(ctx)
	findings, err := det.Detect(context.Background())
	if err != nil {
		t.Fatalf("F003 detect: %v", err)
	}

	if len(findings) != 1 {
		t.Errorf("expected 1 finding for single-module server project, got %d", len(findings))
	}
}

// TestCoachingScopesSingleModuleReturnsAllFiles verifies that coachingScopes
// returns exactly one scope with all non-test files for a single-module project.
func TestCoachingScopesSingleModuleReturnsAllFiles(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"a.go": "package test\nvar X = 1",
		"b.go": "package test\nvar Y = 2",
	})

	scopes := coachingScopes(ctx)
	if len(scopes) != 1 {
		t.Fatalf("expected 1 scope, got %d", len(scopes))
	}
	if len(scopes[0].files) != 2 {
		t.Errorf("expected 2 files in single scope, got %d", len(scopes[0].files))
	}
}

package architecture

import (
	"context"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

// TestE016_PerModuleProfileSuppressesServerLocalModule verifies that E016
// evaluates ServerLocal per-module: a server in a local-only library module
// must NOT fire when an example sub-module is a production server.
func TestE016_PerModuleProfileSuppressesServerLocalModule(t *testing.T) {
	t.Parallel()

	source := `package test

import "net/http"

func runServer() {
	srv := &http.Server{Addr: ":8080"}
	_ = srv.ListenAndServe()
}
`

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"/repo/lib/server.go": source,
	})

	ctx.FeatureProfiles = map[string]analyzer.FeatureProfile{
		"/repo/lib":          {ServerLocal: true},
		"/repo/examples/app": {ServerLocal: false},
	}
	ctx.FeatureProfile = analyzer.FeatureProfile{ServerLocal: false}

	det := NewE016Detector(ctx)
	findings, err := det.Detect(context.Background())
	if err != nil {
		t.Fatalf("E016 detect: %v", err)
	}

	if len(findings) != 0 {
		t.Errorf("expected 0 findings for server-local module, got %d", len(findings))
	}
}

// TestE016_PerModuleProfileFiresForProductionModule verifies that E016 still
// fires when the server IS in a production module (ServerLocal=false).
func TestE016_PerModuleProfileFiresForProductionModule(t *testing.T) {
	t.Parallel()

	source := `package test

import "net/http"

func runServer() {
	srv := &http.Server{Addr: ":8080"}
	_ = srv.ListenAndServe()
}
`

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"/repo/examples/app/server.go": source,
	})

	ctx.FeatureProfiles = map[string]analyzer.FeatureProfile{
		"/repo/lib":          {ServerLocal: true},
		"/repo/examples/app": {ServerLocal: false},
	}
	ctx.FeatureProfile = analyzer.FeatureProfile{ServerLocal: false}

	det := NewE016Detector(ctx)
	findings, err := det.Detect(context.Background())
	if err != nil {
		t.Fatalf("E016 detect: %v", err)
	}

	if len(findings) != 1 {
		t.Errorf("expected 1 finding for production server module, got %d", len(findings))
	}
}

// TestE016_SingleModuleFallback verifies backward compatibility: a
// single-module production project falls back to the primary profile.
func TestE016_SingleModuleFallback(t *testing.T) {
	t.Parallel()

	source := `package test

import "net/http"

func runServer() {
	srv := &http.Server{Addr: ":8080"}
	_ = srv.ListenAndServe()
}
`

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"server.go": source,
	})

	ctx.FeatureProfile = analyzer.FeatureProfile{ServerLocal: false}

	det := NewE016Detector(ctx)
	findings, err := det.Detect(context.Background())
	if err != nil {
		t.Fatalf("E016 detect: %v", err)
	}

	if len(findings) != 1 {
		t.Errorf("expected 1 finding for single-module production server, got %d", len(findings))
	}
}

// TestE016_ServerLocalModuleDoesNotHideProductionModule: in a multi-module
// repo the ServerLocal profile of one module must not suppress the finding
// for a server in another module. E016 used to keep ONE trigger position and
// one profile for the whole workspace.
func TestE016_ServerLocalModuleDoesNotHideProductionModule(t *testing.T) {
	t.Parallel()

	source := `package test

import "net/http"

func runServer() {
	srv := &http.Server{Addr: ":8080"}
	_ = srv.ListenAndServe()
}
`

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"/repo/lib/server.go":          source,
		"/repo/examples/app/server.go": source,
	})
	ctx.FeatureProfiles = map[string]analyzer.FeatureProfile{
		"/repo/lib":          {ServerLocal: true},
		"/repo/examples/app": {ServerLocal: false},
	}

	findings, err := NewE016Detector(ctx).Detect(context.Background())
	if err != nil {
		t.Fatalf("E016 detect: %v", err)
	}

	if len(findings) != 1 || !strings.HasSuffix(string(findings[0].Position.File), "examples/app/server.go") {
		t.Errorf("expected 1 finding for the production module only, got %d: %+v", len(findings), findings)
	}
}

// TestE016_HealthRouteInOneModuleDoesNotSilenceAnother: a /healthz route in
// module A says nothing about the server in module B.
func TestE016_HealthRouteInOneModuleDoesNotSilenceAnother(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"/repo/lib/routes.go": `package test

import "net/http"

func routes() {
	http.HandleFunc("/healthz", healthHandler)
}
`,
		"/repo/examples/app/server.go": `package test

import "net/http"

func runServer() {
	srv := &http.Server{Addr: ":8080"}
	_ = srv.ListenAndServe()
}
`,
	})
	ctx.FeatureProfile = analyzer.FeatureProfile{ServerLocal: false}

	findings, err := NewE016Detector(ctx).Detect(context.Background())
	if err != nil {
		t.Fatalf("E016 detect: %v", err)
	}

	if len(findings) != 1 || !strings.HasSuffix(string(findings[0].Position.File), "examples/app/server.go") {
		t.Errorf("expected 1 finding for the server module, got %d: %+v", len(findings), findings)
	}
}

// TestE016_OneFindingPerServerModule: two server modules yield two findings,
// each anchored in its own module.
func TestE016_OneFindingPerServerModule(t *testing.T) {
	t.Parallel()

	source := `package test

import "net/http"

func runServer() {
	srv := &http.Server{Addr: ":8080"}
	_ = srv.ListenAndServe()
}
`

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"/repo/services/a/server.go": source,
		"/repo/services/b/server.go": source,
	})
	ctx.FeatureProfile = analyzer.FeatureProfile{ServerLocal: false}

	findings, err := NewE016Detector(ctx).Detect(context.Background())
	if err != nil {
		t.Fatalf("E016 detect: %v", err)
	}

	if len(findings) != 2 {
		t.Errorf("expected 2 findings (one per server module), got %d: %+v", len(findings), findings)
	}
}

package architecture

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
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

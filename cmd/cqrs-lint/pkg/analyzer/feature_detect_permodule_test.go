package analyzer

import (
	"go/parser"
	"go/token"
	"testing"

	"golang.org/x/tools/go/packages"
)

// parseGoFile parses a source string into a GoFile with the given module dir.
// Used by per-module feature-detection tests.
func parseGoFile(t *testing.T, fset *token.FileSet, name, src, moduleDir string) *GoFile {
	t.Helper()

	file, err := parser.ParseFile(fset, name, src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse %s: %v", name, err)
	}

	return &GoFile{
		Path:      moduleDir + "/" + name,
		AST:       file,
		ModuleDir: moduleDir,
		Pkg: &packages.Package{
			PkgPath: "test.example/" + name,
		},
	}
}

// pkgWithImports builds a minimal *packages.Package carrying the given import
// paths, for import-based feature detection in tests.
func pkgWithImports(pkgPath string, imports ...string) *packages.Package {
	pkg := &packages.Package{
		PkgPath: pkgPath,
		Imports: map[string]*packages.Package{},
	}
	for _, imp := range imports {
		pkg.Imports[imp] = &packages.Package{PkgPath: imp}
	}

	return pkg
}

func TestDetectFeaturesPerModule_SeparatesLibraryAndExample(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	ctx := &AnalysisContext{
		Fset:     fset,
		Registry: NewCQRSRegistry(),
	}

	libDir := "/repo/lib"
	exampleDir := "/repo/lib/examples/basic"

	// Library module: imports the event module, no server.
	libPkg := pkgWithImports("test.example/lib",
		"github.com/larsartmann/go-cqrs-lite/event/v4")
	ctx.GoFiles = append(ctx.GoFiles, parseGoFile(t, fset, "lib.go", `
package lib
import _ "github.com/larsartmann/go-cqrs-lite/event/v4"
type UserCreated struct{ Name string }
`, libDir))

	// Example module: imports stack/sqlite and calls ListenAndServe (a server).
	examplePkg := pkgWithImports("test.example/examples/basic",
		"github.com/larsartmann/go-cqrs-lite/stack/sqlite")
	ctx.GoFiles = append(ctx.GoFiles, parseGoFile(t, fset, "main.go", `
package main
import "net/http"
func main() { _ = http.ListenAndServe(":8080", nil) }
`, exampleDir))

	packagesByModule := map[string][]*packages.Package{
		libDir:     {libPkg},
		exampleDir: {examplePkg},
	}

	profiles := DetectFeaturesPerModule(ctx, packagesByModule)

	if len(profiles) != 2 {
		t.Fatalf("got %d module profiles, want 2", len(profiles))
	}

	libProfile := profiles[libDir]
	exampleProfile := profiles[exampleDir]

	// The library module must NOT be flagged as a server — the example's
	// ListenAndServe must not leak across module boundaries.
	if libProfile.HasServer {
		t.Errorf("library module should have HasServer=false, got true (example leaked)")
	}
	if libProfile.Store != StoreNone {
		t.Errorf("library module store should be none, got %s", libProfile.Store)
	}

	// The example module IS a server using sqlite.
	if !exampleProfile.HasServer {
		t.Errorf("example module should have HasServer=true, got false")
	}
	if exampleProfile.Store != StoreSQLite {
		t.Errorf("example module store should be sqlite, got %s", exampleProfile.Store)
	}
}

func TestProfileForFile_ResolvesByLongestPrefix(t *testing.T) {
	t.Parallel()

	ctx := &AnalysisContext{
		Registry: NewCQRSRegistry(),
		FeatureProfile: FeatureProfile{
			Store:     StoreNone,
			HasServer: false,
		},
		FeatureProfiles: map[string]FeatureProfile{
			"/repo":              {Store: StorePostgres, HasServer: false},
			"/repo/examples/app": {Store: StoreSQLite, HasServer: true},
		},
	}

	tests := []struct {
		name   string
		file   string
		want   StoreKind
		server bool
	}{
		{"root module file", "/repo/lib/foo.go", StorePostgres, false},
		{"nested example file", "/repo/examples/app/main.go", StoreSQLite, true},
		{"unknown file falls back", "/other/place.go", StoreNone, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p := ctx.ProfileForFile(tc.file)
			if p.Store != tc.want {
				t.Errorf("store = %s, want %s", p.Store, tc.want)
			}
			if p.HasServer != tc.server {
				t.Errorf("HasServer = %v, want %v", p.HasServer, tc.server)
			}
		})
	}
}

func TestBuildContext_SingleModuleUnchangedByPerModule(t *testing.T) {
	t.Parallel()

	// A single-module context (no FeatureProfiles) must fall back to the
	// primary FeatureProfile for every file — backward compatible with the
	// pre-per-module behavior.
	ctx := &AnalysisContext{
		Registry: NewCQRSRegistry(),
		FeatureProfile: FeatureProfile{
			Store:     StoreMemory,
			HasServer: true,
		},
		// FeatureProfiles intentionally nil/empty
	}

	p := ctx.ProfileForFile("/anywhere/foo.go")
	if p.Store != StoreMemory {
		t.Errorf("single-module fallback store = %s, want memory", p.Store)
	}
	if !p.HasServer {
		t.Errorf("single-module fallback HasServer = false, want true")
	}
}

// T20-3: a package importing two different stack presets used to resolve its
// store in pkg.Imports map order — nondeterministic across runs. The scan now
// iterates sorted with a first-wins guard, so the smallest import path
// (stack/postgres) must win and every run must agree.
func TestDetectFeatureSignals_MultiPresetStoreDeterministic(t *testing.T) {
	t.Parallel()

	newPkgs := func() []*packages.Package {
		return []*packages.Package{pkgWithImports(
			"example.com/consumer",
			"github.com/larsartmann/go-cqrs-lite/stack/sqlite/v4",
			"github.com/larsartmann/go-cqrs-lite/stack/postgres/v4",
		)}
	}

	const runs = 40
	first := detectFeatureSignals(newPkgs(), nil, NewCQRSRegistry()).Store
	for range runs - 1 {
		got := detectFeatureSignals(newPkgs(), nil, NewCQRSRegistry()).Store
		if got != first {
			t.Fatalf("store resolution nondeterministic across %d runs: %q then %q", runs, first, got)
		}
	}

	if first != StorePostgres {
		t.Fatalf("multi-preset store = %s, want postgres (sorted first-wins)", first)
	}
}

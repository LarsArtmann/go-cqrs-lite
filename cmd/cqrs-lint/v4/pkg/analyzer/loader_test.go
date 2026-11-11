package analyzer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBuildContext_BrokenModule verifies that BuildContext collects LoadErrors
// when a module's dependencies cannot be resolved. This is the regression test
// for the silent-failure bug where the loader would swallow errors and produce
// an empty AnalysisContext.
func TestBuildContext_BrokenModule(t *testing.T) {
	dir := t.TempDir()

	goMod := `module testproject

go 1.26.4

require github.com/larsartmann/go-cqrs-lite/event/v4 v9.9.9
`
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}

	goFile := `package main

import "github.com/larsartmann/go-cqrs-lite/event/v4"

func main() {
	_ = event.New
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(goFile), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("GOWORK", "off")
	t.Setenv("GOPROXY", "off")

	ctx, err := BuildContext(dir)
	if err != nil {
		t.Fatalf("BuildContext returned unexpected error: %v", err)
	}

	if len(ctx.LoadErrors) == 0 {
		t.Error("expected LoadErrors to be non-empty for a project with unresolvable dependencies")
	}

	if len(ctx.GoFiles) != 0 {
		t.Errorf(
			"expected GoFiles to be empty when all packages failed to load, got %d",
			len(ctx.GoFiles),
		)
	}
}

// TestBuildContext_CleanModule verifies that BuildContext produces no LoadErrors
// for a module that compiles successfully but does not import go-cqrs-lite.
func TestBuildContext_CleanModule(t *testing.T) {
	dir := t.TempDir()

	goMod := `module cleanproject

go 1.26.4
`
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}

	goFile := `package main

import "fmt"

func main() {
	fmt.Println("hello")
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(goFile), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("GOWORK", "off")

	ctx, err := BuildContext(dir)
	if err != nil {
		t.Fatalf("BuildContext returned unexpected error: %v", err)
	}

	if len(ctx.LoadErrors) > 0 {
		t.Errorf(
			"expected no LoadErrors for a clean module, got %d: %v",
			len(ctx.LoadErrors),
			ctx.LoadErrors,
		)
	}

	if len(ctx.GoFiles) != 0 {
		t.Errorf("expected GoFiles to be empty (no go-cqrs-lite imports), got %d", len(ctx.GoFiles))
	}

	if len(ctx.Packages) == 0 {
		t.Error("expected Packages to be non-empty for a valid module")
	}
}

// TestBuildContext_NoGoFiles verifies that BuildContext handles a directory
// with a go.mod but no Go source files gracefully.
func TestBuildContext_NoGoFiles(t *testing.T) {
	dir := t.TempDir()

	goMod := `module emptyproject

go 1.26.4
`
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("GOWORK", "off")

	ctx, err := BuildContext(dir)
	if err != nil {
		t.Fatalf("BuildContext returned unexpected error: %v", err)
	}

	if len(ctx.LoadErrors) > 0 {
		t.Errorf("expected no LoadErrors for an empty module, got %d", len(ctx.LoadErrors))
	}

	if len(ctx.GoFiles) != 0 {
		t.Errorf("expected GoFiles to be empty, got %d", len(ctx.GoFiles))
	}
}

// TestBuildContext_SyntaxError verifies that BuildContext collects LoadErrors
// for packages with syntax errors without crashing.
func TestBuildContext_SyntaxError(t *testing.T) {
	dir := t.TempDir()

	goMod := `module brokenproject

go 1.26.4
`
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}

	goFile := `package main

func main() {
	// syntax error: missing closing brace
`
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(goFile), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("GOWORK", "off")

	ctx, err := BuildContext(dir)
	if err != nil {
		t.Fatalf("BuildContext returned unexpected error: %v", err)
	}

	if len(ctx.LoadErrors) == 0 {
		t.Error("expected LoadErrors to be non-empty for a package with syntax errors")
	}
}

// TestBuildContext_SilentEmptyBrokenGraphFailsLoudly pins the v4.13.1
// silent-empty guard: go/packages can return ZERO packages with ZERO load
// errors when the module graph hides all Go sources (here: every file behind
// a build constraint, the deterministic stand-in for the stale-go.sum shape —
// a require without go.sum surfaces package errors, which is the loud
// LoadErrors path, not the silent one). BuildContext must reject loudly; a
// repo with no Go sources at all stays a legitimate clean verdict.
func TestBuildContext_SilentEmptyBrokenGraphFailsLoudly(t *testing.T) {
	t.Setenv("GOWORK", "off")
	t.Setenv("GOFLAGS", "-mod=mod")

	dir := t.TempDir()

	goMod := `module brokenfixture

go 1.27
`
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}

	tagged := `//go:build never_true_tag

package main

func main() {}
`
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(tagged), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := BuildContext(dir)
	if err == nil {
		t.Fatal("zero packages over a tree WITH Go sources must fail loudly, not report clean")
	}

	for _, want := range []string{"no packages loaded", "zero load errors"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error must mention %q, got: %v", want, err)
		}
	}

	empty := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(empty, "go.mod"),
		[]byte("module emptyroot\n\ngo 1.27\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	ctx, err := BuildContext(empty)
	if err != nil {
		t.Fatalf("a repo with no Go sources is a legitimate clean verdict: %v", err)
	}

	if ctx == nil {
		t.Fatal("context must not be nil on a clean empty repo")
	}
}

// TestFindEnclosingGoModDir_WalksUp pins the upward module-context walk:
// a root inside a module resolves to the nearest enclosing go.mod, and a
// filesystem without one answers false.
func TestFindEnclosingGoModDir_WalksUp(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	nested := filepath.Join(root, "internal", "sub")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module m\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, ok := findEnclosingGoModDir(nested)
	if !ok || got != root {
		t.Fatalf("findEnclosingGoModDir(%s) = (%q, %v), want (%q, true)", nested, got, ok, root)
	}

	bare := t.TempDir()
	if _, ok := findEnclosingGoModDir(bare); ok {
		t.Fatalf("a tree without go.mod must not resolve an enclosing module")
	}
}

func TestIsUnderDir(t *testing.T) {
	t.Parallel()

	if !isUnderDir("/a/b/c.go", "/a/b") || !isUnderDir("/a/b", "/a/b") {
		t.Error("path under (or equal to) dir must match")
	}

	if isUnderDir("/a/bx/c.go", "/a/b") {
		t.Error("a sibling prefix must not match")
	}
}

// TestBuildContext_SubtreeRunUsesEnclosingModuleProfile pins issue #126's
// root cause: analyzing a subtree of a module (no go.mod at or under the
// root) must still detect features from the whole ENCLOSING module —
// HasServer lives in the module's server file, not in the internal/
// package being linted. Findings stay scoped to the requested subtree.
func TestBuildContext_SubtreeRunUsesEnclosingModuleProfile(t *testing.T) {
	t.Setenv("GOWORK", "off")
	t.Setenv("GOFLAGS", "-mod=mod")
	t.Setenv("GOPROXY", "off")
	t.Setenv("GOSUMDB", "off")

	dir := t.TempDir()

	goMod := `module subtreehost

go 1.27

require github.com/larsartmann/go-cqrs-lite/event/v4 v4.15.2
`
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(filepath.Join(dir, "cmd", "server"), 0o755); err != nil {
		t.Fatal(err)
	}
	server := `package main

import (
	"net/http"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

func main() {
	_ = event.New
	_ = http.ListenAndServe(":8080", nil)
}
`
	if err := os.WriteFile(filepath.Join(dir, "cmd", "server", "main.go"), []byte(server), 0o644); err != nil {
		t.Fatal(err)
	}

	subtree := filepath.Join(dir, "internal", "sub")
	if err := os.MkdirAll(subtree, 0o755); err != nil {
		t.Fatal(err)
	}
	bus := `package sub

import "github.com/larsartmann/go-cqrs-lite/event/v4"

var _ = event.New
`
	if err := os.WriteFile(filepath.Join(subtree, "bus.go"), []byte(bus), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx, err := BuildContext(subtree)
	if err != nil {
		t.Fatalf("BuildContext(subtree): %v", err)
	}

	if !ctx.FeatureProfile.HasServer {
		t.Fatal("subtree run must see the enclosing module's server signal")
	}

	if len(ctx.FeatureProfiles) != 1 {
		t.Fatalf("expected exactly one per-module profile (the enclosing module); got %v",
			ctx.FeatureProfiles)
	}
	if _, ok := ctx.FeatureProfiles[dir]; !ok {
		t.Errorf("per-module profile must be keyed by the enclosing module dir %s; got %v", dir,
			ctx.FeatureProfiles)
	}

	for _, gf := range ctx.GoFiles {
		if !isUnderDir(gf.Path, subtree) {
			t.Errorf("findings scope leaked outside the requested subtree: %s", gf.Path)
		}
	}

	if !ctx.ProfileForFile(filepath.Join(subtree, "bus.go")).HasServer {
		t.Error("ProfileForFile must resolve subtree files to the enclosing module's profile")
	}
}

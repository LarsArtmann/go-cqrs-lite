package analyzer

import (
	"os"
	"path/filepath"
	"testing"
)

// writeNestedModules creates a root go.mod plus a nested module under
// sub/, returning the two directories. Both modules are dependency-free so
// package loading works with GOPROXY=off.
func writeNestedModules(t *testing.T) (root, nested string) {
	t.Helper()

	root = t.TempDir()
	nested = filepath.Join(root, "sub")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	modules := []struct{ dir, module, pkg string }{
		{root, "testproject/root", "rootpkg"},
		{nested, "testproject/nested", "nestedpkg"},
	}
	for _, m := range modules {
		goMod := "module " + m.module + "\n\ngo 1.26.4\n"
		if err := os.WriteFile(filepath.Join(m.dir, "go.mod"), []byte(goMod), 0o644); err != nil {
			t.Fatal(err)
		}

		goFile := "package " + m.pkg + "\n\nfunc F() {}\n"
		if err := os.WriteFile(filepath.Join(m.dir, "a.go"), []byte(goFile), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	return root, nested
}

// TestModuleDirs_FindsNestedModules verifies the toolsdk-facing module walk:
// every directory holding a go.mod is reported, including the root itself.
func TestModuleDirs_FindsNestedModules(t *testing.T) {
	root, nested := writeNestedModules(t)

	dirs, err := ModuleDirs(root)
	if err != nil {
		t.Fatalf("ModuleDirs: %v", err)
	}

	want := map[string]bool{root: false, nested: false}
	for _, dir := range dirs {
		if _, known := want[dir]; known {
			want[dir] = true
		}
	}

	for dir, found := range want {
		if !found {
			t.Errorf("ModuleDirs missed %s", dir)
		}
	}
}

// TestBuildContextWithEnv_DiscoverMergesNestedModules verifies the
// GOWORK=off host semantics: scope discovery loads packages from the root
// AND nested modules into one context (asserted on ctx.Packages — GoFiles
// additionally requires a go-cqrs-lite import, IsCQRSImport).
func TestBuildContextWithEnv_DiscoverMergesNestedModules(t *testing.T) {
	root, _ := writeNestedModules(t)

	t.Setenv("GOPROXY", "off")

	ctx, err := BuildContextWithEnv(root, append(os.Environ(), "GOWORK=off"))
	if err != nil {
		t.Fatalf("BuildContextWithEnv: %v", err)
	}

	if len(ctx.Packages) < 2 {
		t.Errorf("expected packages from both modules, got %d Packages", len(ctx.Packages))
	}

	if len(ctx.LoadErrors) != 0 {
		t.Errorf("expected no LoadErrors for dependency-free modules, got %d", len(ctx.LoadErrors))
	}
}

// TestBuildContextForModule_ScopesToSingleModule verifies the CLI per-module
// semantics: nested go.mod discovery is disabled, so exactly the module AT
// projectRoot loads — the nested module's files never leak in.
func TestBuildContextForModule_ScopesToSingleModule(t *testing.T) {
	root, _ := writeNestedModules(t)

	t.Setenv("GOPROXY", "off")

	ctx, err := BuildContextForModule(root, append(os.Environ(), "GOWORK=off"))
	if err != nil {
		t.Fatalf("BuildContextForModule: %v", err)
	}

	if len(ctx.Packages) != 1 {
		t.Fatalf(
			"single-module scope must load exactly the root module's packages, got %d",
			len(ctx.Packages),
		)
	}

	if got := ctx.Packages[0].PkgPath; got != "testproject/root" {
		t.Errorf("loaded %s, want the root module's package", got)
	}
}

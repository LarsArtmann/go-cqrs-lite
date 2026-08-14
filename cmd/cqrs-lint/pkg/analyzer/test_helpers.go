package analyzer

import (
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

// BuildContextFromSource creates an AnalysisContext from Go source code strings.
// Each source is a map of filename → content. This is for unit testing rules
// without needing real Go module infrastructure.
func BuildContextFromSource(t *testing.T, sources map[string]string) *AnalysisContext {
	t.Helper()

	fset := token.NewFileSet()
	ctx := &AnalysisContext{
		Fset:     fset,
		Registry: NewCQRSRegistry(),
	}

	for filename, content := range sources {
		file, err := parser.ParseFile(fset, filename, content, parser.ParseComments)
		if err != nil {
			t.Fatalf("parse %s: %v", filename, err)
		}

		gf := &GoFile{
			Path:   filename,
			AST:    file,
			IsTest: strings.HasSuffix(filename, "_test.go"),
		}

		// Create a minimal packages.Package for type info (nil TypesInfo for AST-only rules).
		gf.Pkg = &packages.Package{
			PkgPath:   "test.example/" + strings.TrimSuffix(filename, ".go"),
			TypesInfo: &types.Info{},
		}

		ctx.GoFiles = append(ctx.GoFiles, gf)
		if !gf.IsTest {
			scanFile(ctx, gf)
		}
	}

	ctx.FeatureProfile = DetectFeatures(ctx)
	ResolveRegisteredTypeConsts(ctx.Registry)
	ResolveHandlerMethods(ctx)
	ResolveTransportAdapters(ctx)

	return ctx
}

// BuildContextFromTempFiles creates an AnalysisContext from Go source code
// strings written to real temp files. Unlike BuildContextFromSource, the
// files exist on disk so SourceLine() and the suppression filter can read
// them — necessary for testing inline //cqrs-lint:ignore suppression.
func BuildContextFromTempFiles(
	t *testing.T,
	sources map[string]string,
) (*AnalysisContext, func()) {
	t.Helper()

	dir := t.TempDir()

	fset := token.NewFileSet()
	ctx := &AnalysisContext{
		Fset:     fset,
		Registry: NewCQRSRegistry(),
	}

	for filename, content := range sources {
		fullPath := dir + "/" + filename
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", fullPath, err)
		}

		file, err := parser.ParseFile(fset, fullPath, content, parser.ParseComments)
		if err != nil {
			t.Fatalf("parse %s: %v", fullPath, err)
		}

		gf := &GoFile{
			Path:   fullPath,
			AST:    file,
			IsTest: strings.HasSuffix(filename, "_test.go"),
		}

		gf.Pkg = &packages.Package{
			PkgPath:   "test.example/" + strings.TrimSuffix(filename, ".go"),
			TypesInfo: &types.Info{},
		}

		ctx.GoFiles = append(ctx.GoFiles, gf)
		if !gf.IsTest {
			scanFile(ctx, gf)
		}
	}

	ctx.FeatureProfile = DetectFeatures(ctx)
	ResolveRegisteredTypeConsts(ctx.Registry)
	ResolveHandlerMethods(ctx)
	ResolveTransportAdapters(ctx)

	return ctx, func() {}
}

// BuildContextWithTypes creates an AnalysisContext with REAL type information.
// It writes sources to temp files and runs go/packages.Load with NeedTypes |
// NeedTypesInfo, producing fully populated TypesInfo maps. Use this instead of
// BuildContextFromSource when testing type-aware rules (C023, F010, etc.)
// that check gf.Pkg.TypesInfo.Types, .Defs, .Uses, .Selections.
//
// The returned cleanup function removes the temp directory.
func BuildContextWithTypes(
	t *testing.T,
	goVersion string,
	sources map[string]string,
) (*AnalysisContext, func()) {
	t.Helper()

	dir := t.TempDir()

	// Write a minimal go.mod so go/packages can resolve the module.
	if goVersion == "" {
		goVersion = "1.26"
	}

	modContent := "module test.example\n\ngo " + goVersion + "\n"
	if err := os.WriteFile(dir+"/go.mod", []byte(modContent), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	for filename, content := range sources {
		fullPath := dir + "/" + filename
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", fullPath, err)
		}
	}

	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedTypesInfo |
			packages.NeedSyntax | packages.NeedFiles,
		Dir:        dir,
		BuildFlags: []string{"-tags=goexperiment.jsonv2"},
	}

	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		t.Fatalf("packages.Load: %v", err)
	}

	// Use the FileSet the loaded packages were parsed into — a fresh empty
	// FileSet resolves every position to zero, silently breaking
	// position-based finding builders in tests.
	fset := token.NewFileSet()
	if len(pkgs) > 0 && pkgs[0].Fset != nil {
		fset = pkgs[0].Fset
	}

	ctx := &AnalysisContext{
		Fset:     fset,
		Registry: NewCQRSRegistry(),
	}

	for _, pkg := range pkgs {
		if len(pkg.Errors) > 0 {
			t.Fatalf("package %s has errors: %v", pkg.PkgPath, pkg.Errors[0])
		}

		for _, file := range pkg.Syntax {
			pos := fset.Position(file.Pos())
			filename := pos.Filename

			gf := &GoFile{
				Path:   filename,
				Pkg:    pkg,
				AST:    file,
				IsTest: strings.HasSuffix(filename, "_test.go"),
			}

			ctx.GoFiles = append(ctx.GoFiles, gf)
			if !gf.IsTest {
				scanFile(ctx, gf)
			}
		}
	}

	ctx.FeatureProfile = DetectFeatures(ctx)
	ResolveRegisteredTypeConsts(ctx.Registry)
	ResolveHandlerMethods(ctx)
	ResolveTransportAdapters(ctx)

	return ctx, func() {}
}

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

	return ctx, func() {}
}

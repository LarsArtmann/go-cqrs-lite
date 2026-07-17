package analyzer

import (
	"go/parser"
	"go/token"
	"go/types"
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

	return ctx
}

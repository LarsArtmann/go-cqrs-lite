package correctness

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/lintutil"
)

// Detects fmt.Errorf without %w in files that import go-cqrs-lite modules.
// While D006 catches this globally at info severity, C025 escalates to warning
// for CQRS consumer code — bare fmt.Errorf in event/command/query handlers
// loses error classification and breaks the 6-family error taxonomy.
//
// C025: fmt.Errorf without %w in CQRS error paths.
//
//nolint:ireturn // factory returns public interface
func NewC025Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C025-bare-errorf-in-cqrs",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				if !fileImportsCQRS(gf.AST) {
					continue
				}

				sentinels := collectPkgLevelVarCalls(ctx)

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}

					if !isFmtErrorf(call) || hasWrapVerb(call) {
						return true
					}

					if sentinels[call.Pos()] {
						return true
					}

					reportBareErrorf(ctx, &findings, call)

					return true
				})
			}

			return findings, nil
		},
	)
}

// fileImportsCQRS returns true if the file's import declarations include
// any go-cqrs-lite module path.
func fileImportsCQRS(file *ast.File) bool {
	for _, imp := range file.Imports {
		if imp == nil || imp.Path == nil {
			continue
		}

		path := strings.Trim(imp.Path.Value, `"`)
		if analyzer.IsCQRSModulePath(path) {
			return true
		}
	}

	return false
}

func isFmtErrorf(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	ident, ok := sel.X.(*ast.Ident)

	return ok && ident.Name == "fmt" && sel.Sel.Name == "Errorf"
}

func hasWrapVerb(call *ast.CallExpr) bool {
	if len(call.Args) == 0 {
		return false
	}

	lit, ok := call.Args[0].(*ast.BasicLit)
	if !ok {
		return true // Non-literal format string — can't analyze statically.
	}

	return strings.Contains(lit.Value, "%w")
}

// collectPkgLevelVarCalls returns the set of CallExpr positions that are
// the initializer of a package-level var declaration (sentinel-error pattern).
func collectPkgLevelVarCalls(ctx *analyzer.AnalysisContext) map[token.Pos]bool {
	positions := make(map[token.Pos]bool)

	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		for _, decl := range gf.AST.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.VAR {
				continue
			}

			for _, spec := range genDecl.Specs {
				valSpec, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}

				for _, val := range valSpec.Values {
					if c, ok := val.(*ast.CallExpr); ok {
						positions[c.Pos()] = true
					}
				}
			}
		}
	}

	return positions
}

func reportBareErrorf(
	ctx *analyzer.AnalysisContext,
	findings *[]finding.Finding,
	call *ast.CallExpr,
) {
	pos := ctx.Fset.Position(call.Pos())

	f, err := finding.NewBuilder(
		"C025", toolName,
		fmt.Sprintf(
			"fmt.Errorf without %%w in CQRS code at %s — "+
				"loses error classification, breaks the 6-family taxonomy",
			pos.String(),
		),
		finding.SeverityWarning,
		finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
	).
		WithCategory(finding.CategoryCorrectness).
		WithConfidence(finding.ConfidenceMedium).
		WithSuggestion("Use fmt.Errorf(\"...: %w\", err) to wrap, or " +
			"errorfamily.WrapConflict/WrapTransient/etc. for classified errors").
		WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
		Build()
	lintutil.AppendBuild(findings, f, err)
}

package architecture

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/lintutil"
)

// importsPathSuffix reports whether any non-test file imports a path containing
// suffix. Works with AST-level import declarations so it is testable via
// analyzer.BuildContextFromSource.
func importsPathSuffix(ctx *analyzer.AnalysisContext, suffix string) bool {
	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		if fileImportsPath(gf, suffix) {
			return true
		}
	}

	return false
}

// fileImportsPath reports whether a single GoFile imports a path containing
// suffix. Used by per-module detectors (e.g. E009) that need to evaluate
// imports per-module rather than workspace-wide.
func fileImportsPath(gf *analyzer.GoFile, suffix string) bool {
	for _, imp := range gf.AST.Imports {
		if imp == nil || imp.Path == nil {
			continue
		}

		path := strings.Trim(imp.Path.Value, `"`)
		if strings.Contains(path, suffix) {
			return true
		}
	}

	return false
}

// fileImportsCustomHTTP reports whether the file imports net/http or a common
// third-party HTTP framework. Used by E009 to detect hand-rolled delivery
// layers outside the sanctioned modules (watermill/, go-sse, cqrs-htmx).
func fileImportsCustomHTTP(gf *analyzer.GoFile) bool {
	for _, imp := range gf.AST.Imports {
		if imp == nil || imp.Path == nil {
			continue
		}

		path := strings.Trim(imp.Path.Value, `"`)

		if path == "net/http" ||
			strings.Contains(path, "gin-gonic/gin") ||
			strings.Contains(path, "labstack/echo") ||
			strings.Contains(path, "go-chi/chi") ||
			strings.Contains(path, "gorilla/mux") ||
			strings.Contains(path, "gofiber/fiber") ||
			strings.Contains(path, "/httprouter") {
			return true
		}
	}

	return false
}

// firstFilePos returns the package declaration position of the first non-test
// file. Used as an anchor for project-level findings without a specific call site.
func firstFilePos(ctx *analyzer.AnalysisContext) (token.Position, bool) {
	return lintutil.FirstFilePos(ctx)
}

// singleFinding builds and returns a single finding with the common architecture
// defaults. Category is CategoryStructure, matching all existing E-series rules.
func singleFinding(
	ctx *analyzer.AnalysisContext,
	ruleID, message, suggestion string,
	pos token.Position,
	severity finding.Severity,
	confidence finding.Confidence,
) []finding.Finding {
	f, err := finding.NewBuilder(
		finding.RuleName(ruleID), toolName,
		message,
		severity,
		finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
	).
		WithCategory(finding.CategoryStructure).
		WithConfidence(confidence).
		WithSuggestion(suggestion).
		WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
		Build()
	if err != nil {
		return nil
	}

	return []finding.Finding{f}
}

// projectCallsImportPathBool is a boolean-only wrapper around
// projectCallsImportPath for use in compound boolean expressions.
func projectCallsImportPathBool(
	ctx *analyzer.AnalysisContext,
	importPathSubstr string,
	funcNames ...string,
) bool {
	_, ok := projectCallsImportPath(ctx, importPathSubstr, funcNames...)
	return ok
}

// projectCallsImportPath checks whether any non-test file calls any of funcNames
// on a receiver whose import path contains importPathSubstr. This resolves
// import aliases: `import es "go-cqrs-lite/event"` then `es.NewEvent(...)` is
// correctly matched even though the qualifier is "es", not "event".
func projectCallsImportPath(
	ctx *analyzer.AnalysisContext,
	importPathSubstr string,
	funcNames ...string,
) (token.Position, bool) {
	nameSet := make(map[string]bool, len(funcNames))
	for _, n := range funcNames {
		nameSet[n] = true
	}

	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		var hit *ast.CallExpr

		ast.Inspect(gf.AST, func(n ast.Node) bool {
			if hit != nil {
				return false
			}

			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			sel, ok := analyzer.SelectorFromExpr(call.Fun)
			if !ok {
				return true
			}

			if !nameSet[sel.Sel.Name] {
				return true
			}

			pkgIdent, ok := sel.X.(*ast.Ident)
			if !ok {
				return true
			}

			impPath, found := lintutil.QualifierToImportPath(gf.AST, pkgIdent.Name)
			if !found {
				return true
			}

			if strings.Contains(impPath, importPathSubstr) {
				hit = call
				return false
			}

			return true
		})

		if hit != nil {
			return ctx.Fset.Position(hit.Pos()), true
		}
	}

	return token.Position{}, false
}

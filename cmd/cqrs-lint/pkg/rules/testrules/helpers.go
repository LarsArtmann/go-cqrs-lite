package testrules

import (
	"go/ast"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/lintutil"
)

const toolName = lintutil.ToolName

// hasTestFiles returns true if any Go file in the context is a test file.
func hasTestFiles(ctx *analyzer.AnalysisContext) bool {
	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			return true
		}
	}

	return false
}

// fileImportsSubstr returns true if the AST file imports a path containing substr.
func fileImportsSubstr(file *ast.File, substr string) bool {
	for _, imp := range file.Imports {
		if imp == nil || imp.Path == nil {
			continue
		}

		path := strings.Trim(imp.Path.Value, `"`)
		if strings.Contains(path, substr) {
			return true
		}
	}

	return false
}

// anyFileImports checks if any Go file (test or production) imports a path
// containing substr.
func anyFileImports(ctx *analyzer.AnalysisContext, substr string) bool {
	for _, gf := range ctx.GoFiles {
		if fileImportsSubstr(gf.AST, substr) {
			return true
		}
	}

	return false
}

// anyProdFileImports checks only non-test files.
func anyProdFileImports(ctx *analyzer.AnalysisContext, substr string) bool {
	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		if fileImportsSubstr(gf.AST, substr) {
			return true
		}
	}

	return false
}

// testFileCallsPkgFunc checks if any test file contains a call of the form
// pkgName.funcName(...) — e.g. scenario.Given(...).
func testFileCallsPkgFunc(ctx *analyzer.AnalysisContext, pkgName, funcName string) bool {
	for _, gf := range ctx.GoFiles {
		if !gf.IsTest {
			continue
		}

		if astFileCallsPkgFunc(gf.AST, pkgName, funcName) {
			return true
		}
	}

	return false
}

// testFileCallsMethod checks if any test file calls a method named methodName
// on any receiver — e.g. .ThenError(...).
func testFileCallsMethod(ctx *analyzer.AnalysisContext, methodName string) bool {
	for _, gf := range ctx.GoFiles {
		if !gf.IsTest {
			continue
		}

		found := false

		ast.Inspect(gf.AST, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			sel, ok := analyzer.SelectorFromExpr(call.Fun)
			if !ok {
				return true
			}

			if sel.Sel.Name == methodName {
				found = true

				return false
			}

			return true
		})

		if found {
			return true
		}
	}

	return false
}

// testFilesCallBoth returns true if the union of test files contains calls to
// both methodName1 and methodName2 (not necessarily in the same file).
func testFilesCallBoth(ctx *analyzer.AnalysisContext, method1, method2 string) bool {
	return testFileCallsMethod(ctx, method1) && testFileCallsMethod(ctx, method2)
}

// astFileCallsPkgFunc returns true if the AST contains a call pkgName.funcName(...).
func astFileCallsPkgFunc(file *ast.File, pkgName, funcName string) bool {
	found := false

	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		sel, ok := analyzer.SelectorFromExpr(call.Fun)
		if !ok {
			return true
		}

		pkgIdent, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}

		if pkgIdent.Name == pkgName && sel.Sel.Name == funcName {
			found = true

			return false
		}

		return true
	})

	return found
}

// projectFinding builds a single project-level finding positioned at go.mod.
func projectFinding(
	ruleID, message, suggestion string,
	confidence finding.Confidence,
	ctx *analyzer.AnalysisContext,
) finding.Finding {
	f, _ := finding.NewBuilder(
		finding.RuleName(ruleID), toolName,
		message, finding.SeverityInfo,
		finding.Pos(finding.FilePath(ctx.ProjectRoot+"/go.mod"), 1, 1),
	).
		WithCategory(finding.CategoryTesting).
		WithConfidence(confidence).
		WithSuggestion(suggestion).
		Build()

	return f
}

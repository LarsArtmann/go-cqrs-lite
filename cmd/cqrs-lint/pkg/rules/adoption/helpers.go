package adoption

import (
	"go/ast"
	"go/token"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/lintutil"
)

// isFoldConstructor reports whether fnName is a metaengine fold
// constructor: the deprecated payload-only pair (On/OnTyped) and the
// record-aware default pair (OnRecord/OnRecordTyped).
func isFoldConstructor(fnName string) bool {
	switch fnName {
	case "On", "OnTyped", "OnRecord", "OnRecordTyped":
		return true
	default:
		return false
	}
}

// projectHasCall reports whether any non-test file calls pkgName.funcName.
func projectHasCall(ctx *analyzer.AnalysisContext, pkgName, funcName string) bool {
	return projectHasCallAny(ctx, pkgName, funcName)
}

// projectHasCallAny reports whether any non-test file calls any of funcNames
// on pkgName.
func projectHasCallAny(ctx *analyzer.AnalysisContext, pkgName string, funcNames ...string) bool {
	return hasCallIn(ctx.GoFiles, pkgName, funcNames...)
}

// firstCallPos returns the position of the first call to pkgName.funcName
// in any non-test file.
func firstCallPos(
	ctx *analyzer.AnalysisContext,
	pkgName, funcName string,
) (token.Position, bool) {
	return firstCallPosIn(ctx.Fset, ctx.GoFiles, pkgName, funcName)
}

// firstFilePos returns the package declaration position of the first non-test
// file. Used as an anchor for project-level findings without a specific call site.
func firstFilePos(ctx *analyzer.AnalysisContext) (token.Position, bool) {
	return lintutil.FirstFilePos(ctx)
}

// firstFuncDeclPos returns the position of the first function or method
// declaration whose name starts with prefix.
func firstFuncDeclPos(
	ctx *analyzer.AnalysisContext,
	prefix string,
) (token.Position, bool) {
	return firstFuncDeclPosIn(ctx.Fset, ctx.GoFiles, prefix)
}

// astInspectCalls walks the AST and calls fn for every *ast.CallExpr.
// Stops early if fn returns false.
func astInspectCalls(root ast.Node, fn func(*ast.CallExpr) bool) {
	ast.Inspect(root, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		return fn(call)
	})
}

// singleInfoFinding builds and returns a single info-level finding with the
// common F-series defaults: CategoryBestPractice, FixStrategySuggest.
func singleInfoFinding(
	ctx *analyzer.AnalysisContext,
	ruleID, message, suggestion string,
	pos token.Position,
	confidence finding.Confidence,
) []finding.Finding {
	f, err := finding.NewBuilder(
		finding.RuleName(ruleID), toolName,
		message,
		finding.SeverityInfo,
		finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
	).
		WithCategory(finding.CategoryBestPractice).
		WithConfidence(confidence).
		WithFixStrategy(finding.FixStrategySuggest).
		WithSuggestion(suggestion).
		WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
		Build()
	if err != nil {
		return nil
	}

	return []finding.Finding{f}
}

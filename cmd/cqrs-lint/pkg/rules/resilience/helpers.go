package resilience

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/larsartmann/go-finding"
	"golang.org/x/tools/go/packages"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// singleInfoFinding builds a single info-level finding with the common
// resilience defaults: CategoryBestPractice, FixStrategySuggest.
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

// isBusName reports whether name looks like a bus or dispatcher variable.
func isBusName(name string) bool {
	name = strings.ToLower(name)

	return strings.HasSuffix(name, "bus") ||
		strings.HasSuffix(name, "dispatcher") ||
		strings.HasSuffix(name, "disp")
}

func hasBusMethodCall(ctx *analyzer.AnalysisContext, varName string) bool {
	busMethodNames := map[string]bool{
		"Use":           true,
		"UsePublish":    true,
		"Publish":       true,
		"Subscribe":     true,
		"SubscribeAll":  true,
		"Handle":        true,
		"Dispatch":      true,
		"RegisterTyped": true,
		"RegisterQuery": true,
	}

	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		found := false

		ast.Inspect(gf.AST, func(n ast.Node) bool {
			if found {
				return false
			}

			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}

			ident, ok := sel.X.(*ast.Ident)
			if !ok || ident.Name != varName {
				return true
			}

			if busMethodNames[sel.Sel.Name] {
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

// hasMiddlewareKeyword scans all non-test files for x.Use(...) or x.UsePublish(...)
// calls where any argument or the method name contains keyword (case-insensitive).
func hasMiddlewareKeyword(ctx *analyzer.AnalysisContext, varName, keyword string) bool {
	keywordLower := strings.ToLower(keyword)

	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		found := false

		ast.Inspect(gf.AST, func(n ast.Node) bool {
			if found {
				return false
			}

			exprStmt, ok := n.(*ast.ExprStmt)
			if !ok {
				return true
			}

			call, ok := exprStmt.X.(*ast.CallExpr)
			if !ok {
				return true
			}

			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}

			ident, ok := sel.X.(*ast.Ident)
			if !ok || ident.Name != varName {
				return true
			}

			methodName := strings.ToLower(sel.Sel.Name)
			if methodName != "use" && methodName != "usepublish" &&
				methodName != "usemiddleware" && methodName != "addmiddleware" {
				return true
			}

			for _, arg := range call.Args {
				if callContainsKeyword(arg, keywordLower) {
					found = true
					return false
				}
			}

			return true
		})

		if found {
			return true
		}
	}

	return false
}

// callContainsKeyword checks whether an expression contains an identifier
// matching keyword (case-insensitive).
func callContainsKeyword(expr ast.Expr, keyword string) bool {
	found := false

	ast.Inspect(expr, func(n ast.Node) bool {
		if found {
			return false
		}

		if ident, ok := n.(*ast.Ident); ok {
			if strings.Contains(strings.ToLower(ident.Name), keyword) {
				found = true
				return false
			}
		}

		return true
	})

	return found
}

// findBusVariables returns a map of bus/dispatcher variable names to their
// position, collected from assignment statements in non-test files.
// Only variables that also have CQRS bus method calls (Use, Publish, etc.)
// are included — this filters out lookalike names (errorBus.Notify).
func findBusVariables(ctx *analyzer.AnalysisContext) map[string]token.Position {
	buses := make(map[string]token.Position)

	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		ast.Inspect(gf.AST, func(n ast.Node) bool {
			assign, ok := n.(*ast.AssignStmt)
			if !ok {
				return true
			}

			for _, lhs := range assign.Lhs {
				if ident, ok := lhs.(*ast.Ident); ok {
					if isBusName(ident.Name) && hasBusMethodCall(ctx, ident.Name) {
						buses[ident.Name] = ctx.Fset.Position(ident.Pos())
					}
				}
			}

			return true
		})
	}

	return buses
}

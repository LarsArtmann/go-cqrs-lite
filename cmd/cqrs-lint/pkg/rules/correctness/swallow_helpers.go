package correctness

import (
	"go/ast"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

func isPayloadCall(expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}

	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	return sel.Sel.Name == "Payload"
}

func isLikelyDecider(fn *ast.FuncDecl) bool {
	name := fn.Name.Name

	return name == "decide" || name == "Decide" ||
		strings.HasPrefix(name, "decide") ||
		strings.HasPrefix(name, "Decide")
}

func isFloat64(expr ast.Expr) bool {
	id, ok := expr.(*ast.Ident)

	return ok && id.Name == "float64"
}

func inspectForSwallowedError(
	ctx *analyzer.AnalysisContext,
	fn *ast.FuncDecl,
	findings *[]finding.Finding,
) bool {
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		assign, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}

		if len(assign.Lhs) == 0 {
			return true
		}

		for _, lhs := range assign.Lhs {
			id, ok := lhs.(*ast.Ident)
			if !ok || id.Name != "_" {
				continue
			}

			for _, rhs := range assign.Rhs {
				call, ok := rhs.(*ast.CallExpr)
				if !ok {
					continue
				}

				_, ok = call.Fun.(*ast.SelectorExpr)
				if !ok {
					continue
				}

				callStr := analyzer.ExprString(call.Fun)
				if strings.Contains(callStr, "Decode") || strings.Contains(callStr, "Unmarshal") {
					pos := ctx.Fset.Position(assign.Pos())

					f, err := finding.NewBuilder(
						"C010", toolName,
						"Error from decode/unmarshal call is discarded — use the error or handle it",
						finding.SeverityWarning,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryCorrectness).
						WithConfidence(finding.ConfidenceHigh).
						WithSuggestion("Check the error return: `if err != nil { return state, fmt.Errorf(\"decode: %w\", err) }`").
						WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
						Build()
					if err == nil {
						*findings = append(*findings, f)
					}
				}
			}
		}

		return true
	})

	return true
}

func findBodyParam(fn *ast.FuncDecl) string {
	if fn.Type == nil || fn.Type.Params == nil {
		return ""
	}

	for _, param := range fn.Type.Params.List {
		if ft, ok := param.Type.(*ast.FuncType); ok {
			if ft.Params != nil && len(ft.Params.List) > 0 {
				if len(param.Names) > 0 {
					return param.Names[0].Name
				}
			}
		}
	}

	return ""
}

func ignoresBodyError(fn *ast.FuncDecl, bodyVar string) bool {
	calledWithoutCheck := false

	ast.Inspect(fn, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		if id, ok := call.Fun.(*ast.Ident); ok && id.Name == bodyVar {
			assign := findContainingAssignStmt(fn, call)
			if assign == nil {
				calledWithoutCheck = true

				return false
			}

			for _, lhs := range assign.Lhs {
				if id, ok := lhs.(*ast.Ident); ok && (id.Name == "_" || id.Name == "") {
					calledWithoutCheck = true
				}
			}
		}

		return true
	})

	return calledWithoutCheck
}

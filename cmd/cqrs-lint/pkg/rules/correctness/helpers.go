package correctness

import (
	"go/ast"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

func returnsError(fn *ast.FuncDecl) bool {
	if fn.Type == nil || fn.Type.Results == nil {
		return false
	}

	for _, field := range fn.Type.Results.List {
		if id, ok := field.Type.(*ast.Ident); ok && id.Name == "error" {
			return true
		}
	}

	return false
}

func findBeginTxVar(fn *ast.FuncDecl) string {
	var txVar string

	ast.Inspect(fn, func(n ast.Node) bool {
		if txVar != "" {
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

		if sel.Sel.Name != "BeginTx" && sel.Sel.Name != "Begin" {
			return true
		}

		if assignStmt := findContainingAssignStmt(fn, call); assignStmt != nil {
			if len(assignStmt.Lhs) > 0 {
				if id, ok := assignStmt.Lhs[0].(*ast.Ident); ok {
					txVar = id.Name
				}
			}
		}

		return true
	})

	return txVar
}

func findContainingAssignStmt(fn *ast.FuncDecl, target ast.Node) *ast.AssignStmt {
	var result *ast.AssignStmt

	ast.Inspect(fn, func(n ast.Node) bool {
		if result != nil {
			return false
		}

		assign, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}

		for _, rhs := range assign.Rhs {
			if containsNode(rhs, target) {
				result = assign

				return false
			}
		}

		return true
	})

	return result
}

func containsNode(parent, target ast.Node) bool {
	if parent == target {
		return true
	}

	found := false

	ast.Inspect(parent, func(n ast.Node) bool {
		if n == target {
			found = true

			return false
		}

		return !found
	})

	return found
}

func hasCommitCall(fn *ast.FuncDecl, txVar string) bool {
	found := false

	ast.Inspect(fn, func(n ast.Node) bool {
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

		if id, ok := sel.X.(*ast.Ident); ok && id.Name == txVar && sel.Sel.Name == "Commit" {
			found = true
		}

		return true
	})

	return found
}

func hasDeferCommit(fn *ast.FuncDecl, txVar string) bool {
	found := false

	ast.Inspect(fn, func(n ast.Node) bool {
		if found {
			return false
		}

		deferStmt, ok := n.(*ast.DeferStmt)
		if !ok {
			return true
		}

		call := deferStmt.Call
		if call == nil {
			return true
		}

		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		if id, ok := sel.X.(*ast.Ident); ok && id.Name == txVar && sel.Sel.Name == "Commit" {
			found = true
		}

		return true
	})

	return found
}

func hasReturnNil(fn *ast.FuncDecl) bool {
	found := false

	ast.Inspect(fn, func(n ast.Node) bool {
		if found {
			return false
		}

		ret, ok := n.(*ast.ReturnStmt)
		if !ok {
			return true
		}

		for _, result := range ret.Results {
			if id, ok := result.(*ast.Ident); ok && id.Name == "nil" {
				found = true

				return false
			}
		}

		return true
	})

	return found
}

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
		strings.HasPrefix(name, "Decide") ||
		strings.Contains(name, "decide") ||
		strings.Contains(name, "Decide")
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

package correctness

import (
	"go/ast"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/lintutil"
)

func isPayloadCall(expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}

	sel, ok := analyzer.SelectorFromExpr(call.Fun)
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

	return ok && (id.Name == "float64" || id.Name == "float32")
}

func inspectForSwallowedError(
	ctx *analyzer.AnalysisContext,
	fn *ast.FuncDecl,
	findings *[]finding.Finding,
) bool {
	inspectBodyForSwallowedError(ctx, fn.Body, findings)
	return true
}

// inspectBodyForSwallowedError scans a function body for swallowed
// decode/unmarshal errors (p, _ := event.DecodePayloadAuto[T](evt)).
func inspectBodyForSwallowedError(
	ctx *analyzer.AnalysisContext,
	body *ast.BlockStmt,
	findings *[]finding.Finding,
) {
	ast.Inspect(body, func(n ast.Node) bool {
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

				_, ok = analyzer.SelectorFromExpr(call.Fun)
				if !ok {
					continue
				}

				callStr := analyzer.ExprString(call.Fun)
				if cat := swallowedCallCategory(callStr); cat != "" {
					pos := ctx.Fset.Position(assign.Pos())

					f, err := finding.NewBuilder(
						"C010", toolName,
						"Error from "+cat+" call is discarded — use the error or handle it",
						finding.SeverityWarning,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryCorrectness).
						WithConfidence(finding.ConfidenceHigh).
						WithSuggestion("Check the error return: `if err != nil { return state, fmt.Errorf(\"" + cat + ": %w\", err) }`").
						WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
						Build()
					lintutil.AppendBuild(findings, f, err)
				}
			}
		}

		return true
	})
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

// swallowedCallCategory returns a human-readable category string if the call
// expression is one whose error return should never be silently discarded.
// Returns "" if the call does not match any known swallowable pattern.
//
// Covers decode/unmarshal operations (original C010 scope) and SQL operations
// (item 169 extension: Exec, Query, QueryRow, Scan, Get, Select).
func swallowedCallCategory(callStr string) string {
	switch {
	case strings.Contains(callStr, "Decode"), strings.Contains(callStr, "Unmarshal"):
		return "decode/unmarshal"
	case strings.Contains(callStr, ".Exec"),
		strings.Contains(callStr, ".Query"),
		strings.Contains(callStr, ".Scan"),
		strings.Contains(callStr, ".Get"),
		strings.Contains(callStr, ".Select"):
		return "SQL"
	default:
		return ""
	}
}

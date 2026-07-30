package correctness

import (
	"context"
	"fmt"
	"go/ast"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// Detects fold functions whose switch default case returns nil error,
// OR whose if-statement checks evt.Type() != X and returns nil error
// (same bug, different syntax).
//
//nolint:ireturn // factory returns public interface
func NewC003Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C003-silent-unknown-event-fold",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, fold := range ctx.Registry.Folds {
				if fold.HasSwitch && fold.HasDefault && fold.DefaultNil {
					f, err := finding.NewBuilder(
						"C003", toolName,
						fmt.Sprintf("Fold %s silently ignores unknown event types in default case", fold.FuncName),
						finding.SeverityError,
						finding.Pos(finding.FilePath(fold.File), fold.Pos.Line, fold.Pos.Column),
					).
						WithCategory(finding.CategoryCorrectness).
						WithConfidence(finding.ConfidenceHigh).
						WithFixStrategy(finding.FixStrategyDirect).
						WithSuggestion("Return an error in the default case: return state, fmt.Errorf(\"fold: unknown event type: %s\", evt.Type())").
						WithBeforeCode("return state, nil").
						WithAfterCode(`return state, fmt.Errorf("fold: unknown event type: %s", evt.Type())`).
						WithSnippet(ctx.SourceLine(fold.File, fold.Pos.Line)).
						Build()
					if err != nil {
						continue
					}

					findings = append(findings, f)
				}

				// Also check for the if-statement variant of the same bug:
				// if evt.Type() != "expected" { return state, nil }
				if foldHasSilentIfStmt(ctx, fold) {
					f, err := finding.NewBuilder(
						"C003", toolName,
						fmt.Sprintf("Fold %s silently ignores unknown event types via if-statement", fold.FuncName),
						finding.SeverityError,
						finding.Pos(finding.FilePath(fold.File), fold.Pos.Line, fold.Pos.Column),
					).
						WithCategory(finding.CategoryCorrectness).
						WithConfidence(finding.ConfidenceMedium).
						WithFixStrategy(finding.FixStrategyDirect).
						WithSuggestion("Return an error for unknown event types: return state, fmt.Errorf(\"fold: unknown event type: %s\", evt.Type())").
						WithSnippet(ctx.SourceLine(fold.File, fold.Pos.Line)).
						Build()
					if err == nil {
						findings = append(findings, f)
					}
				}
			}

			return findings, nil
		},
	)
}

// foldHasSilentIfStmt checks whether a fold function body contains an
// if-statement that compares the event's Type() with != and returns nil
// error in the body — the if-statement equivalent of the switch-default-nil.
func foldHasSilentIfStmt(ctx *analyzer.AnalysisContext, fold analyzer.FoldInfo) bool {
	name := fold.FuncName
	if idx := strings.LastIndex(name, "."); idx >= 0 {
		name = name[idx+1:]
	}

	for _, gf := range ctx.GoFiles {
		if gf.Path != fold.File || gf.IsTest {
			continue
		}

		found := false

		ast.Inspect(gf.AST, func(n ast.Node) bool {
			if found {
				return false
			}

			fn, ok := n.(*ast.FuncDecl)
			if !ok || fn.Name == nil || fn.Name.Name != name {
				return true
			}

			ast.Inspect(fn.Body, func(inner ast.Node) bool {
				if found {
					return false
				}

				ifStmt, ok := inner.(*ast.IfStmt)
				if !ok {
					return true
				}

				if isEventTypeCheck(ifStmt.Cond) && bodyReturnsNilError(ifStmt.Body) {
					found = true
					return false
				}

				return true
			})

			return true
		})

		if found {
			return true
		}
	}

	return false
}

// isEventTypeCheck returns true if the expression is a binary comparison
// involving a .Type() call (e.g., evt.Type() != "X").
func isEventTypeCheck(expr ast.Expr) bool {
	bin, ok := expr.(*ast.BinaryExpr)
	if !ok {
		return false
	}

	if bin.Op.String() != "!=" && bin.Op.String() != "==" {
		return false
	}

	return exprCallsType(bin.X) || exprCallsType(bin.Y)
}

func exprCallsType(expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}

	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	return sel.Sel.Name == "Type"
}

// bodyReturnsNilError returns true if the block contains a return statement
// where the last returned value is the nil identifier.
func bodyReturnsNilError(body *ast.BlockStmt) bool {
	if body == nil {
		return false
	}

	for _, stmt := range body.List {
		ret, ok := stmt.(*ast.ReturnStmt)
		if !ok || len(ret.Results) == 0 {
			continue
		}

		last := ret.Results[len(ret.Results)-1]
		ident, ok := last.(*ast.Ident)

		return ok && ident.Name == "nil"
	}

	return false
}

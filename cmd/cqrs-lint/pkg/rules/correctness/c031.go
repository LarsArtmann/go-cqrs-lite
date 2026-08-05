package correctness

import (
	"context"
	"go/ast"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/lintutil"
)

// C031: Error swallowing in command handlers.
// Detects `if err != nil { return nil }` or `if err != nil { return }` in
// function literals passed to RegisterTyped/RegisterQuery. When a handler
// checks an error but returns nil (success), the command appears to succeed
// while actually failing — events are silently not emitted, state is
// inconsistent, and the caller gets no feedback.
//
//nolint:ireturn // factory returns public interface
func NewC031Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C031-error-swallow-in-handler",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}

					sel, ok := analyzer.SelectorFromExpr(call.Fun)
					if !ok {
						return true
					}

					if sel.Sel.Name != "RegisterTyped" && sel.Sel.Name != "RegisterQuery" {
						return true
					}

					for _, arg := range call.Args {
						lit, ok := arg.(*ast.FuncLit)
						if !ok || lit.Body == nil {
							continue
						}

						scanHandlerBodyForSwallowedError(ctx, lit.Body, &findings)
					}

					return true
				})
			}

			return findings, nil
		},
	)
}

// scanHandlerBodyForSwallowedError finds `if err != nil { return nil }` or
// `if err != nil { return }` patterns where an error is checked but the
// handler returns success (nil) instead of propagating the error.
func scanHandlerBodyForSwallowedError(
	ctx *analyzer.AnalysisContext,
	body *ast.BlockStmt,
	findings *[]finding.Finding,
) {
	ast.Inspect(body, func(n ast.Node) bool {
		ifStmt, ok := n.(*ast.IfStmt)
		if !ok {
			return true
		}

		if !isErrNotNilCheck(ifStmt.Cond) {
			return true
		}

		for _, stmt := range ifStmt.Body.List {
			if isSwallowingReturn(stmt) {
				pos := ctx.Fset.Position(stmt.Pos())

				f, err := finding.NewBuilder(
					"C031",
					toolName,
					"Error is checked but handler returns nil — the command/query appears successful when it actually failed",
					finding.SeverityWarning,
					finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
				).
					WithCategory(finding.CategoryCorrectness).
					WithConfidence(finding.ConfidenceHigh).
					WithFixStrategy(finding.FixStrategySuggest).
					WithSuggestion("Return the error: `return fmt.Errorf(\"handler: %w\", err)`").
					WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
					Build()
				lintutil.AppendBuild(findings, f, err)

				break
			}
		}

		return true
	})
}

// isErrNotNilCheck reports whether expr is `err != nil` or `nil != err`.
func isErrNotNilCheck(expr ast.Expr) bool {
	bin, ok := expr.(*ast.BinaryExpr)
	if !ok || bin.Op.String() != "!=" {
		return false
	}

	left, leftIsIdent := bin.X.(*ast.Ident)
	right, rightIsIdent := bin.Y.(*ast.Ident)

	if leftIsIdent && left.Name == "err" {
		if _, ok := bin.Y.(*ast.Ident); ok && right.Name == "nil" {
			return true
		}
	}

	if rightIsIdent && right.Name == "err" {
		if _, ok := bin.X.(*ast.Ident); ok && left.Name == "nil" {
			return true
		}
	}

	return false
}

// isSwallowingReturn reports whether stmt is `return nil`, `return nil, nil`,
// or bare `return` inside an error-check block — the error is swallowed.
//
// For multi-value returns like `return nil, err` (the canonical (any, error)
// handler pattern), the error IS propagated via the second value, so this
// returns false. Only returns where ALL values are nil (or bare returns)
// indicate a swallowed error.
func isSwallowingReturn(stmt ast.Stmt) bool {
	ret, ok := stmt.(*ast.ReturnStmt)
	if !ok {
		return false
	}

	if len(ret.Results) == 0 {
		return true
	}

	for _, result := range ret.Results {
		if !isNilLiteral(result) {
			return false
		}
	}

	return true
}

func isNilLiteral(expr ast.Expr) bool {
	id, ok := expr.(*ast.Ident)
	return ok && id.Name == "nil"
}

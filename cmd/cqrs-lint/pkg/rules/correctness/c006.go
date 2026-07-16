package correctness

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// Detects event.Version(x.Int()+1) instead of x.Increment().
func NewC006Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C006-manual-version-arithmetic",
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

					sel, ok := call.Fun.(*ast.SelectorExpr)
					if !ok {
						return true
					}

					pkgIdent, ok := sel.X.(*ast.Ident)
					if !ok || pkgIdent.Name != "event" || sel.Sel.Name != "Version" {
						return true
					}

					if len(call.Args) != 1 {
						return true
					}

					binOp, ok := call.Args[0].(*ast.BinaryExpr)
					if !ok || binOp.Op != token.ADD {
						return true
					}

					leftCall, ok := binOp.X.(*ast.CallExpr)
					if !ok {
						return true
					}

					leftSel, ok := leftCall.Fun.(*ast.SelectorExpr)
					if !ok || leftSel.Sel.Name != "Int" {
						return true
					}

					rightLit, ok := binOp.Y.(*ast.BasicLit)
					if !ok || rightLit.Value != "1" {
						return true
					}

					pos := ctx.Fset.Position(call.Pos())
					versionVar := analyzer.ExprString(leftSel.X)
					oldExpr := fmt.Sprintf("event.Version(%s.Int()+1)", versionVar)
					newExpr := versionVar + ".Increment()"

					f, err := finding.NewBuilder(
						"C006", toolName,
						"Manual version arithmetic — use Version.Increment() instead",
						finding.SeverityWarning,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryCorrectness).
						WithConfidence(finding.ConfidenceHigh).
						WithFixStrategy(finding.FixStrategyDirect).
						WithSuggestion(fmt.Sprintf("Replace %s with %s", oldExpr, newExpr)).
						WithBeforeCode(oldExpr).
						WithAfterCode(newExpr).
						WithMetadata(map[string]string{
							"oldExpr": oldExpr,
							"newExpr": newExpr,
						}).
						WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
						Build()
					if err != nil {
						return true
					}

					findings = append(findings, f)

					return true
				})
			}

			return findings, nil
		},
	)
}

package correctness

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// Detects manual event version arithmetic (+1) instead of Version.Increment().
//
// Catches three real-world patterns:
//   - event.Version(x.Int()+1) — should be x.Increment() (auto-fixable)
//   - event.Version(varName+1) — should use varName.Increment() (suggest)
//   - event.NewEvent(..., ver+1, ...) — bare arithmetic in version position (suggest)
//
//nolint:ireturn // factory returns public interface
func NewC006Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C006-manual-version-arithmetic",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding
			seen := make(map[token.Pos]bool)

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

					pkgIdent, ok := sel.X.(*ast.Ident)
					if !ok || pkgIdent.Name != "event" {
						return true
					}

					// Pattern 1 & 2: event.Version(arg) where arg involves +1.
					if sel.Sel.Name == "Version" && len(call.Args) == 1 {
						if binOp, ok := call.Args[0].(*ast.BinaryExpr); ok && binOp.Op == token.ADD {
							if rightLit, ok := binOp.Y.(*ast.BasicLit); ok && rightLit.Value == "1" {
								if !seen[call.Pos()] {
									seen[call.Pos()] = true
									findings = append(findings, c006Finding(ctx, call, binOp))
								}
							}
						}
					}

					// Pattern 3: event.NewEvent / event.New with bare +1 in the
					// version position (4th argument, index 3).
					if (sel.Sel.Name == "NewEvent" || sel.Sel.Name == "New") && len(call.Args) >= 4 {
						if binOp, ok := call.Args[3].(*ast.BinaryExpr); ok && binOp.Op == token.ADD {
							if rightLit, ok := binOp.Y.(*ast.BasicLit); ok && rightLit.Value == "1" {
								if !seen[binOp.Pos()] {
									seen[binOp.Pos()] = true
									findings = append(findings, c006Finding(ctx, binOp, binOp))
								}
							}
						}
					}

					return true
				})
			}

			return findings, nil
		},
	)
}

// c006Finding builds a finding for a manual version arithmetic expression.
func c006Finding(
	ctx *analyzer.AnalysisContext,
	node ast.Node,
	binOp *ast.BinaryExpr,
) finding.Finding {
	pos := ctx.Fset.Position(node.Pos())
	oldExpr := analyzer.ExprString(binOp)

	// If the left side is x.Int(), the fix is x.Increment() (auto-fixable).
	var newExpr, msg string
	strategy := finding.FixStrategySuggest

	if leftCall, ok := binOp.X.(*ast.CallExpr); ok {
		if leftSel, ok := analyzer.SelectorFromExpr(leftCall.Fun); ok && leftSel.Sel.Name == "Int" {
			versionVar := analyzer.ExprString(leftSel.X)
			oldExpr = fmt.Sprintf("event.Version(%s.Int()+1)", versionVar)
			newExpr = versionVar + ".Increment()"
			msg = "Manual version arithmetic — use Version.Increment() instead"
			strategy = finding.FixStrategyDirect
		}
	}

	if newExpr == "" {
		leftStr := analyzer.ExprString(binOp.X)
		newExpr = leftStr + ".Increment()"
		msg = "Manual version arithmetic (+1) — use Version.Increment() instead"
	}

	b := finding.NewBuilder(
		"C006", toolName,
		msg,
		finding.SeverityWarning,
		finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
	).
		WithCategory(finding.CategoryCorrectness).
		WithConfidence(finding.ConfidenceHigh).
		WithFixStrategy(strategy).
		WithSuggestion(fmt.Sprintf("Replace %s with %s", oldExpr, newExpr)).
		WithSnippet(ctx.SourceLine(pos.Filename, pos.Line))

	if strategy == finding.FixStrategyDirect {
		b = b.WithBeforeCode(oldExpr).WithAfterCode(newExpr).
			WithMetadata(map[string]string{"oldExpr": oldExpr, "newExpr": newExpr})
	}

	f, _ := b.Build()
	return f
}

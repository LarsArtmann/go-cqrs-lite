package performance

import (
	"context"
	"go/ast"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// Detects repo.Load / repository.Load calls inside SubscribeAll handlers.
// Each Load re-reads all prior events for the stream, making the projection
// O(N²) instead of O(N).
//
//nolint:ireturn // factory returns public interface
func NewP001Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"P001-load-in-subscribeall",
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

					method := sel.Sel.Name
					if method != "Subscribe" && method != "SubscribeAll" {
						return true
					}

					handlerExpr := analyzer.FindHandlerArg(call)
					if handlerExpr == nil {
						return true
					}

					var body *ast.BlockStmt

					switch h := handlerExpr.(type) {
					case *ast.FuncLit:
						body = h.Body
					case *ast.Ident:
						fn := analyzer.FindFuncDecl(ctx, h.Name)
						if fn != nil {
							body = fn.Body
						}
					case *ast.SelectorExpr:
						fn := analyzer.FindMethodDecl(ctx, h)
						if fn != nil {
							body = fn.Body
						}
					}

					if body == nil {
						return true
					}

					ast.Inspect(body, func(inner ast.Node) bool {
						innerCall, ok := inner.(*ast.CallExpr)
						if !ok {
							return true
						}

						innerSel, ok := analyzer.SelectorFromExpr(innerCall.Fun)
						if !ok {
							return true
						}

						if innerSel.Sel.Name != "Load" {
							return true
						}

						pkg := analyzer.SelectorPackage(innerSel)
						if pkg != "repo" && pkg != "repository" && pkg != "r" && pkg != "rep" {
							return true
						}

						pos := ctx.Fset.Position(innerCall.Pos())

						f, err := finding.NewBuilder(
							"P001", toolName,
							"repo.Load inside SubscribeAll handler — O(N²) replay, "+
								"each Load re-reads all prior events",
							finding.SeverityError,
							finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
						).
							WithCategory(finding.CategoryPerformance).
							WithConfidence(finding.ConfidenceHigh).
							WithFixStrategy(finding.FixStrategySuggest).
							WithSuggestion("Project directly from event payloads — use the event data, " +
								"don't re-load the aggregate").
							WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
							Build()
						if err != nil {
							return true
						}

						findings = append(findings, f)
						return true
					})

					return true
				})
			}

			return findings, nil
		},
	)
}

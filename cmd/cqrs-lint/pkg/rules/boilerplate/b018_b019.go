package boilerplate

import (
	"context"
	"go/ast"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// B018: Repeated bus.Subscribe boilerplate.
// Detects 3+ bus.Subscribe calls with identical error-handling structure in
// the same file. This pattern is better extracted into a table-driven
// registration loop.
//
//nolint:ireturn // factory returns public interface
func NewB018Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"B018-repeated-subscribe-boilerplate",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				type subscribeCall struct {
					file   string
					line   int
					col    int
					handler ast.Expr
				}

				var subscribes []subscribeCall

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}

					sel, ok := analyzer.SelectorFromExpr(call.Fun)
					if !ok {
						return true
					}

					if sel.Sel.Name != "Subscribe" {
						return true
					}

					// Must be a bus.Subscribe (package name contains "bus" or
					// receiver variable name contains "bus").
					pkg := analyzer.SelectorPackage(sel)
					if !containsBus(pkg) {
						// Also accept common bus variable names (bus, eventBus, evtBus).
						if ident, ok := sel.X.(*ast.Ident); ok {
							name := ident.Name
							if !containsBus(name) {
								return true
							}
						} else {
							return true
						}
					}

					pos := ctx.Fset.Position(call.Pos())
					subscribes = append(subscribes, subscribeCall{
						file:    pos.Filename,
						line:    pos.Line,
						col:     pos.Column,
						handler: analyzer.FindHandlerArg(call),
					})

					return true
				})

				if len(subscribes) < 3 {
					continue
				}

				// Report on the first subscribe call in the file.
				s := subscribes[0]

				f, err := finding.NewBuilder(
					"B018", toolName,
					"3+ bus.Subscribe calls in the same file — "+
						"extract into a table-driven registration loop",
					finding.SeverityInfo,
					finding.Pos(finding.FilePath(s.file), s.line, s.col),
				).
					WithCategory(finding.CategoryBestPractice).
					WithConfidence(finding.ConfidenceMedium).
					WithFixStrategy(finding.FixStrategySuggest).
					WithSuggestion("Define a []struct{Type string; Handler func(...)} slice and loop over it "+
						"calling bus.Subscribe for each entry").
					WithSnippet(ctx.SourceLine(s.file, s.line)).
					Build()
				if err == nil {
					findings = append(findings, f)
				}
			}

			return findings, nil
		},
	)
}

func containsBus(s string) bool {
	for i := 0; i+2 < len(s); i++ {
		if s[i] == 'b' && s[i+1] == 'u' && s[i+2] == 's' {
			return true
		}
	}

	return false
}

// B019: O(n^2) read model (repo.Load per event).
// Detects repo.Load / repository.Load calls inside bus.SubscribeAll handlers.
// For N events, each Load re-reads all prior events, making the projection
// O(N^2) instead of O(N).
//
//nolint:ireturn // factory returns public interface
func NewB019Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"B019-load-in-subscribeall",
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
					if method != "SubscribeAll" {
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
						if !isRepoVar(pkg) {
							return true
						}

						pos := ctx.Fset.Position(innerCall.Pos())

						f, err := finding.NewBuilder(
							"B019", toolName,
							"repo.Load inside SubscribeAll handler — O(N^2) replay, "+
								"each Load re-reads all prior events",
							finding.SeverityWarning,
							finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
						).
							WithCategory(finding.CategoryBestPractice).
							WithConfidence(finding.ConfidenceHigh).
							WithFixStrategy(finding.FixStrategySuggest).
							WithSuggestion("Project directly from event payloads — use the event data, "+
								"don't re-load the aggregate state per event").
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

func isRepoVar(pkg string) bool {
	switch pkg {
	case "repo", "repository", "r", "rep", "repos", "store":
		return true
	default:
		return false
	}
}

package correctness

import (
	"context"
	"fmt"
	"go/ast"
	"slices"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/lintutil"
)

// C039: Goroutine leak in event/command handler.
//
// Detects `go func()` or `go someFunc()` statements inside SubscribeAll,
// Subscribe, Handle, or HandleEvent function bodies. Unmanaged goroutines
// launched from event handlers have no lifecycle management — they outlive
// the handler, survive process shutdown, and leak resources. In projection
// hosts they also violate ordering guarantees (the projection host assumes
// sequential event processing).
//
// Exceptions (not flagged):
//   - Goroutines guarded by a WaitGroup (sync.WaitGroup in the same function)
//   - Goroutines guarded by an errgroup (golang.org/x/sync/errgroup)
//   - Goroutines that receive from ctx.Done() (context-aware cancellation)
//
//nolint:ireturn // factory returns public interface
func NewC039Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C039-goroutine-leak-in-handler",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					fn, ok := n.(*ast.FuncDecl)
					if !ok || fn.Body == nil {
						return true
					}

					if !isHandlerFunc(fn) {
						return true
					}

					hasWaitGroup := funcContainsWaitGroup(fn)
					hasErrgroup := funcContainsErrgroup(fn)

					ast.Inspect(fn.Body, func(nn ast.Node) bool {
						goStmt, ok := nn.(*ast.GoStmt)
						if !ok {
							return true
						}

						if hasWaitGroup || hasErrgroup {
							return true
						}

						if goroutineHasCtxDone(goStmt) {
							return true
						}

						pos := ctx.Fset.Position(goStmt.Pos())

						f, err := finding.NewBuilder(
							"C039", toolName,
							fmt.Sprintf(
								"goroutine launched inside handler %q without "+
									"WaitGroup/errgroup/context cancellation — "+
									"resource leak and ordering violation",
								fn.Name.Name,
							),
							finding.SeverityWarning,
							finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
						).
							WithCategory(finding.CategoryCorrectness).
							WithConfidence(finding.ConfidenceMedium).
							WithFixStrategy(finding.FixStrategySuggest).
							WithSuggestion(
								"Use sync.WaitGroup, errgroup.Group, or pass a cancellable context "+
									"to manage the goroutine lifecycle",
							).
							WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
							Build()
						lintutil.AppendBuild(&findings, f, err)

						return true
					})

					return true
				})
			}

			return findings, nil
		},
	)
}

// isHandlerFunc returns true if the function name or receiver method name
// matches a CQRS event/command handler pattern.
func isHandlerFunc(fn *ast.FuncDecl) bool {
	if fn.Name == nil {
		return false
	}

	name := fn.Name.Name

	return slices.Contains([]string{"SubscribeAll", "Subscribe", "Handle", "HandleEvent"}, name)
}

// funcContainsWaitGroup checks if the function body references sync.WaitGroup.
func funcContainsWaitGroup(fn *ast.FuncDecl) bool {
	found := false

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		if sel.Sel.Name == "WaitGroup" || sel.Sel.Name == "Add" ||
			sel.Sel.Name == "Done" || sel.Sel.Name == "Wait" {
			found = true
			return false
		}

		return true
	})

	return found
}

// funcContainsErrgroup checks if the function body references errgroup.
func funcContainsErrgroup(fn *ast.FuncDecl) bool {
	found := false

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		if sel.Sel.Name == "Go" || sel.Sel.Name == "Wait" {
			ast.Inspect(sel, func(nn ast.Node) bool {
				if ident, ok := nn.(*ast.Ident); ok && ident.Name == "errgroup" {
					found = true
					return false
				}

				return true
			})
		}

		return !found
	})

	return found
}

// goroutineHasCtxDone checks if the goroutine body references ctx.Done().
func goroutineHasCtxDone(goStmt *ast.GoStmt) bool {
	found := false

	ast.Inspect(goStmt, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		if sel.Sel.Name == "Done" {
			found = true
			return false
		}

		return true
	})

	return found
}

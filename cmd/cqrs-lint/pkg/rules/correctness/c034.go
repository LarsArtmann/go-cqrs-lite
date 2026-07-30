package correctness

import (
	"context"
	"go/ast"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/lintutil"
)

// C034: Goroutine without context cancellation.
// Detects `go func()` inside handler code without ctx propagation.
// A goroutine that doesn't receive ctx can outlive the parent handler,
// leading to resource leaks on shutdown.
//
//nolint:ireturn // factory returns public interface
func NewC034Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C034-goroutine-without-ctx",
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

					if !hasCtxParam(fn.Type) {
						return true
					}

					ast.Inspect(fn.Body, func(inner ast.Node) bool {
						goStmt, ok := inner.(*ast.GoStmt)
						if !ok {
							return true
						}

						if goroutineHasCtxArg(goStmt) {
							return true
						}

						pos := ctx.Fset.Position(goStmt.Pos())

						f, err := finding.NewBuilder(
							"C034",
							toolName,
							"go func() without ctx — goroutine outlives parent handler, risk of resource leak on shutdown",
							finding.SeverityWarning,
							finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
						).
							WithCategory(finding.CategoryCorrectness).
							WithConfidence(finding.ConfidenceMedium).
							WithFixStrategy(finding.FixStrategySuggest).
							WithSuggestion("Pass ctx to the goroutine: go process(ctx, ...)").
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

// goroutineHasCtxArg reports whether the go statement's function call passes
// a ctx-like argument.
func goroutineHasCtxArg(goStmt *ast.GoStmt) bool {
	if goStmt.Call == nil {
		return false
	}

	for _, arg := range goStmt.Call.Args {
		if id, ok := arg.(*ast.Ident); ok {
			name := strings.ToLower(id.Name)
			if name == "ctx" || name == "context" {
				return true
			}
		}
	}

	// Check function literals that capture ctx
	if fnLit, ok := goStmt.Call.Fun.(*ast.FuncLit); ok && fnLit.Body != nil {
		hasCtxRef := false
		ast.Inspect(fnLit.Body, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok {
				name := strings.ToLower(id.Name)
				if name == "ctx" || name == "context" {
					hasCtxRef = true
					return false
				}
			}
			return true
		})
		return hasCtxRef
	}

	return false
}

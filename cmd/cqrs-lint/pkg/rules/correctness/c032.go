package correctness

import (
	"context"
	"go/ast"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/lintutil"
)

// C032: Context propagation gaps in handlers.
// Detects `context.Background()` or `context.TODO()` calls inside CQRS
// handlers/projectors that already receive a context.Context parameter.
// Using a fresh context breaks distributed tracing (the trace ID is lost),
// cancels parent cancellation propagation, and disconnects timeout budgets.
//
// Scope is deliberately narrow: only functions that look like CQRS handlers
// (Handle/Apply/Project/etc.) or methods on handler/projection/read-model
// receiver types. Firing on EVERY ctx-accepting function produces false
// positives (background workers, initializers, detached goroutines all
// legitimately create fresh contexts).
//
//nolint:ireturn // factory returns public interface
func NewC032Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C032-ctx-propagation-gap",
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

					if !hasCtxParam(fn.Type) || !isHandlerOrProjector(fn) {
						return true
					}

					ast.Inspect(fn.Body, func(inner ast.Node) bool {
						call, ok := inner.(*ast.CallExpr)
						if !ok {
							return true
						}

						if isContextCreation(call) {
							pos := ctx.Fset.Position(call.Pos())

							f, err := finding.NewBuilder(
								"C032", toolName,
								"context.Background()/TODO() inside a handler that already receives ctx — breaks tracing and cancellation propagation",
								finding.SeverityWarning,
								finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
							).
								WithCategory(finding.CategoryCorrectness).
								WithConfidence(finding.ConfidenceHigh).
								WithFixStrategy(finding.FixStrategySuggest).
								WithSuggestion("Use the ctx parameter passed to this function instead of creating a new context").
								WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
								Build()
							lintutil.AppendBuild(&findings, f, err)
						}

						return true
					})

					return true
				})
			}

			return findings, nil
		},
	)
}

// hasCtxParam reports whether the function type has a context.Context
// first parameter (standard Go convention: func(ctx context.Context, ...)).
func hasCtxParam(ft *ast.FuncType) bool {
	if ft == nil || ft.Params == nil {
		return false
	}

	for _, param := range ft.Params.List {
		sel, ok := param.Type.(*ast.SelectorExpr)
		if !ok {
			continue
		}

		if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Name == "context" && sel.Sel.Name == "Context" {
			return true
		}
	}

	return false
}

// handlerFuncNames are CQRS handler/projector method names (lowercased —
// matching is case-insensitive so "handle", "Handle", and "HANDLE" all match).
var handlerFuncNames = map[string]bool{ //nolint:gochecknoglobals // static lookup table
	"handle": true, "handleevent": true, "handlecommand": true,
	"handlequery": true, "handlecontext": true, "handlectx": true,
	"handleerror": true, "apply": true, "project": true,
	"fold": true, "decide": true, "execute": true, "executectx": true,
}

// handlerReceiverKeywords identify receiver type names that look like CQRS
// handlers, projectors, or read models.
var handlerReceiverKeywords = []string{ //nolint:gochecknoglobals // static lookup table
	"handler", "projector", "projection", "readmodel", "viewstore",
	"view", "consumer", "subscriber", "listener", "worker",
}

// isHandlerOrProjector reports whether the function declaration looks like a
// CQRS handler or projector. It matches either by function name (Handle,
// Apply, Project, etc.) or by the receiver type name containing
// handler/projection/read-model keywords.
func isHandlerOrProjector(fn *ast.FuncDecl) bool {
	if handlerFuncNames[strings.ToLower(fn.Name.Name)] {
		return true
	}

	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return false
	}

	recvName := receiverTypeName(fn.Recv.List[0].Type)
	lower := strings.ToLower(recvName)
	for _, kw := range handlerReceiverKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}

	return false
}

// receiverTypeName extracts the type name from a receiver expression,
// unwrapping pointers and generics.
func receiverTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return receiverTypeName(t.X)
	case *ast.Ident:
		return t.Name
	case *ast.IndexExpr:
		return receiverTypeName(t.X)
	case *ast.IndexListExpr:
		return receiverTypeName(t.X)
	}
	return ""
}

// isContextCreation reports whether the call is context.Background() or
// context.TODO().
func isContextCreation(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	pkg, ok := sel.X.(*ast.Ident)
	if !ok || pkg.Name != "context" {
		return false
	}

	return sel.Sel.Name == "Background" || sel.Sel.Name == "TODO"
}

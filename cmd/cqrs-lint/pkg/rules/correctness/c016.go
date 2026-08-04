package correctness

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// C016: context.Background() or context.TODO() in handlers.
//
// Detects usage of context.Background() or context.TODO() inside functions
// that receive a context.Context parameter. Handler functions should propagate
// the caller's context for cancellation, timeouts, and tracing — not create a
// fresh detached context that ignores the caller's lifecycle.
//
//nolint:ireturn // factory returns public interface
func NewC016Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C016-background-in-handler",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					fnDecl, ok := n.(*ast.FuncDecl)
					if !ok {
						return true
					}

					if !hasContextParam(fnDecl) {
						return true
					}

					// Collect line numbers of server-lifecycle calls (Shutdown,
					// ListenAndServe, Serve) so we can exempt the
					// context.WithTimeout(context.Background(), ...) shutdown idiom.
					lifecycleLines := collectServerLifecycleLines(ctx, fnDecl)

					// Collect positions of context.Background()/TODO() calls that
					// are arguments to context.WithCancel/WithTimeout/WithDeadline/
					// WithValue — these are legitimate parent-context creation
					// patterns, not detached handler contexts.
					legitBgPositions := collectContextCreationBgPositions(fnDecl)

					// Walk the function body for context.Background()/TODO() calls.
					if fnDecl.Body == nil {
						return true
					}

					ast.Inspect(fnDecl.Body, func(inner ast.Node) bool {
						// Don't flag nested function literals — they may have
						// their own (or no) context parameter.
						if _, ok := inner.(*ast.FuncLit); ok {
							return false
						}

						call, ok := inner.(*ast.CallExpr)
						if !ok {
							return true
						}

						sel, ok := call.Fun.(*ast.SelectorExpr)
						if !ok {
							return true
						}

						ident, ok := sel.X.(*ast.Ident)
						if !ok || ident.Name != "context" {
							return true
						}

						if sel.Sel.Name != "Background" && sel.Sel.Name != "TODO" {
							return true
						}

						pos := ctx.Fset.Position(call.Pos())

						// Exempt the graceful-shutdown idiom:
						//   ctx, cancel := context.WithTimeout(context.Background(), timeout)
						//   server.Shutdown(ctx)
						// If a server-lifecycle call is within 5 lines, this
						// Background() is the shutdown timeout root — not a
						// detached handler context.
						if nearServerLifecycle(pos.Line, lifecycleLines) {
							return true
						}

						// Exempt context.Background()/TODO() used as the parent for
						// context.WithCancel/WithTimeout/WithDeadline/WithValue —
						// these are root-context creation patterns, not detached
						// handler contexts.
						if legitBgPositions[call.Pos()] {
							return true
						}

						f, err := finding.NewBuilder(
							"C016",
							toolName,
							fmt.Sprintf(
								"context.%s() in handler %s at %s — discards caller context (cancellation, timeouts, tracing lost)",
								sel.Sel.Name, fnDecl.Name.Name, pos.String(),
							),
							finding.SeverityWarning,
							finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
						).
							WithCategory(finding.CategoryCorrectness).
							WithConfidence(finding.ConfidenceHigh).
							WithSuggestion("Use the context.Context parameter passed to the handler. If you need a detached context for a background task, extract it explicitly and document why the caller's context cannot be used.").
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

// hasContextParam returns true if the function declaration has a parameter
// of type context.Context (by name convention "ctx" or by type).
func hasContextParam(fn *ast.FuncDecl) bool {
	if fn.Type == nil || fn.Type.Params == nil {
		return false
	}

	for _, field := range fn.Type.Params.List {
		if isContextType(field.Type) {
			return true
		}
	}

	return false
}

func isContextType(expr ast.Expr) bool {
	// Direct: context.Context
	sel, ok := expr.(*ast.SelectorExpr)
	if ok {
		ident, ok := sel.X.(*ast.Ident)

		return ok && ident.Name == "context" && sel.Sel.Name == "Context"
	}

	// Pointer or ellipsis: recurse one level
	if star, ok := expr.(*ast.StarExpr); ok {
		return isContextType(star.X)
	}

	if ell, ok := expr.(*ast.Ellipsis); ok {
		return isContextType(ell.Elt)
	}

	return false
}

// serverLifecycleMethods are method names that signal a server shutdown or
// startup call. When context.Background() appears near one, it's the
// graceful-shutdown timeout idiom, not a detached handler context.
var serverLifecycleMethods = map[string]bool{ //nolint:gochecknoglobals // static lookup
	"Shutdown":          true,
	"ListenAndServe":    true,
	"ListenAndServeTLS": true,
	"Serve":             true,
	"Close":             true,
}

// collectServerLifecycleLines returns the line numbers of server-lifecycle
// method calls (Shutdown, ListenAndServe, Serve, Close) within the function
// body. Used to exempt the context.WithTimeout(context.Background(), ...)
// shutdown idiom from C016.
func collectServerLifecycleLines(ctx *analyzer.AnalysisContext, fn *ast.FuncDecl) map[int]bool {
	lines := make(map[int]bool)

	if fn.Body == nil {
		return lines
	}

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		if serverLifecycleMethods[sel.Sel.Name] {
			pos := ctx.Fset.Position(call.Pos())
			lines[pos.Line] = true
		}

		return true
	})

	return lines
}

// nearServerLifecycle reports whether line is within 5 lines of any
// server-lifecycle call. This proximity window covers the typical
// shutdown pattern where context.Background() and server.Shutdown()
// are separated by a defer cancel() or a few lines of error handling.
const shutdownProximityLines = 5

func nearServerLifecycle(line int, lifecycleLines map[int]bool) bool {
	for l := range lifecycleLines {
		diff := line - l
		if diff < 0 {
			diff = -diff
		}

		if diff <= shutdownProximityLines {
			return true
		}
	}

	return false
}

// contextWithFuncs lists context package functions that create derived
// contexts from a parent. When context.Background() or context.TODO() is
// passed as the parent argument, it is a legitimate root-context pattern
// — not a detached handler context.
var contextWithFuncs = map[string]bool{ //nolint:gochecknoglobals // static lookup
	"WithCancel":      true,
	"WithTimeout":     true,
	"WithDeadline":    true,
	"WithValue":       true,
	"WithoutCancel":   true,
	"WithCancelCause": true,
}

// collectContextCreationBgPositions returns the positions of context.Background()
// and context.TODO() calls that appear as arguments to context.With* functions.
// These are legitimate parent-context creation patterns that C016 should exempt.
func collectContextCreationBgPositions(fn *ast.FuncDecl) map[token.Pos]bool {
	positions := make(map[token.Pos]bool)

	if fn.Body == nil {
		return positions
	}

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		ident, ok := sel.X.(*ast.Ident)
		if !ok || ident.Name != "context" {
			return true
		}

		if !contextWithFuncs[sel.Sel.Name] {
			return true
		}

		for _, arg := range call.Args {
			argCall, ok := arg.(*ast.CallExpr)
			if !ok {
				continue
			}

			argSel, ok := argCall.Fun.(*ast.SelectorExpr)
			if !ok {
				continue
			}

			argIdent, ok := argSel.X.(*ast.Ident)
			if !ok || argIdent.Name != "context" {
				continue
			}

			if argSel.Sel.Name == "Background" || argSel.Sel.Name == "TODO" {
				positions[argCall.Pos()] = true
			}
		}

		return true
	})

	return positions
}

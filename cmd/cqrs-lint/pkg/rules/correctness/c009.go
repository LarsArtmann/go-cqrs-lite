package correctness

import (
	"context"
	"go/ast"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// Detects panic() calls in non-test, non-init files.
// Skips panics inside functions prefixed with "must" (e.g., mustCommand,
// mustCompile) — this is an established Go convention for functions that
// panic on programming errors, like regexp.MustCompile or template.Must.
//
//nolint:ireturn // factory returns public interface
func NewC009Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C009-panic-in-production",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				for _, decl := range gf.AST.Decls {
					fn, ok := decl.(*ast.FuncDecl)
					if !ok || fn.Body == nil {
						continue
					}

					if isMustFunc(fn) {
						continue
					}

					ast.Inspect(fn.Body, func(n ast.Node) bool {
						call, ok := n.(*ast.CallExpr)
						if !ok {
							return true
						}

						ident, ok := call.Fun.(*ast.Ident)
						if !ok || ident.Name != "panic" {
							return true
						}

						pos := ctx.Fset.Position(call.Pos())

						f, err := finding.NewBuilder(
							"C009", toolName,
							"panic() in production code — use error returns instead",
							finding.SeverityWarning,
							finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
						).
							WithCategory(finding.CategoryCorrectness).
							WithConfidence(finding.ConfidenceHigh).
							WithSuggestion("Return an error instead of panicking. Panics crash the process and bypass error handling middleware.").
							WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
							Build()
						if err != nil {
							return true
						}

						findings = append(findings, f)

						return true
					})
				}
			}

			return findings, nil
		},
	)
}

// isMustFunc returns true if the function name follows the must* convention
// (e.g., mustCommand, mustCompile, mustEvent), indicating that panics are
// intentional — the function is a programming-error guard.
func isMustFunc(fn *ast.FuncDecl) bool {
	if fn.Name == nil {
		return false
	}

	return strings.HasPrefix(fn.Name.Name, "must") && len(fn.Name.Name) > 4
}

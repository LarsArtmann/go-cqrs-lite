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

// isMustFunc returns true if the function follows a panic-safe convention:
//
//   - Must*/must* (e.g., MustParseUserID, mustCommand) — the standard Go
//     "panic on programming error" convention, like regexp.MustCompile.
//   - New* (e.g., NewCollectCommand, NewServer) — constructor panics on
//     invalid arguments are a widely used Go idiom. All New* functions are
//     exempted regardless of return type (pointer, value, interface, or
//     multi-return with error) because constructor-validation panics are
//     conventional in Go.
func isMustFunc(fn *ast.FuncDecl) bool {
	if fn.Name == nil {
		return false
	}

	name := fn.Name.Name

	if (strings.HasPrefix(name, "must") || strings.HasPrefix(name, "Must")) && len(name) > 4 {
		return true
	}

	if strings.HasPrefix(name, "New") && len(name) > 3 {
		return true
	}

	return false
}

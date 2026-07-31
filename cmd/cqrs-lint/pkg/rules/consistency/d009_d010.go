package consistency

import (
	"context"
	"go/ast"
	"go/token"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// D009: Inconsistent Close detection pattern.
// Detects projects that use both io.Closer and anonymous
// interface{ Close() error } for the same operation. Standardize on io.Closer
// (stdlib type) for consistency and to avoid redundant interface definitions.
//
//nolint:ireturn // factory returns public interface
func NewD009Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"D009-inconsistent-close-pattern",
		func(_ context.Context) ([]finding.Finding, error) {
			hasIOCloser := false
			hasAnonCloser := false
			firstFile := ""
			firstLine := 0

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					// io.Closer as a selector expression (type reference, not call).
					if sel, ok := n.(*ast.SelectorExpr); ok {
						ident, ok := sel.X.(*ast.Ident)
						if ok && ident.Name == "io" && sel.Sel.Name == "Closer" {
							hasIOCloser = true
							if firstFile == "" {
								pos := ctx.Fset.Position(sel.Pos())
								firstFile = pos.Filename
								firstLine = pos.Line
							}
						}
					}

					// Anonymous interface{ Close() error }.
					if it, ok := n.(*ast.InterfaceType); ok && it.Methods != nil {
						if isSingleCloseInterface(it) {
							hasAnonCloser = true
							if firstFile == "" {
								pos := ctx.Fset.Position(it.Pos())
								firstFile = pos.Filename
								firstLine = pos.Line
							}
						}
					}

					return true
				})
			}

			if !hasIOCloser || !hasAnonCloser {
				return nil, nil
			}

			pos := anchorPos(ctx, firstFile, firstLine)

			f, err := finding.NewBuilder(
				"D009", toolName,
				"Project uses both io.Closer and anonymous interface{ Close() error } — standardize on io.Closer",
				finding.SeverityInfo,
				pos,
			).
				WithCategory(finding.CategoryStyle).
				WithConfidence(finding.ConfidenceLow).
				WithSuggestion(
					"Replace interface{ Close() error } with io.Closer — the stdlib already defines this exact interface",
				).
				WithSnippet(ctx.SourceLine(firstFile, firstLine)).
				Build()
			if err != nil {
				return nil, nil //nolint:nilerr // best-effort: drop malformed finding
			}

			return []finding.Finding{f}, nil
		},
	)
}

// isSingleCloseInterface reports whether an inline interface type declares
// exactly one method named "Close".
func isSingleCloseInterface(it *ast.InterfaceType) bool {
	if it.Methods == nil || len(it.Methods.List) != 1 {
		return false
	}

	field := it.Methods.List[0]
	if len(field.Names) != 1 || field.Names[0].Name != "Close" {
		return false
	}

	return true
}

// D010: Generic error code "internal".
// Detects errorfamily.Wrap* and errorfamily.New* calls that use the string
// literal "internal" as an error code. Generic codes like "internal" provide
// no semantic value — use descriptive namespaced codes instead (e.g.
// "user.create.conflict", "payment.charge.rejection").
//
//nolint:ireturn // factory returns public interface
func NewD010Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"D010-generic-error-code-internal",
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

					pkg, name, ok := selectorPkgAndName(call.Fun)
					if !ok || !isErrorFamilyPkg(gf.AST, pkg) {
						return true
					}

					if !isErrorFamilyWrapper(name) {
						return true
					}

					if !hasInternalLiteral(call) {
						return true
					}

					pos := ctx.Fset.Position(call.Pos())

					f, err := finding.NewBuilder(
						"D010", toolName,
						"Generic error code \"internal\" in errorfamily."+name+
							"() — use a descriptive namespaced code instead",
						finding.SeverityInfo,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryStyle).
						WithConfidence(finding.ConfidenceMedium).
						WithSuggestion(
							"Use descriptive codes like \"user.create.conflict\" or \"payment.charge.rejection\" " +
								"that convey semantic meaning for log analysis and error handling",
						).
						WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
						Build()
					if err == nil {
						findings = append(findings, f)
					}

					return true
				})
			}

			return findings, nil
		},
	)
}

// isErrorFamilyWrapper reports whether name is an errorfamily constructor or
// wrapper that accepts an error code argument.
func isErrorFamilyWrapper(name string) bool {
	switch name {
	case "NewRejection", "NewConflict", "NewTransient",
		"NewInfrastructure", "NewCorruption", "NewOrchestration",
		"WrapRejection", "WrapConflict", "WrapTransient",
		"WrapInfrastructure", "WrapCorruption", "WrapOrchestration":
		return true
	default:
		return false
	}
}

// hasInternalLiteral reports whether any argument to call is a string literal
// with the exact value "internal".
func hasInternalLiteral(call *ast.CallExpr) bool {
	for _, arg := range call.Args {
		lit, ok := arg.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			continue
		}

		if lit.Value == `"internal"` {
			return true
		}
	}

	return false
}

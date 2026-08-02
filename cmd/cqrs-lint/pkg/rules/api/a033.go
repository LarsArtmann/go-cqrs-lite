package api

import (
	"context"
	"go/ast"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/lintutil"
)

// A033: Pointless branded-ID string roundtrip.
//
// Detects id.Parse[T](x.String()) or id.MustParse[T](x.String()) — converting
// a branded ID to a raw string only to parse it straight back into a branded ID.
// This discards the existing typed value, re-runs validation, and defeats the
// type safety branded IDs exist to provide. When the source and target types
// genuinely differ (e.g. UserID → OrderID), the roundtrip hides a type-unsafe
// conversion behind a string boundary; when they are the same type, the call is
// pure waste.
//
// Fix: pass the typed value directly, or use an explicit, named conversion.
//
//nolint:ireturn // factory returns public interface
func NewA033Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"A033-branded-id-string-roundtrip",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				if !fileImportsIDPackage(gf.AST) {
					continue
				}

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}

					qual, methodName, ok := parseGenericIDCall(gf.AST, call)
					if !ok {
						return true
					}

					if methodName != "Parse" && methodName != "MustParse" {
						return true
					}

					if !isStringMethodArg(call) {
						return true
					}

					pos := ctx.Fset.Position(call.Pos())

					f, err := finding.NewBuilder(
						"A033", toolName,
						"id."+methodName+"["+qual+"] called with a .String() argument — "+
							"pointless branded-ID roundtrip that discards the typed value",
						finding.SeverityWarning,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryBestPractice).
						WithConfidence(finding.ConfidenceHigh).
						WithFixStrategy(finding.FixStrategySuggest).
						WithSuggestion("Pass the branded ID value directly instead of " +
							"stringifying and re-parsing it").
						WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
						Build()
					lintutil.AppendBuild(&findings, f, err)

					return true
				})
			}

			return findings, nil
		},
	)
}

// parseGenericIDCall inspects a generic call of the form <qual>.<method>[T](...).
// It returns the qualifier, method name, and true when the call targets the
// go-cqrs-lite id package (resolving import aliases).
func parseGenericIDCall(file *ast.File, call *ast.CallExpr) (qual, method string, ok bool) {
	idx, ok := call.Fun.(*ast.IndexExpr)
	if !ok {
		return "", "", false
	}

	sel, ok := idx.X.(*ast.SelectorExpr)
	if !ok {
		return "", "", false
	}

	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return "", "", false
	}

	if !lintutil.QualifierResolvesTo(file, ident.Name, "go-cqrs-lite/id") {
		return "", "", false
	}

	return ident.Name, sel.Sel.Name, true
}

// isStringMethodArg reports whether the first argument is a "<x>.String()" call,
// the signature of a branded-ID stringification.
func isStringMethodArg(call *ast.CallExpr) bool {
	if len(call.Args) == 0 {
		return false
	}

	arg, ok := call.Args[0].(*ast.CallExpr)
	if !ok {
		return false
	}

	sel, ok := arg.Fun.(*ast.SelectorExpr)

	return ok && sel.Sel.Name == "String"
}

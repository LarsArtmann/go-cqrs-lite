package api

import (
	"context"
	"go/ast"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/lintutil"
)

// A002: event.NewEvent with json.Marshal argument.
// Detects event.NewEvent(type, id, aggType, ver, json.Marshal(payload)) instead of event.New.
// Also detects the indirect marshalPayload helper pattern: a local function
// that calls json.Marshal and whose result is passed to event.NewEvent.
//
//nolint:ireturn // factory returns public interface
func NewA002Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"A002-newevent-manual-marshal",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			// Pre-scan: find local functions that call json.Marshal and return the result.
			// These are the "marshalPayload" helper pattern.
			marshalHelpers := collectMarshalPayloadHelpers(ctx)

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

					pkgIdent, ok := sel.X.(*ast.Ident)
					if !ok || pkgIdent.Name != "event" || sel.Sel.Name != "NewEvent" {
						return true
					}

					// Check if the 5th argument (payload) is a json.Marshal call.
					if len(call.Args) < 5 {
						return true
					}

					payloadArg := call.Args[4]

					if isDirectJSONMarshal(payloadArg) ||
						isMarshalHelperCall(payloadArg, marshalHelpers) {
						pos := ctx.Fset.Position(call.Pos())

						f, err := finding.NewBuilder(
							"A002",
							toolName,
							"event.NewEvent with json.Marshal payload — use event.New which auto-marshals typed payloads",
							finding.SeverityWarning,
							finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
						).
							WithCategory(finding.CategoryBestPractice).
							WithConfidence(finding.ConfidenceHigh).
							WithSuggestion("Replace event.NewEvent(type, id, aggType, ver, json.Marshal(payload)) " +
								"with event.New(type, id, aggType, ver, payload)").
							WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
							Build()
						if err != nil {
							return true
						}

						findings = append(findings, f)
					}

					return true
				})
			}

			return findings, nil
		},
	)
}

// isDirectJSONMarshal returns true if expr is a direct json.Marshal() call.
func isDirectJSONMarshal(expr ast.Expr) bool {
	sel, ok := lintutil.ExprCallSelector(expr)
	if !ok {
		return false
	}

	return lintutil.SelectorMatches(sel, "json", "Marshal")
}

// isMarshalHelperCall returns true if expr is a call to one of the
// marshalPayload helper functions (indirect json.Marshal).
func isMarshalHelperCall(expr ast.Expr, helpers map[string]bool) bool {
	argCall, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}

	// Direct function call: marshalPayload(p)
	if ident, ok := argCall.Fun.(*ast.Ident); ok {
		return helpers[ident.Name]
	}

	// Method call: x.marshalPayload(p)
	if sel, ok := analyzer.SelectorFromExpr(argCall.Fun); ok {
		return helpers[sel.Sel.Name]
	}

	return false
}

// collectMarshalPayloadHelperHelpers scans all non-test files for function
// declarations that call json.Marshal and return the result. These are the
// "marshalPayload" helper pattern found in github-local-sync and InboxClean.
func collectMarshalPayloadHelpers(ctx *analyzer.AnalysisContext) map[string]bool {
	helpers := make(map[string]bool)

	for _, gf := range ctx.GoFiles {
		if gf.IsTest {
			continue
		}

		for _, decl := range gf.AST.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}

			if !funcReturnsJSONMarshal(fn) {
				continue
			}

			helpers[fn.Name.Name] = true
		}
	}

	return helpers
}

// funcReturnsJSONMarshal returns true if the function body contains a
// json.Marshal call and returns at least one result. This is a heuristic
// for detecting marshalPayload helper functions: they call json.Marshal
// and return the (possibly assigned-then-returned) result.
func funcReturnsJSONMarshal(fn *ast.FuncDecl) bool {
	hasMarshal := false

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if hasMarshal {
			return false
		}

		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		sel, ok := analyzer.SelectorFromExpr(call.Fun)
		if !ok {
			return true
		}

		pkg, ok := sel.X.(*ast.Ident)

		if ok && pkg.Name == "json" && sel.Sel.Name == "Marshal" {
			hasMarshal = true
		}

		return !hasMarshal
	})

	return hasMarshal
}

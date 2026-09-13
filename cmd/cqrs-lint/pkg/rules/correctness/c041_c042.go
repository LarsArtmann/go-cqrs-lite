package correctness

import (
	"context"
	"go/ast"
	"go/token"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-finding"
)

// C041: Store Save implementation ignores expectedVersion.
// Detects custom Save method implementations that do not reference the
// expectedVersion parameter. Without referencing it, the store cannot
// enforce optimistic concurrency control, leading to lost updates.
//
//nolint:ireturn // factory returns public interface
func NewC041Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C041-save-ignores-expected-version",
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

					if fn.Name == nil || fn.Name.Name != "Save" {
						continue
					}

					versionParam := findVersionParam(fn.Type)
					if versionParam == "" {
						continue
					}

					if paramUsedInBody(fn.Body, versionParam) {
						continue
					}

					pos := ctx.Fset.Position(fn.Pos())

					f, err := finding.NewBuilder(
						"C041", toolName,
						"Save method does not reference "+versionParam+
							" — optimistic concurrency control is not enforced",
						finding.SeverityWarning,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryCorrectness).
						WithConfidence(finding.ConfidenceMedium).
						WithFixStrategy(finding.FixStrategySuggest).
						WithSuggestion("Check " + versionParam +
							" against the current stream version and return " +
							"a conflict error on mismatch").
						WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
						Build()
					if err != nil {
						continue
					}

					findings = append(findings, f)
				}
			}

			return findings, nil
		},
	)
}

// C042: Save call with literal 0 as expectedVersion.
// Detects store.Save(ctx, ref, events, expectedVersion) calls where
// expectedVersion is the literal 0 (bare or via event.Version(0)). While
// valid for new streams, passing 0 for existing streams bypasses optimistic
// concurrency, risking lost updates.
//
//nolint:ireturn // factory returns public interface
func NewC042Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C042-save-with-zero-version",
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

					sel, ok := call.Fun.(*ast.SelectorExpr)
					if !ok {
						return true
					}

					if sel.Sel.Name != "Save" {
						return true
					}

					// Canonical event.Store.Save signature: (ctx, ref, events, expectedVersion).
					if len(call.Args) < 4 {
						return true
					}

					versionArg := call.Args[3]

					if conv, ok := versionArg.(*ast.CallExpr); ok && len(conv.Args) == 1 {
						if convSel, ok := conv.Fun.(*ast.SelectorExpr); ok &&
							convSel.Sel.Name == "Version" {
							versionArg = conv.Args[0]
						}
					}

					lit, ok := versionArg.(*ast.BasicLit)
					if !ok || lit.Kind != token.INT || lit.Value != "0" {
						return true
					}

					pos := ctx.Fset.Position(call.Pos())

					f, err := finding.NewBuilder(
						"C042", toolName,
						"Save called with expectedVersion=0 — "+
							"optimistic concurrency is bypassed for this write",
						finding.SeverityInfo,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryCorrectness).
						WithConfidence(finding.ConfidenceLow).
						WithFixStrategy(finding.FixStrategySuggest).
						WithSuggestion("Load the current stream version before Save to " +
							"enable optimistic concurrency conflict detection. " +
							"Passing 0 is only safe for new streams with no prior events.").
						WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
						Build()
					if err != nil {
						return true
					}

					findings = append(findings, f)

					return true
				})
			}

			return findings, nil
		},
	)
}

// --- helpers ---

// findVersionParam returns the name of the parameter that likely represents
// expectedVersion in a Save function signature, or "" if none found.
func findVersionParam(ftype *ast.FuncType) string {
	if ftype.Params == nil {
		return ""
	}

	for _, field := range ftype.Params.List {
		for _, name := range field.Names {
			lower := strings.ToLower(name.Name)
			if strings.Contains(lower, "version") ||
				strings.Contains(lower, "expected") {
				return name.Name
			}
		}
	}

	return ""
}

// paramUsedInBody reports whether paramName appears as an identifier
// reference in the function body.
func paramUsedInBody(body *ast.BlockStmt, paramName string) bool {
	used := false

	ast.Inspect(body, func(n ast.Node) bool {
		if used {
			return false
		}

		if ident, ok := n.(*ast.Ident); ok {
			if ident.Name == paramName {
				used = true
				return false
			}
		}

		return true
	})

	return used
}

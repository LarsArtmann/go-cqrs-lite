package correctness

import (
	"context"
	"go/ast"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/lintutil"
)

// cqrsErrorMethods are CQRS library methods that return errors which should
// always be wrapped with context for debuggability.
var cqrsErrorMethods = map[string]bool{ //nolint:gochecknoglobals // static lookup table
	"Save": true, "Load": true, "LoadFromVersion": true,
	"Execute": true, "ExecuteCommand": true,
	"Dispatch": true, "Publish": true, "AppendBatch": true,
	"Register": true, "RegisterTyped": true,
	"BeginTx": true, "Commit": true,
}

// C033: Missing error wrapping for CQRS library call errors.
// Detects `if err := cqrsMethod(...); err != nil { return err }` where the
// bare error is returned without wrapping context. Unwrapped errors lose
// call-site context ("which Save failed? for what stream?"), making
// debugging production issues significantly harder.
//
//nolint:ireturn // factory returns public interface
func NewC033Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C033-missing-error-wrapping",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					ifStmt, ok := n.(*ast.IfStmt)
					if !ok {
						return true
					}

					// Pattern: if err := method(...); err != nil { return err }
					init, ok := ifStmt.Init.(*ast.AssignStmt)
					if !ok {
						return true
					}

					if !ifBodyIsBareReturnErr(ifStmt.Body) {
						return true
					}

					if !assignFromCQRSMethod(init) {
						return true
					}

					pos := ctx.Fset.Position(ifStmt.Body.Pos())

					f, err := finding.NewBuilder(
						"C033", toolName,
						"Bare return err from CQRS call — wrap with context for debuggability",
						finding.SeverityInfo,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryCorrectness).
						WithConfidence(finding.ConfidenceMedium).
						WithFixStrategy(finding.FixStrategySuggest).
						WithSuggestion("Wrap the error: `return fmt.Errorf(\"<operation>: %w\", err)`").
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

// ifBodyIsBareReturnErr reports whether the if-body is a single `return err`
// statement (bare identifier, no wrapping).
func ifBodyIsBareReturnErr(body *ast.BlockStmt) bool {
	if body == nil || len(body.List) != 1 {
		return false
	}

	ret, ok := body.List[0].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 1 {
		return false
	}

	id, ok := ret.Results[0].(*ast.Ident)

	return ok && id.Name == "err"
}

// assignFromCQRSMethod reports whether the init assignment is `err := method(...)`
// where method is a CQRS library call.
func assignFromCQRSMethod(assign *ast.AssignStmt) bool {
	for _, rhs := range assign.Rhs {
		call, ok := rhs.(*ast.CallExpr)
		if !ok {
			continue
		}

		sel, ok := analyzer.SelectorFromExpr(call.Fun)
		if !ok {
			continue
		}

		if cqrsErrorMethods[sel.Sel.Name] {
			return true
		}
	}

	return false
}

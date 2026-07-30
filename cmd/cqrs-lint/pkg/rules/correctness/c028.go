package correctness

import (
	"context"
	"go/ast"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// C028: Swallowed CQRS operation errors.
// Detects `_ = dispatch.Dispatch(...)`, `_ = repo.Execute(...)`, etc. where
// the error return from a core CQRS operation is explicitly discarded via `_`.
// These operations return errors that indicate real failures (handler not
// found, version conflict, store write failure) and should not be ignored.
//
//nolint:ireturn // factory returns public interface
func NewC028Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C028-swallowed-cqrs-error",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			cqrsMethods := map[string]bool{
				"Dispatch":       true,
				"Execute":        true,
				"ExecuteCommand": true,
				"Load":           true,
				"LoadAtVersion":  true,
				"LoadAtTime":     true,
				"Save":           true,
				"AppendBatch":    true,
				"Publish":        true,
				"Subscribe":      true,
				"Register":       true,
				"RegisterTyped":  true,
			}

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				ast.Inspect(gf.AST, func(n ast.Node) bool {
					assign, ok := n.(*ast.AssignStmt)
					if !ok {
						return true
					}

					if len(assign.Lhs) != len(assign.Rhs) {
						return true
					}

					for i, lhs := range assign.Lhs {
						ident, ok := lhs.(*ast.Ident)
						if !ok || ident.Name != "_" {
							continue
						}

						call, ok := assign.Rhs[i].(*ast.CallExpr)
						if !ok {
							continue
						}

						sel, ok := analyzer.SelectorFromExpr(call.Fun)
						if !ok {
							continue
						}

						if !cqrsMethods[sel.Sel.Name] {
							continue
						}

						// Suppress if inside an _ = init() pattern or
						// test helper (already filtered by IsTest).
						pkgName := analyzer.SelectorPackage(sel)

						if !isCQRSContext(pkgName) {
							continue
						}

						pos := ctx.Fset.Position(call.Pos())

						f, err := finding.NewBuilder(
							"C028", toolName,
							"Error from "+sel.Sel.Name+
								"() discarded — CQRS operation failures indicate real problems "+
								"(handler not found, version conflict, store failure)",
							finding.SeverityWarning,
							finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
						).
							WithCategory(finding.CategoryCorrectness).
							WithConfidence(finding.ConfidenceMedium).
							WithFixStrategy(finding.FixStrategySuggest).
							WithSuggestion("Check the error: if err := " + sel.Sel.Name +
								"(...); err != nil { return err }").
							WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
							Build()
						if err == nil {
							findings = append(findings, f)
						}
					}

					return true
				})
			}

			return findings, nil
		},
	)
}

// isCQRSContext returns true when the package qualifier or call context
// indicates a go-cqrs-lite operation. We accept any package that looks like
// a CQRS module (dispatch, repo, store, cmd, qry, bus) or an unqualified
// call (local variable).
func isCQRSContext(pkgName string) bool {
	pkg := strings.ToLower(pkgName)

	cqrsPkgHints := []string{
		"dispatch", "repo", "store", "bus",
		"cmd", "qry", "command", "query",
		"event", "decider", "projection",
	}

	for _, hint := range cqrsPkgHints {
		if strings.Contains(pkg, hint) {
			return true
		}
	}

	// Also match common CQRS local variable names.
	return pkg == ""
}

package correctness

import (
	"context"
	"fmt"
	"go/ast"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// Detects withTx-like helpers that call BeginTx but return nil instead of tx.Commit().
//
// CAUTION: when the tx variable escapes to a callback/closure argument
// (e.g. `body(tx)` in a closure-based transaction helper), the commit cannot
// be statically verified in this function body — the callback contractually
// owns it. Flagging would suggest a "fix" (return tx.Commit()) that
// double-commits. Such cases are skipped via txVarEscapesToArg.
//
//nolint:ireturn // factory returns public interface
//nolint:ireturn // factory returns public interface
func NewC001Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C001-missing-tx-commit",
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

					if !returnsError(fn) {
						continue
					}

					txVar := findBeginTxVar(fn)
					if txVar == "" {
						continue
					}

					// Single-pass collection of every tx signal C001 needs
					// (commit/defer-commit/return-nil/escape/used), replacing five
					// separate ast.Inspect walks.
					tx := analyzeTxUsage(fn, txVar)

					if tx.deferCommit || tx.commitCalled || tx.escapesToArg {
						continue
					}

					// Fire when there's either a bare success-path return
					// (returnsNil) OR the tx is actually used (txUsed). tx usage
					// is the stronger signal: if tx.Exec/tx.Query ran and the tx is
					// never committed and doesn't escape to a callback, the work is
					// lost regardless of the function's return shape. Requiring
					// returnsNil alone missed functions that return a sentinel or
					// wrapped error after using tx.
					if !tx.returnsNil && !tx.txUsed {
						continue
					}

					pos := ctx.Fset.Position(fn.Pos())

					f, err := finding.NewBuilder(
						"C001",
						toolName,
						fmt.Sprintf(
							"Function %s calls BeginTx but never commits — data silently lost on success path",
							fn.Name.Name,
						),
						finding.SeverityCritical,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryCorrectness).
						WithConfidence(finding.ConfidenceHigh).
						WithFixStrategy(finding.FixStrategyDirect).
						WithSuggestion(fmt.Sprintf("Change `return nil` to `return %s.Commit()`", txVar)).
						WithBeforeCode("return nil").
						WithAfterCode(fmt.Sprintf("return %s.Commit()", txVar)).
						WithMetadata(map[string]string{"txVar": txVar}).
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

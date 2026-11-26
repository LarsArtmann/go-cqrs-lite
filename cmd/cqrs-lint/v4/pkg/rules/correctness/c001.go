package correctness

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
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

					if tx.deferCommit || tx.commitCalled || tx.escapes {
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

					f, ok := c001Finding(ctx, fn, txVar, tx)
					if ok {
						findings = append(findings, f)
					}
				}
			}

			return findings, nil
		},
	)
}

// c001Finding builds the C001 finding for a function whose tx is never
// committed. A direct fix is offered only for a single-result `return nil`:
// the finding is anchored AT that statement (line and column of the `return`
// keyword) because the fix provider matches BeforeCode at the reported
// position, then on the reported line — anchored at the func declaration, as
// this rule used to be, the advertised fix could never be located and applied
// as a silent no-op (the C003 bug class). A `return nil` among several
// results, or a used-but-uncommitted tx with no bare return, is suggest-only:
// rewriting it would produce `return tx.Commit(), err`.
func c001Finding(
	ctx *analyzer.AnalysisContext,
	fn *ast.FuncDecl,
	txVar string,
	tx txAnalysis,
) (finding.Finding, bool) {
	pos := ctx.Fset.Position(fn.Pos())
	bare := tx.returnsNil && tx.bareReturnNilPos != token.NoPos
	if bare {
		pos = ctx.Fset.Position(tx.bareReturnNilPos)
	}

	b := findingTemplate.Builder(
		"C001",
		fmt.Sprintf(
			"Function %s calls BeginTx but never commits — data silently lost on success path",
			fn.Name.Name,
		),
		finding.SeverityCritical,
		finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
	).
		WithConfidence(finding.ConfidenceHigh).
		WithSnippet(ctx.SourceLine(pos.Filename, pos.Line))

	if !bare {
		b = b.
			WithFixStrategy(finding.FixStrategySuggest).
			WithSuggestion(fmt.Sprintf(
				"%s is used but never committed — add %s.Commit() (and handle its error) on the success path",
				txVar,
				txVar,
			))
	} else {
		b = b.
			WithFixStrategy(finding.FixStrategyDirect).
			WithSuggestion(fmt.Sprintf("Change `return nil` to `return %s.Commit()`", txVar)).
			WithBeforeCode("return nil").
			WithAfterCode(fmt.Sprintf("return %s.Commit()", txVar)).
			WithMetadata(map[string]string{"txVar": txVar})
	}

	f, err := b.Build()

	return f, err == nil
}

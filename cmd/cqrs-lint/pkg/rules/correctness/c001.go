package correctness

import (
	"context"
	"fmt"
	"go/ast"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// Detects withTx-like helpers that call BeginTx but return nil instead of tx.Commit().
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

					if hasDeferCommit(fn, txVar) {
						continue
					}

					if hasCommitCall(fn, txVar) {
						continue
					}

					if !hasReturnNil(fn) {
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

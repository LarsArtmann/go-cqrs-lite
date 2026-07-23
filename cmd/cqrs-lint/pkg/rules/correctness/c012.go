package correctness

import (
	"context"
	"fmt"
	"go/ast"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// Detects withTx-like functions that don't return the body's error.
//
//nolint:ireturn // factory returns public interface
func NewC012Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C012-missing-error-return-in-with-tx",
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
					// Check if the function body calls body(tx) and ignores the error.
					bodyVar := findBodyParam(fn)
					if bodyVar == "" {
						continue
					}

					if !ignoresBodyError(fn, bodyVar) {
						continue
					}

					pos := ctx.Fset.Position(fn.Pos())

					f, err := finding.NewBuilder(
						"C012",
						toolName,
						fmt.Sprintf(
							"Function %s ignores error from body callback — failures silently lost",
							fn.Name.Name,
						),
						finding.SeverityCritical,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryCorrectness).
						WithConfidence(finding.ConfidenceHigh).
						WithSuggestion(fmt.Sprintf("Check the error from %s(tx) and return it if non-nil", bodyVar)).
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

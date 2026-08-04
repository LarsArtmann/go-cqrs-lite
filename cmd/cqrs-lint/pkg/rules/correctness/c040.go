package correctness

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/lintutil"
)

// C040: Dead fold case detection (reverse direction of C038).
//
// C038 checks "emitted type not handled by any fold" (emit → fold direction).
// C040 checks "fold handles a type that nobody ever emits" (fold → emit
// direction). A dead fold case is either leftover from a removed/renamed
// event, or the fold case string itself has a typo.
//
// Safety: C040 only fires when the fold case has NO near-miss in the emit
// set. If a near-miss exists, C038 already catches the mismatch from the
// emit side — reporting twice would be noise. C040 also suppresses entirely
// when no emissions are detected (cross-module safety: the linter may only
// see the fold side).
//
//nolint:ireturn // factory returns public interface
func NewC040Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C040-dead-fold-case",
		func(_ context.Context) ([]finding.Finding, error) {
			emitted := ctx.Registry.EventTypesEmitted
			if len(emitted) == 0 {
				return nil, nil
			}

			emittedList := make([]string, 0, len(emitted))
			for t := range emitted {
				emittedList = append(emittedList, t)
			}

			emittedSet := make(map[string]bool, len(emitted))
			for t := range emitted {
				emittedSet[t] = true
			}

			foldCases := collectFoldCasesWithPos(ctx)
			if len(foldCases) == 0 {
				return nil, nil
			}

			var findings []finding.Finding

			for _, fc := range foldCases {
				if emittedSet[fc.Value] {
					continue
				}

				if _, dist := nearestMatch(fc.Value, emittedList); dist <= 2 {
					continue
				}

				f, err := finding.NewBuilder(
					"C040", toolName,
					fmt.Sprintf(
						"Fold case %q in %s is never emitted via event.New — "+
							"dead code or a typo in the fold case string",
						fc.Value, fc.FoldName,
					),
					finding.SeverityWarning,
					finding.Pos(finding.FilePath(fc.File), fc.Pos.Line, fc.Pos.Column),
				).
					WithCategory(finding.CategoryCorrectness).
					WithConfidence(finding.ConfidenceMedium).
					WithFixStrategy(finding.FixStrategySuggest).
					WithSuggestion(fmt.Sprintf(
						"Remove the case for %q or verify it is emitted in another module", fc.Value,
					)).
					WithSnippet(ctx.SourceLine(fc.File, fc.Pos.Line)).
					Build()
				lintutil.AppendBuild(&findings, f, err)
			}

			return findings, nil
		},
	)
}

type foldCaseInfo struct {
	Value    string
	FoldName string
	File     string
	Pos      token.Position
}

// collectFoldCasesWithPos walks all fold functions and extracts case clause
// string literals with their positions. This is the position-aware variant of
// C038's collectFoldCaseStrings — C040 needs positions to report findings at
// the case clause location.
func collectFoldCasesWithPos(ctx *analyzer.AnalysisContext) []foldCaseInfo {
	var cases []foldCaseInfo

	for _, fold := range ctx.Registry.Folds {
		for _, gf := range ctx.GoFiles {
			if gf.Path != fold.File || gf.IsTest {
				continue
			}

			ast.Inspect(gf.AST, func(n ast.Node) bool {
				fn, ok := n.(*ast.FuncDecl)
				if !ok || fn.Name == nil {
					return true
				}

				if fn.Name.Name != fold.FuncName {
					return true
				}

				ast.Inspect(fn.Body, func(nn ast.Node) bool {
					sw, ok := nn.(*ast.SwitchStmt)
					if !ok {
						return true
					}

					for _, stmt := range sw.Body.List {
						cc, ok := stmt.(*ast.CaseClause)
						if !ok || cc.List == nil {
							continue
						}

						for _, expr := range cc.List {
							if s := analyzer.StringLit(expr); s != "" {
								cases = append(cases, foldCaseInfo{
									Value:    s,
									FoldName: fold.FuncName,
									File:     fold.File,
									Pos:      ctx.Fset.Position(expr.Pos()),
								})
							}
						}
					}

					return true
				})

				return true
			})
		}
	}

	return cases
}

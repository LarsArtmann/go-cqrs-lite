package api

import (
	"context"
	"go/ast"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// A030: Incomplete snapshot configuration.
// Detects decider.NewRepository/NewTypedRepository with WithSnapshotStrategy
// but without WithSnapshotStore — this returns ErrIncompleteSnapshotConfig
// at runtime (guaranteed startup crash). A017 catches the inverse
// (store without strategy); this catches strategy without store.
//
//nolint:ireturn // factory returns public interface
func NewA030Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"A030-incomplete-snapshot-config",
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

					sel, ok := analyzer.SelectorFromExpr(call.Fun)
					if !ok {
						return true
					}

					name := sel.Sel.Name
					if name != "NewRepository" && name != "NewTypedRepository" {
						return true
					}

					pkg := analyzer.SelectorPackage(sel)
					if pkg != "decider" {
						return true
					}

					hasStrategy := false
					hasStore := false

					for _, arg := range call.Args {
						argCall, ok := arg.(*ast.CallExpr)
						if !ok {
							continue
						}
						argSel, ok := analyzer.SelectorFromExpr(argCall.Fun)
						if !ok {
							continue
						}
						switch argSel.Sel.Name {
						case "WithSnapshotStrategy":
							hasStrategy = true
						case "WithSnapshotStore":
							hasStore = true
						}
					}

					if !hasStrategy || hasStore {
						return true
					}

					pos := ctx.Fset.Position(call.Pos())

					f, err := finding.NewBuilder(
						"A030", toolName,
						"WithSnapshotStrategy without WithSnapshotStore — "+
							"returns ErrIncompleteSnapshotConfig at runtime (startup crash)",
						finding.SeverityError,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryBestPractice).
						WithConfidence(finding.ConfidenceHigh).
						WithFixStrategy(finding.FixStrategySuggest).
						WithSuggestion("Add decider.WithSnapshotStore(snapStore) " +
							"alongside WithSnapshotStrategy, or remove WithSnapshotStrategy if " +
							"snapshotting is not needed").
						WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
						Build()
					if err == nil {
						findings = append(findings, f)
					}

					return true
				})
			}

			return findings, nil
		},
	)
}

package api

import (
	"context"
	"go/ast"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// A002: event.NewEvent with json.Marshal argument.
// Detects event.NewEvent(type, id, aggType, ver, json.Marshal(payload)) instead of event.New.
func NewA002Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"A002-newevent-manual-marshal",
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

					pkgIdent, ok := sel.X.(*ast.Ident)
					if !ok || pkgIdent.Name != "event" || sel.Sel.Name != "NewEvent" {
						return true
					}
					// Check if the 5th argument (payload) is a json.Marshal call.
					if len(call.Args) < 5 {
						return true
					}

					payloadArg := call.Args[4]

					argCall, ok := payloadArg.(*ast.CallExpr)
					if !ok {
						return true
					}

					argSel, ok := analyzer.SelectorFromExpr(argCall.Fun)
					if !ok {
						return true
					}

					argPkg, ok := argSel.X.(*ast.Ident)
					if !ok || argPkg.Name != "json" || argSel.Sel.Name != "Marshal" {
						return true
					}

					pos := ctx.Fset.Position(call.Pos())

					f, err := finding.NewBuilder(
						"A002", toolName,
						"event.NewEvent with json.Marshal payload — use event.New which auto-marshals typed payloads",
						finding.SeverityWarning,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryBestPractice).
						WithConfidence(finding.ConfidenceHigh).
						WithSuggestion("Replace event.NewEvent(type, id, aggType, ver, json.Marshal(payload)) with event.New(type, id, aggType, ver, payload)").
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

package api

import (
	"context"
	"go/ast"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// A003: Explicit codec in decode.
// Detects event.DecodePayload[T](evt, codec.JSONCodec{}) — should use DecodePayloadAuto.
func NewA003Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"A003-explicit-codec-in-decode",
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

					pkgIdent, ok := sel.X.(*ast.Ident)
					if !ok || pkgIdent.Name != "event" {
						return true
					}

					if sel.Sel.Name != "DecodePayload" {
						return true
					}
					// Check if there are more than 1 argument (the event) — extra args are codec.
					if len(call.Args) <= 1 {
						return true
					}

					pos := ctx.Fset.Position(call.Pos())

					f, err := finding.NewBuilder(
						"A003", toolName,
						"Explicit codec passed to DecodePayload — use DecodePayloadAuto[T] for automatic codec detection",
						finding.SeverityInfo,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryBestPractice).
						WithConfidence(finding.ConfidenceMedium).
						WithSuggestion("Use event.DecodePayloadAuto[T](evt) — it auto-detects JSON/CBOR from the event's encoding stamp").
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

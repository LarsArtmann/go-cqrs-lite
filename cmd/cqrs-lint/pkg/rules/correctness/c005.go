package correctness

import (
	"context"
	"go/ast"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// Detects json.Unmarshal(evt.Payload(), ...) instead of event.DecodePayloadAuto[T](evt).
func NewC005Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C005-raw-json-unmarshal-payload",
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
					if !ok || pkgIdent.Name != "json" {
						return true
					}

					if sel.Sel.Name != "Unmarshal" && sel.Sel.Name != "NewDecoder" {
						return true
					}

					var payloadArg ast.Expr
					if sel.Sel.Name == "Unmarshal" && len(call.Args) > 0 {
						payloadArg = call.Args[0]
					}

					if sel.Sel.Name == "NewDecoder" && len(call.Args) > 0 {
						payloadArg = call.Args[0]
					}

					if payloadArg == nil {
						return true
					}

					if !isPayloadCall(payloadArg) {
						return true
					}

					pos := ctx.Fset.Position(call.Pos())

					f, err := finding.NewBuilder(
						"C005", toolName,
						"Raw json.Unmarshal on event payload — use event.DecodePayloadAuto[T] instead",
						finding.SeverityError,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryCorrectness).
						WithConfidence(finding.ConfidenceHigh).
						WithFixStrategy(finding.FixStrategySuggest).
						WithSuggestion("Use event.DecodePayloadAuto[YourPayload](evt) for automatic codec detection and schema versioning").
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

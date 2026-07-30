package consistency

import (
	"context"
	"go/ast"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// Detects nil payload passed to event.New or event.NewEvent. Events with nil
// payloads cannot be decoded and provide no audit trail.
//
//nolint:ireturn // factory returns public interface
func NewD011Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"D011-nil-payload-event",
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

					pkg, ok := sel.X.(*ast.Ident)
					if !ok || pkg.Name != "event" {
						return true
					}

					if sel.Sel.Name != "New" && sel.Sel.Name != "NewEvent" {
						return true
					}

					// Payload is the 5th argument (index 4) for both New and NewEvent:
					// New(type, streamID, streamType, version, payload, opts...)
					if len(call.Args) < 5 {
						return true
					}

					ident, ok := call.Args[4].(*ast.Ident)
					if !ok || ident.Name != "nil" {
						return true
					}

					pos := ctx.Fset.Position(call.Pos())

					f, err := finding.NewBuilder(
						"D011", toolName,
						"Event created with nil payload — cannot be decoded, provides no audit trail",
						finding.SeverityWarning,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryNaming).
						WithConfidence(finding.ConfidenceHigh).
						WithFixStrategy(finding.FixStrategySuggest).
						WithSuggestion("Define a payload struct even for simple events (e.g., type TogglePayload struct{})").
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

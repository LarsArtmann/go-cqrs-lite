package api

import (
	"context"
	"fmt"
	"go/ast"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// A006: Adapter layer wrapping.
// Detects WrapEvent/UnwrapEvent/ToEvent adapter methods that add unnecessary indirection.
//
//nolint:ireturn // factory returns public interface
func NewA006Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"A006-adapter-layer-wrapping",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			adapterNames := []string{
				"WrapEvent",
				"UnwrapEvent",
				"ToEvent",
				"FromEvent",
				"ToCQRSEvent",
				"FromCQRSEvent",
			}

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				for _, decl := range gf.AST.Decls {
					fn, ok := decl.(*ast.FuncDecl)
					if !ok || fn.Name == nil {
						continue
					}

					for _, adapter := range adapterNames {
						if fn.Name.Name == adapter {
							pos := ctx.Fset.Position(fn.Pos())

							f, err := finding.NewBuilder(
								"A006", toolName,
								fmt.Sprintf("Adapter method %s — go-cqrs-lite events are concrete types, no wrapping needed", adapter),
								finding.SeverityInfo,
								finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
							).
								WithCategory(finding.CategoryBestPractice).
								WithConfidence(finding.ConfidenceLow).
								WithSuggestion("Work with event.Event directly — it's a type alias for *ImmutableEvent, no adapter layer needed").
								WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
								Build()
							if err == nil {
								findings = append(findings, f)
							}
						}
					}
				}
			}

			return findings, nil
		},
	)
}

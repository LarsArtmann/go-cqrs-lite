package api

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// A008: Parallel type system.
// Detects custom AggregateID/Version/CommandType types duplicating go-cqrs-lite.
//
//nolint:ireturn // factory returns public interface
func NewA008Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"A008-parallel-type-system",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			duplicateTypes := map[string]bool{
				"AggregateID": true,
				"CommandType": true,
				"EventType":   true,
				"Version":     true,
			}

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				for _, decl := range gf.AST.Decls {
					gd, ok := decl.(*ast.GenDecl)
					if !ok || gd.Tok != token.TYPE {
						continue
					}

					for _, spec := range gd.Specs {
						ts, ok := spec.(*ast.TypeSpec)
						if !ok {
							continue
						}

						if duplicateTypes[ts.Name.Name] {
							// Check if this is in a types package (not the actual event package).
							if strings.Contains(gf.Path, "/event/") ||
								strings.HasPrefix(gf.Path, "event/") ||
								strings.Contains(gf.Path, "/command/") ||
								strings.HasPrefix(gf.Path, "command/") {
								continue
							}

							pos := ctx.Fset.Position(ts.Pos())

							f, err := finding.NewBuilder(
								"A008", toolName,
								fmt.Sprintf("Custom type %s duplicates go-cqrs-lite's type system — use id.%s or event.%s instead", ts.Name.Name, ts.Name.Name, ts.Name.Name),
								finding.SeverityError,
								finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
							).
								WithCategory(finding.CategoryBestPractice).
								WithConfidence(finding.ConfidenceHigh).
								WithSuggestion(fmt.Sprintf("Replace custom %s with the library's type from id/ or event/ package", ts.Name.Name)).
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

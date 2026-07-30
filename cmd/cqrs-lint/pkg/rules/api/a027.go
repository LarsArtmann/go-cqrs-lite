package api

import (
	"context"
	"go/ast"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// Detects event.WithCodec called 3+ times in the same file. The codec should
// be set once via event.DefaultCodec or at the repository/bundle level, not
// repeated on every event.New call.
//
//nolint:ireturn // factory returns public interface
func NewA027Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"A027-repeated-withcodec",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if gf.IsTest {
					continue
				}

				count := 0
				var firstPos ast.Node

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

					if sel.Sel.Name != "WithCodec" {
						return true
					}

					count++
					if firstPos == nil {
						firstPos = call
					}

					return true
				})

				if count < 3 {
					continue
				}

				pos := ctx.Fset.Position(firstPos.Pos())

				f, err := finding.NewBuilder(
					"A027", toolName,
					"event.WithCodec called "+itoa(count)+
						" times in this file — set codec once via event.DefaultCodec",
					finding.SeverityInfo,
					finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
				).
					WithCategory(finding.CategoryBestPractice).
					WithConfidence(finding.ConfidenceHigh).
					WithFixStrategy(finding.FixStrategySuggest).
					WithSuggestion("Set event.DefaultCodec = codec.JSONCodec{} once, or pass the codec at the repository level").
					WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
					Build()
				if err != nil {
					continue
				}

				findings = append(findings, f)
			}

			return findings, nil
		},
	)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}

	var buf [20]byte
	i := len(buf)

	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}

	return string(buf[i:])
}

package correctness

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// Detects command ID() methods that return a zero-value composite literal.
//nolint:ireturn // factory returns public interface
func NewC002Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"C002-broken-command-id",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, cmd := range ctx.Registry.Commands {
				if !cmd.IDReturnsZero {
					continue
				}

				f, err := finding.NewBuilder(
					"C002", toolName,
					fmt.Sprintf("Command %s ID() returns zero value — breaks idempotency and tracing", cmd.Name),
					finding.SeverityCritical,
					finding.Pos(finding.FilePath(cmd.File), cmd.Pos.Line, cmd.Pos.Column),
				).
					WithCategory(finding.CategoryCorrectness).
					WithConfidence(finding.ConfidenceHigh).
					WithFixStrategy(finding.FixStrategySuggest).
					WithSuggestion("Generate a unique CommandID per instance, or embed *command.BasicCommand which provides ID() automatically").
					WithSnippet(ctx.SourceLine(cmd.File, cmd.Pos.Line)).
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

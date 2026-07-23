package api

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// A001: Manual command interface.
// Detects command structs with manual Type()/StreamID()/ID() methods instead of embedding *command.BasicCommand.
//
//nolint:ireturn // factory returns public interface
func NewA001Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"A001-manual-command-interface",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, cmd := range ctx.Registry.Commands {
				if cmd.HasBasicCmd {
					continue
				}

				manualCount := 0
				if cmd.ManualID {
					manualCount++
				}
				// Check for manual Type() and StreamID() methods.
				if hasMethod(ctx, cmd, "Type") {
					manualCount++
				}

				if hasMethod(ctx, cmd, "StreamID") {
					manualCount++
				}

				if manualCount >= 2 {
					f, err := finding.NewBuilder(
						"A001", toolName,
						fmt.Sprintf("Command %s manually implements Type()/ID()/StreamID() — embed *command.BasicCommand instead", cmd.Name),
						finding.SeverityError,
						finding.Pos(finding.FilePath(cmd.File), cmd.Pos.Line, cmd.Pos.Column),
					).
						WithCategory(finding.CategoryBestPractice).
						WithConfidence(finding.ConfidenceHigh).
						WithSuggestion("Embed *command.BasicCommand to get Type(), ID(), and StreamID() for free, constructed via command.New(type, streamID)").
						WithSnippet(ctx.SourceLine(cmd.File, cmd.Pos.Line)).
						Build()
					if err != nil {
						continue
					}

					findings = append(findings, f)
				}
			}

			return findings, nil
		},
	)
}

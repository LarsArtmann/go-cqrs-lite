// Package architecture implements cross-module architecture rules.
package architecture

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

const toolName finding.ToolName = "cqrs-lint"

// E004: Event not in catalog.
// Detects event types emitted via event.New/NewEvent that are not registered in the catalog.
//
//nolint:ireturn // factory returns public interface
func NewE004Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"E004-event-not-in-catalog",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for eventType, emission := range ctx.Registry.EventTypesEmitted {
				if ctx.Registry.IsEventInCatalog(eventType) {
					continue
				}

				pos := finding.Pos(finding.FilePath(emission.File), emission.Line, 1)
				if emission.File == "" {
					pos = finding.Pos(finding.FilePath(ctx.ProjectRoot+"/go.mod"), 1, 1)
				}

				f, err := finding.NewBuilder(
					"E004", toolName,
					fmt.Sprintf("Event type %q is emitted but not registered in catalog — consumers cannot discover it", eventType),
					finding.SeverityInfo,
					pos,
				).
					WithCategory(finding.CategoryStructure).
					WithConfidence(finding.ConfidenceMedium).
					WithSuggestion(fmt.Sprintf("Register %q via catalog.Event(%q, ...) or add it to your catalog Registry", eventType, eventType)).
					WithSnippet(ctx.SourceLine(emission.File, emission.Line)).
					Build()
				if err == nil {
					findings = append(findings, f)
				}
			}

			return findings, nil
		},
	)
}

// E005: Command without handler.
// Detects command types that are defined but never registered with a handler.
//
//nolint:ireturn // factory returns public interface
func NewE005Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"E005-command-without-handler",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, cmd := range ctx.Registry.Commands {
				if ctx.Registry.IsCommandRegistered(cmd.Name) {
					continue
				}
				// Skip commands that are just embedding BasicCommand (they might be registered elsewhere).
				if cmd.Name == "" {
					continue
				}

				f, err := finding.NewBuilder(
					"E005", toolName,
					fmt.Sprintf("Command type %q has no registered handler — dispatching it will return ErrNoHandler", cmd.Name),
					finding.SeverityWarning,
					finding.Pos(finding.FilePath(cmd.File), cmd.Pos.Line, cmd.Pos.Column),
				).
					WithCategory(finding.CategoryStructure).
					WithConfidence(finding.ConfidenceMedium).
					WithSuggestion(fmt.Sprintf("Register a handler for %s via command.RegisterTyped or dispatcher.Register", cmd.Name)).
					WithSnippet(ctx.SourceLine(cmd.File, cmd.Pos.Line)).
					Build()
				if err == nil {
					findings = append(findings, f)
				}
			}

			return findings, nil
		},
	)
}

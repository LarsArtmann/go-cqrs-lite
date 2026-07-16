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
func NewE004Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"E004-event-not-in-catalog",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for eventType, file := range ctx.Registry.EventTypesEmitted {
				if ctx.Registry.IsEventInCatalog(eventType) {
					continue
				}

				f, err := finding.NewBuilder(
					"E004", toolName,
					fmt.Sprintf("Event type %q is emitted but not registered in catalog — consumers cannot discover it", eventType),
					finding.SeverityInfo,
					finding.Pos(finding.FilePath(file), 1, 1),
				).
					WithCategory(finding.CategoryStructure).
					WithConfidence(finding.ConfidenceMedium).
					WithSuggestion(fmt.Sprintf("Register %q via catalog.Event(%q, ...) or add it to your catalog Registry", eventType, eventType)).
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
					Build()
				if err == nil {
					findings = append(findings, f)
				}
			}

			return findings, nil
		},
	)
}

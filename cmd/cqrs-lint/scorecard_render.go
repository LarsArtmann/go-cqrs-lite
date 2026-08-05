package main

import (
	"encoding/json/v2"
	"fmt"
	"strings"

	output "github.com/larsartmann/go-output"
	"github.com/larsartmann/go-output/table"
)

// renderScorecard renders the scorecard as a text table or JSON based on the
// format parameter. Returns the rendered string.
func renderScorecard(
	result ScorecardResult,
	format string,
	colorMode output.ColorMode,
) (string, error) {
	switch strings.ToLower(format) {
	case "json":
		return renderScorecardJSON(result)
	case "markdown", "md":
		return renderScorecardMarkdown(result), nil
	case "sarif":
		return renderScorecardSARIF(result)
	default:
		return renderScorecardText(result, colorMode), nil
	}
}

// renderScorecardText renders the scorecard as a human-readable table with
// a summary banner, Used table, Missing table, and recommendations.
func renderScorecardText(result ScorecardResult, colorMode output.ColorMode) string {
	var b strings.Builder

	// Summary banner.
	b.WriteString("\n")
	fmt.Fprintf(&b, "Adoption: %d/%d relevant modules (%d%%) — Grade: %s\n",
		result.Summary.UsedCount,
		result.Summary.RelevantTotal,
		result.Summary.CoveragePercent,
		result.Summary.Grade)
	if result.Summary.IrrelevantCount > 0 {
		fmt.Fprintf(&b, "  (%d modules excluded as irrelevant for this profile)\n",
			result.Summary.IrrelevantCount)
	}
	b.WriteString("\n")

	// Used modules table.
	if len(result.Used) > 0 {
		b.WriteString("USED\n")
		usedTable, err := renderModuleTable(result.Used, colorMode)
		if err != nil {
			b.WriteString(formatModuleList(result.Used))
		} else {
			b.WriteString(usedTable)
		}
		b.WriteString("\n")
	}

	// Missing modules table.
	if len(result.Missing) > 0 {
		b.WriteString("MISSING\n")
		missingTable, err := renderModuleTable(result.Missing, colorMode)
		if err != nil {
			b.WriteString(formatModuleList(result.Missing))
		} else {
			b.WriteString(missingTable)
		}
		b.WriteString("\n")
	}

	// Recommendations.
	if len(result.Recommendations) > 0 {
		b.WriteString("RECOMMENDATIONS\n")
		for _, rec := range result.Recommendations {
			fmt.Fprintf(&b, "  → %s\n", rec)
		}
		b.WriteString("\n")
	}

	return b.String()
}

// renderModuleTable renders a list of modules as a table. The Evidence
// column is included only when at least one module has non-empty evidence
// (typically the USED table — missing/irrelevant modules have none).
func renderModuleTable(modules []ScorecardModule, colorMode output.ColorMode) (string, error) {
	showEvidence := false
	for _, m := range modules {
		if m.Evidence != "" {
			showEvidence = true
			break
		}
	}

	builder := output.NewTableBuilder()
	if showEvidence {
		builder.SetHeaders("Module", "Category", "Status", "Evidence")
	} else {
		builder.SetHeaders("Module", "Category", "Status")
	}

	for _, m := range modules {
		if showEvidence {
			builder.AddRow(m.DisplayName, m.Category, m.Status, m.Evidence)
		} else {
			builder.AddRow(m.DisplayName, m.Category, m.Status)
		}
	}

	builder.SetFooter(fmt.Sprintf("%d modules", len(modules)))

	data := builder.Build()
	return table.Render(data, table.WithColorMode(colorMode))
}

// formatModuleList is a fallback when table rendering fails. It outputs
// a simple aligned list without table borders.
func formatModuleList(modules []ScorecardModule) string {
	var b strings.Builder
	for _, m := range modules {
		if m.Evidence != "" {
			fmt.Fprintf(
				&b,
				"  %-24s  %-16s  %-8s  %s\n",
				m.DisplayName,
				m.Category,
				m.Status,
				m.Evidence,
			)
		} else {
			fmt.Fprintf(&b, "  %-24s  %-16s  %s\n", m.DisplayName, m.Category, m.Status)
		}
	}
	return b.String()
}

// renderScorecardJSON marshals the scorecard as canonical JSON.
func renderScorecardJSON(result ScorecardResult) (string, error) {
	data, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("marshal scorecard JSON: %w", err)
	}
	return string(data) + "\n", nil
}

// renderScorecardMarkdown renders the scorecard as a GitHub-flavored Markdown
// document — ideal for PR comments, README badges, and CI artifacts. The
// output uses standard Markdown tables so it renders natively on GitHub,
// GitLab, and most Markdown renderers.
func renderScorecardMarkdown(result ScorecardResult) string {
	var b strings.Builder

	fmt.Fprintf(&b, "## cqrs-lint Adoption Scorecard\n\n")
	fmt.Fprintf(&b, "**Adoption: %d/%d relevant modules (%d%%)** — Grade: %s\n",
		result.Summary.UsedCount,
		result.Summary.RelevantTotal,
		result.Summary.CoveragePercent,
		result.Summary.Grade)
	if result.Summary.IrrelevantCount > 0 {
		fmt.Fprintf(&b, "\n_%d modules excluded as irrelevant for this profile._\n",
			result.Summary.IrrelevantCount)
	}

	renderMarkdownTable := func(title string, modules []ScorecardModule) {
		if len(modules) == 0 {
			return
		}
		fmt.Fprintf(&b, "\n### %s (%d)\n\n", title, len(modules))
		b.WriteString("| Module | Category | Status | Suggestion |\n")
		b.WriteString("|--------|----------|--------|------------|\n")
		for _, m := range modules {
			suggestion := m.Suggestion
			if suggestion == "" {
				suggestion = "—"
			}
			fmt.Fprintf(&b, "| %s | %s | %s | %s |\n",
				m.DisplayName, m.Category, m.Status, suggestion)
		}
	}

	renderMarkdownTable("Used", result.Used)
	renderMarkdownTable("Missing", result.Missing)

	if len(result.Recommendations) > 0 {
		b.WriteString("\n### Recommendations\n\n")
		for _, rec := range result.Recommendations {
			fmt.Fprintf(&b, "- %s\n", rec)
		}
	}

	return b.String()
}

// renderScorecardSARIF emits the scorecard as a SARIF 2.1.0 report for CI
// integration (GitHub Code Scanning, Azure DevOps). Missing modules appear as
// info-level results; the adoption summary lives in run.properties so CI
// scripts can extract coverage metrics without parsing human-readable output.
func renderScorecardSARIF(result ScorecardResult) (string, error) {
	report := sarifReport{
		Schema:  "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
		Version: "2.1.0",
		Runs: []sarifRun{{
			Tool: sarifTool{
				Driver: sarifDriver{
					Name:           "cqrs-lint-scorecard",
					Version:        version,
					InformationURI: "https://github.com/larsartmann/go-cqrs-lite/tree/main/cmd/cqrs-lint",
					Rules: []sarifRule{
						{
							ID:   "scorecard/missing-module",
							Name: "MissingModule",
							ShortDescription: sarifMessage{
								Text: "Relevant go-cqrs-lite module not adopted",
							},
							FullDescription: sarifMessage{
								Text: "This module is relevant to the project's feature profile but is not imported.",
							},
							DefaultConfig: sarifConfig{Level: "info"},
						},
					},
				},
			},
			Properties: map[string]any{
				"coveragePercent": result.Summary.CoveragePercent,
				"grade":           result.Summary.Grade,
				"usedCount":       result.Summary.UsedCount,
				"relevantTotal":   result.Summary.RelevantTotal,
				"irrelevantCount": result.Summary.IrrelevantCount,
			},
		}},
	}

	for _, m := range result.Missing {
		msg := fmt.Sprintf("Missing module: %s (%s)", m.DisplayName, m.Category)
		if m.Suggestion != "" {
			msg += " — " + m.Suggestion
		}

		report.Runs[0].Results = append(report.Runs[0].Results, sarifResult{
			RuleID:  "scorecard/missing-module",
			Level:   "info",
			Message: sarifMessage{Text: msg},
			Locations: []sarifLocation{{
				PhysicalLocation: sarifPhysicalLocation{
					ArtifactLocation: sarifArtifactLocation{URI: "go.mod"},
				},
			}},
		})
	}

	data, err := json.Marshal(report)
	if err != nil {
		return "", fmt.Errorf("marshal scorecard SARIF: %w", err)
	}
	return string(data) + "\n", nil
}

// SARIF 2.1.0 structural types for scorecard output.

type sarifMessage struct {
	Text string `json:"text"`
}

type sarifConfig struct {
	Level string `json:"level"`
}

type sarifRule struct {
	ID               string       `json:"id"`
	Name             string       `json:"name"`
	ShortDescription sarifMessage `json:"shortDescription"`
	FullDescription  sarifMessage `json:"fullDescription"`
	DefaultConfig    sarifConfig  `json:"defaultConfiguration"` //nolint:tagliatelle // SARIF spec key
}

type sarifDriver struct {
	Name           string      `json:"name"`
	Version        string      `json:"version"`
	InformationURI string      `json:"informationUri"` //nolint:tagliatelle // SARIF spec key
	Rules          []sarifRule `json:"rules,omitempty"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifArtifactLocation struct {
	URI string `json:"uri"`
}

type sarifPhysicalLocation struct {
	ArtifactLocation sarifArtifactLocation `json:"artifactLocation"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysicalLocation `json:"physicalLocation"`
}

type sarifResult struct {
	RuleID    string          `json:"ruleId"`
	Level     string          `json:"level"`
	Message   sarifMessage    `json:"message"`
	Locations []sarifLocation `json:"locations,omitempty"`
}

type sarifRun struct {
	Tool       sarifTool      `json:"tool"`
	Properties map[string]any `json:"properties,omitempty"`
	Results    []sarifResult  `json:"results,omitempty"`
}

type sarifReport struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []sarifRun `json:"runs"`
}

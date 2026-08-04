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
	if strings.ToLower(format) == "json" {
		return renderScorecardJSON(result)
	}
	return renderScorecardText(result, colorMode), nil
}

// renderScorecardText renders the scorecard as a human-readable table with
// a summary banner, Used table, Missing table, and recommendations.
func renderScorecardText(result ScorecardResult, colorMode output.ColorMode) string {
	var b strings.Builder

	// Summary banner.
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("Adoption: %d/%d relevant modules (%d%%) — Grade: %s\n",
		result.Summary.UsedCount,
		result.Summary.RelevantTotal,
		result.Summary.CoveragePercent,
		result.Summary.Grade))
	if result.Summary.IrrelevantCount > 0 {
		b.WriteString(fmt.Sprintf("  (%d modules excluded as irrelevant for this profile)\n",
			result.Summary.IrrelevantCount))
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
			b.WriteString(fmt.Sprintf("  → %s\n", rec))
		}
		b.WriteString("\n")
	}

	return b.String()
}

// renderModuleTable renders a list of modules as a table.
func renderModuleTable(modules []ScorecardModule, colorMode output.ColorMode) (string, error) {
	builder := output.NewTableBuilder().
		SetHeaders("Module", "Category", "Status")

	for _, m := range modules {
		builder.AddRow(m.DisplayName, m.Category, m.Status)
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
		b.WriteString(fmt.Sprintf("  %-24s  %-16s  %s\n", m.DisplayName, m.Category, m.Status))
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

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
	if result.Summary.ModernityGrade != "" {
		fmt.Fprintf(&b, "Modernity: %s — %s\n",
			result.Summary.ModernityGrade, ModernityHint(result.Summary.ModernityGrade))
	}
	if result.Summary.WaivedCount > 0 {
		fmt.Fprintf(&b, "  (%d modules waived — recorded refusals, see WAIVED)\n",
			result.Summary.WaivedCount)
	}
	b.WriteString("\n")

	// Metaengine section.
	if result.Metaengine != nil {
		b.WriteString("METAENGINE\n")
		fmt.Fprintf(&b, "  Detected: yes\n")
		if len(result.Metaengine.Engines) > 0 {
			fmt.Fprintf(&b, "  Engines:  %s\n", strings.Join(result.Metaengine.Engines, ", "))
		}
		if result.Metaengine.PushdownAdopted {
			b.WriteString("  Pushdown: adopted (FilterOnField/SortOnField)\n")
		} else {
			b.WriteString("  Pushdown: not adopted\n")
		}
		if result.Metaengine.Suggestion != "" {
			fmt.Fprintf(&b, "  → %s\n", result.Metaengine.Suggestion)
		}
		b.WriteString("\n")
	}

	// Deprecated-surface panel (V007/F030 — v5-readiness).
	if result.Deprecated != nil {
		b.WriteString("DEPRECATED SURFACES (v5-readiness)\n")
		fmt.Fprintf(&b, "  v5-removed API uses:       %d\n", result.Deprecated.RemovedAPIUses)
		fmt.Fprintf(&b, "  transport/http SSE uses:   %d\n", result.Deprecated.DeprecatedTransport)
		if result.Deprecated.Suggestion != "" {
			fmt.Fprintf(&b, "  → %s\n", result.Deprecated.Suggestion)
		}
		b.WriteString("\n")
	}

	// Used modules table.
	writeModuleSection(&b, "USED", result.Used, colorMode)

	// Missing modules table.
	writeModuleSection(&b, "MISSING", result.Missing, colorMode)

	// Waived modules table (recorded refusals — pressure stays visible).
	writeModuleSection(&b, "WAIVED (recorded refusals — re-litigate on trigger)",
		result.Waived, colorMode)

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
// (typically the USED/WAIVED tables — missing/irrelevant modules have none).
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

// writeModuleSection renders one titled module-table section, falling back
// to the plain list when table rendering fails. Empty sections are skipped.
func writeModuleSection(
	b *strings.Builder,
	title string,
	modules []ScorecardModule,
	colorMode output.ColorMode,
) {
	if len(modules) == 0 {
		return
	}
	b.WriteString(title + "\n")
	tableOut, err := renderModuleTable(modules, colorMode)
	if err != nil {
		b.WriteString(formatModuleList(modules))
	} else {
		b.WriteString(tableOut)
	}
	b.WriteString("\n")
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
	data, err := json.Marshal(result, json.Deterministic(true))
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
	if result.Summary.ModernityGrade != "" {
		fmt.Fprintf(&b, "\n**Modernity: %s** — %s\n",
			result.Summary.ModernityGrade, ModernityHint(result.Summary.ModernityGrade))
	}
	if result.Summary.WaivedCount > 0 {
		fmt.Fprintf(&b, "\n_%d modules waived — recorded refusals with reasons and triggers below._\n",
			result.Summary.WaivedCount)
	}

	if result.Metaengine != nil {
		b.WriteString("\n### Metaengine\n\n")
		fmt.Fprintf(&b, "- **Detected:** yes\n")
		if len(result.Metaengine.Engines) > 0 {
			fmt.Fprintf(&b, "- **Engines:** %s\n",
				strings.Join(result.Metaengine.Engines, ", "))
		}
		if result.Metaengine.PushdownAdopted {
			b.WriteString("- **Pushdown:** adopted (FilterOnField/SortOnField)\n")
		} else {
			b.WriteString("- **Pushdown:** not adopted\n")
		}
		if result.Metaengine.Suggestion != "" {
			fmt.Fprintf(&b, "\n_💡 %s_\n", result.Metaengine.Suggestion)
		}
	}

	if result.Deprecated != nil {
		b.WriteString("\n### Deprecated surfaces (v5-readiness)\n\n")
		fmt.Fprintf(&b, "- **v5-removed API uses:** %d\n", result.Deprecated.RemovedAPIUses)
		fmt.Fprintf(
			&b,
			"- **transport/http SSE uses:** %d\n",
			result.Deprecated.DeprecatedTransport,
		)
		if result.Deprecated.Suggestion != "" {
			fmt.Fprintf(&b, "\n_💡 %s_\n", result.Deprecated.Suggestion)
		}
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

	if len(result.Waived) > 0 {
		fmt.Fprintf(&b, "\n### Waived (%d) — recorded refusals\n\n", len(result.Waived))
		b.WriteString("| Module | Status | Reason / revisit trigger |\n")
		b.WriteString("|--------|--------|--------------------------|\n")
		for _, m := range result.Waived {
			fmt.Fprintf(&b, "| %s | %s | %s |\n", m.DisplayName, m.Status, m.Evidence)
		}
	}

	if len(result.Recommendations) > 0 {
		b.WriteString("\n### Recommendations\n\n")
		for _, rec := range result.Recommendations {
			fmt.Fprintf(&b, "- %s\n", rec)
		}
	}

	return b.String()
}

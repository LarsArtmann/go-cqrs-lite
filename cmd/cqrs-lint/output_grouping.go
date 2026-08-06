package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/larsartmann/go-finding"
	output "github.com/larsartmann/go-output"
)

// formatSuppressedFindings prints suppressed findings with their suppression
// reason, for use with --show-suppressed. These findings are excluded from
// the normal output, health score, and exit-code check — this is an audit view.
func formatSuppressedFindings(w io.Writer, findings []finding.Finding, cm output.ColorMode) {
	useColor := shouldColor(cm, w)

	_, _ = fmt.Fprintf(w, "\n--- Suppressed Findings (%d) ---\n\n", len(findings))

	for _, f := range findings {
		sevStr := strings.ToUpper(f.Severity.String())
		if useColor {
			sevStr = ansiGray + sevStr + ansiReset
		}

		location := fmt.Sprintf("%s:%d:%d", f.Position.File, f.Position.Line, f.Position.Column)

		_, _ = fmt.Fprintf(w, "%s %s %s\n", sevStr, location, f.Message)

		if f.Rule != "" {
			_, _ = fmt.Fprintf(w, "  [%s]\n", f.Rule)
		}

		if f.Suppression != nil {
			reason := f.Suppression.Reason
			if reason == "" {
				reason = "(no reason given)"
			}
			_, _ = fmt.Fprintf(w, "  Suppressed: %s\n", reason)
		}

		_, _ = fmt.Fprintln(w)
	}
}

func outputFindings(ctx context.Context, findings []finding.Finding, cfg *AppConfig) error {
	report := finding.NewReport(finding.ToolInfo{Name: "cqrs-lint", Version: version})
	report.AddFindings(findings)

	switch strings.ToLower(cfg.Format) {
	case "json":
		json, err := report.PrettyJSON()
		if err != nil {
			return err
		}

		fmt.Println(json)

	case "sarif":
		err := report.WriteSARIF(ctx, os.Stdout)
		if err != nil {
			return err
		}

	case "markdown":
		groupMode := resolveGroupMode(cfg)
		if groupMode == "aggregate" || groupMode == "module" {
			printMarkdownGrouped(os.Stdout, findings, groupMode)
		} else {
			err := finding.FormatMarkdown(os.Stdout, findings)
			if err != nil {
				return err
			}
		}

	default:
		if len(findings) == 0 {
			if !cfg.Quiet {
				fmt.Println("No findings. Clean!")
			}

			return nil
		}

		cm := parseColorMode(cfg.Color)
		groupMode := resolveGroupMode(cfg)

		switch groupMode {
		case "aggregate":
			printFindingsByAggregate(os.Stdout, findings, cm)
		case "module":
			printFindingsGrouped(os.Stdout, findings, cm)
		default:
			formatFindingsText(os.Stdout, findings, cm)
		}
	}

	return nil
}

// printFindingsGrouped prints findings grouped by module directory.
// Each group is preceded by a header showing the module path and finding count.
func printFindingsGrouped(w io.Writer, findings []finding.Finding, cm output.ColorMode) {
	useColor := shouldColor(cm, w)

	groups := groupFindingsByModule(findings)

	for _, g := range groups {
		header := fmt.Sprintf("=== %s (%d) ===", g.name, len(g.findings))
		if useColor {
			header = ansiBold + ansiCyan + header + ansiReset
		}

		_, _ = fmt.Fprintln(w, header)
		formatFindingsText(w, g.findings, cm)
	}
}

type findingGroup struct {
	name     string
	findings []finding.Finding
}

func groupFindingsByModule(findings []finding.Finding) []findingGroup {
	groupMap := make(map[string][]finding.Finding)

	for _, f := range findings {
		mod := moduleFromPath(string(f.Position.File))
		groupMap[mod] = append(groupMap[mod], f)
	}

	groups := make([]findingGroup, 0, len(groupMap))
	for mod, fs := range groupMap {
		groups = append(groups, findingGroup{name: mod, findings: fs})
	}

	sort.Slice(groups, func(i, j int) bool {
		return groups[i].name < groups[j].name
	})

	return groups
}

// moduleFromPath extracts the module directory from a file path.
// Returns "root" for files directly in the project root.
func moduleFromPath(path string) string {
	dir := filepath.Dir(path)
	if dir == "." || dir == "/" {
		return "root"
	}

	return dir
}

// resolveGroupMode determines which grouping mode to use for text output.
// Explicit --group-by takes precedence. When unset, --verbose maps to "module"
// (backward compatibility). Otherwise, no grouping (flat output).
func resolveGroupMode(cfg *AppConfig) string {
	if cfg.GroupBy != "" {
		return strings.ToLower(cfg.GroupBy)
	}

	if cfg.Verbose {
		return "module"
	}

	return "none"
}

// groupFindingsByAggregate groups findings by their stamped aggregate metadata
// (Finding.Metadata["aggregate"]). Findings without an aggregate tag land in
// the "Uncategorized" bucket. Groups are sorted by finding count descending,
// then alphabetically — the most issues surface first.
func groupFindingsByAggregate(findings []finding.Finding) []findingGroup {
	const uncategorized = "Uncategorized"

	groupMap := make(map[string][]finding.Finding)

	for _, f := range findings {
		agg := uncategorized

		if f.Metadata != nil {
			if a := f.Metadata[aggregateMetadataKey]; a != "" {
				agg = a
			}
		}

		groupMap[agg] = append(groupMap[agg], f)
	}

	groups := make([]findingGroup, 0, len(groupMap))

	for name, fs := range groupMap {
		groups = append(groups, findingGroup{name: name, findings: fs})
	}

	sort.Slice(groups, func(i, j int) bool {
		if len(groups[i].findings) != len(groups[j].findings) {
			return len(groups[i].findings) > len(groups[j].findings)
		}

		return groups[i].name < groups[j].name
	})

	return groups
}

// printFindingsByAggregate prints findings grouped by aggregate/domain.
// Each group is preceded by a header showing the aggregate name and finding
// count. Groups are ordered by severity (most findings first).
func printFindingsByAggregate(w io.Writer, findings []finding.Finding, cm output.ColorMode) {
	useColor := shouldColor(cm, w)

	groups := groupFindingsByAggregate(findings)

	for _, g := range groups {
		header := fmt.Sprintf("--- %s (%d) ---", g.name, len(g.findings))
		if useColor {
			header = ansiBold + ansiCyan + header + ansiReset
		}

		_, _ = fmt.Fprintln(w, header)
		formatFindingsText(w, g.findings, cm)
	}
}

// printMarkdownGrouped renders findings as markdown with group headings.
// Each group becomes a "## GroupName (N findings)" section, with the
// findings rendered via finding.FormatMarkdown within that section.
// Supports both "aggregate" and "module" grouping modes.
func printMarkdownGrouped(w io.Writer, findings []finding.Finding, groupMode string) {
	var groups []findingGroup

	switch groupMode {
	case "aggregate":
		groups = groupFindingsByAggregate(findings)
	case "module":
		groups = groupFindingsByModule(findings)
	default:
		groups = []findingGroup{{name: "Findings", findings: findings}}
	}

	for _, g := range groups {
		fmt.Fprintf(w, "## %s (%d)\n\n", g.name, len(g.findings))
		_ = finding.FormatMarkdown(w, g.findings)
		fmt.Fprintln(w)
	}
}

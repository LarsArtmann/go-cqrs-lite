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
	"github.com/larsartmann/go-output/table"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules"
)

func parseColorMode(s string) output.ColorMode {
	switch strings.ToLower(s) {
	case "always":
		return output.ColorModeAlways
	case "never":
		return output.ColorModeNever
	default:
		return output.ColorModeAuto
	}
}

func shouldColor(cm output.ColorMode, w io.Writer) bool {
	switch cm {
	case output.ColorModeAlways:
		return true
	case output.ColorModeNever:
		return false
	case output.ColorModeAuto:
		f, ok := w.(*os.File)
		if !ok {
			return false
		}

		info, err := f.Stat()
		if err != nil {
			return false
		}

		return (info.Mode() & os.ModeCharDevice) != 0
	default:
		return false
	}
}

// ANSI color codes for terminal output.
const (
	ansiReset   = "\033[0m"
	ansiBold    = "\033[1m"
	ansiRed     = "\033[31m"
	ansiBoldRed = "\033[1;31m"
	ansiYellow  = "\033[93m"
	ansiGreen   = "\033[32m"
	ansiGray    = "\033[90m"
	ansiCyan    = "\033[36m"
)

func severityANSI(sev finding.Severity) string {
	switch sev {
	case finding.SeverityCritical:
		return ansiBoldRed
	case finding.SeverityError:
		return ansiRed
	case finding.SeverityWarning:
		return ansiYellow
	case finding.SeverityInfo:
		return ansiGreen
	default:
		return ansiReset
	}
}

func renderRulesTable(colorMode output.ColorMode) (string, error) {
	allRules := rules.AllRules()

	builder := output.NewTableBuilder().
		SetHeaders("ID", "Rule", "Severity", "Category", "Description")

	for _, r := range allRules {
		sev := strings.ToUpper(r.Severity)
		if r.AutoFix {
			sev += " *"
		}

		builder.AddRow(r.ID, r.Name, sev, r.Category, r.Description)
	}

	builder.SetFooter(fmt.Sprintf("%d rules  (* = auto-fixable)", len(allRules)))

	data := builder.Build()

	return table.Render(data, table.WithColorMode(colorMode))
}

func renderHealthScore(hs HealthScore, colorMode output.ColorMode) string {
	data := output.NewTableBuilder().
		SetHeaders("Score", "Grade").
		AddRow(fmt.Sprintf("%d/100", hs.Score), hs.Grade).
		Build()

	scoreTable, err := table.Render(data, table.WithColorMode(colorMode))
	if err != nil {
		return fmt.Sprintf("Score: %d/100 (%s)\n", hs.Score, hs.Grade)
	}

	if len(hs.Breakdown) == 0 {
		return scoreTable + "\n"
	}

	breakdownBuilder := output.NewTableBuilder().
		SetHeaders("Deduction", "Severity", "Rule")

	for _, entry := range hs.sortedBreakdown() {
		breakdownBuilder.AddRow(
			fmt.Sprintf("-%d", entry.Deduction),
			entry.Severity,
			entry.Rule,
		)
	}

	breakdownData := breakdownBuilder.Build()

	breakdownTable, err := table.Render(breakdownData, table.WithColorMode(colorMode))
	if err != nil {
		return scoreTable + "\n"
	}

	return scoreTable + "\n" + breakdownTable + "\n"
}

func formatFindingsText(w io.Writer, findings []finding.Finding, cm output.ColorMode) {
	useColor := shouldColor(cm, w)

	for _, f := range findings {
		sevStr := strings.ToUpper(f.Severity.String())
		if useColor {
			sevStr = severityANSI(f.Severity) + sevStr + ansiReset
		}

		location := fmt.Sprintf("%s:%d:%d", f.Position.File, f.Position.Line, f.Position.Column)
		if useColor {
			location = ansiGray + location + ansiReset
		}

		_, _ = fmt.Fprintf(w, "%s %s %s\n", sevStr, location, f.Message)

		if f.Rule != "" {
			ruleStr := fmt.Sprintf("  [%s]", f.Rule)
			if useColor {
				ruleStr = ansiCyan + ruleStr + ansiReset
			}

			_, _ = fmt.Fprintln(w, ruleStr)
		}

		if f.Suggestion != "" {
			_, _ = fmt.Fprintf(w, "  %sSuggestion:%s %s\n",
				val(useColor, ansiBold, ""), val(useColor, ansiReset, ""), f.Suggestion)
		}

		if f.Snippet != "" {
			snippetStr := "  |> " + f.Snippet
			if useColor {
				snippetStr = ansiGray + snippetStr + ansiReset
			}

			_, _ = fmt.Fprintln(w, snippetStr)
		}

		_, _ = fmt.Fprintln(w)
	}
}

func val(cond bool, ifTrue, ifFalse string) string {
	if cond {
		return ifTrue
	}

	return ifFalse
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
		err := finding.FormatMarkdown(os.Stdout, findings)
		if err != nil {
			return err
		}

	default:
		if len(findings) == 0 {
			if !cfg.Quiet {
				fmt.Println("No findings. Clean!")
			}

			return nil
		}

		if cfg.Verbose {
			printFindingsGrouped(os.Stdout, findings, parseColorMode(cfg.Color))
		} else {
			formatFindingsText(os.Stdout, findings, parseColorMode(cfg.Color))
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
		header := fmt.Sprintf("=== %s (%d) ===", g.module, len(g.findings))
		if useColor {
			header = ansiBold + ansiCyan + header + ansiReset
		}

		_, _ = fmt.Fprintln(w, header)
		formatFindingsText(w, g.findings, cm)
	}
}

type findingGroup struct {
	module   string
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
		groups = append(groups, findingGroup{module: mod, findings: fs})
	}

	sort.Slice(groups, func(i, j int) bool {
		return groups[i].module < groups[j].module
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

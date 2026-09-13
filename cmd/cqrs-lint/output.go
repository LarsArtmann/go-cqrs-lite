package main

// Render layering in cqrs-lint:
//   - Structured/tabular output (rules table, health-score breakdown, scorecard
//     tables) goes through go-output's table.Render, which honors ColorMode
//     (NO_COLOR/CI/FORCE_COLOR) internally via cm.ShouldColor().
//   - Findings text is a bespoke diagnostic layout (severity + location +
//     message + rule + suggestion + snippet). go-output has no styled-text API,
//     only format renderers, so raw ANSI codes are used here. Color is gated by
//     cm.ShouldColor() so findings stay consistent with the tables under
//     NO_COLOR/CI/FORCE_COLOR — do NOT reintroduce a hand-rolled terminal check.

import (
	"encoding/json/v2"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/larsartmann/go-finding"
	output "github.com/larsartmann/go-output"
	"github.com/larsartmann/go-output/table"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules"
)

// parseColorMode resolves a --color flag value into a go-output ColorMode.
// It delegates to output.ParseColorMode so the accepted values, case mapping,
// and error type stay in lockstep with the library that actually consumes the
// mode. Invalid input falls back to ColorModeAuto (lenient, preserves prior
// behavior) instead of failing the whole run over a cosmetic flag.
func parseColorMode(s string) output.ColorMode {
	cm, err := output.ParseColorMode(strings.ToLower(s))
	if err != nil {
		return output.ColorModeAuto
	}

	return cm
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
	scoreLabel := fmt.Sprintf("%d/100", hs.Score)
	if hs.Score == 0 && hs.RawScore < 0 {
		scoreLabel = fmt.Sprintf("%d/100 (clamped from %d)", hs.Score, hs.RawScore)
	}

	data := output.NewTableBuilder().
		SetHeaders("Score", "Grade").
		AddRow(scoreLabel, hs.Grade).
		Build()

	scoreTable, err := table.Render(data, table.WithColorMode(colorMode))
	if err != nil {
		return fmt.Sprintf("Score: %s (%s)\n", scoreLabel, hs.Grade)
	}

	if len(hs.Breakdown) == 0 {
		return scoreTable + "\n" + formatConfigExcluded(hs.ConfigExcluded)
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

	result := scoreTable + "\n" + breakdownTable + "\n"

	if hs.InfoCapped {
		result += fmt.Sprintf("Info deductions capped: raw -%d → capped at -%d\n",
			hs.InfoRawDeduction, hs.InfoCapApplied)
	}

	result += formatConfigExcluded(hs.ConfigExcluded)

	return result
}

// formatConfigExcluded renders the config-disabled-rule transparency footer
// for the health score. Disabled rules that would have fired are listed with
// their dropped-finding count so score inflation via config disables is
// always visible. Returns "" when nothing was excluded.
func formatConfigExcluded(configExcluded map[string]int) string {
	if len(configExcluded) == 0 {
		return ""
	}

	rules := make([]string, 0, len(configExcluded))
	for rule := range configExcluded {
		rules = append(rules, rule)
	}
	slices.Sort(rules)

	entries := make([]string, 0, len(rules))
	for _, rule := range rules {
		entries = append(entries, fmt.Sprintf("%s (%d)", rule, configExcluded[rule]))
	}

	return fmt.Sprintf("Excluded from score by config: %s\n", strings.Join(entries, ", "))
}

// findingsToTable models findings as a flat output.Table for delimited formats
// (CSV/TSV). Each finding becomes one row with structured columns suitable for
// spreadsheet analysis and CI pipeline ingestion. Snippet is excluded because
// it is often multi-line and a poor fit for a single delimited cell.
func findingsToTable(findings []finding.Finding) *output.Table {
	builder := output.NewTableBuilder().
		SetHeaders("Rule", "Severity", "File", "Line", "Column", "Message", "Suggestion", "Category", "Confidence")

	for _, f := range findings {
		builder.AddRow(
			string(f.Rule),
			f.Severity.String(),
			string(f.Position.File),
			strconv.Itoa(f.Position.Line),
			strconv.Itoa(f.Position.Column),
			f.Message,
			f.Suggestion,
			f.Category.String(),
			f.Confidence.String(),
		)
	}

	return builder.Build()
}

func formatFindingsText(w io.Writer, findings []finding.Finding, cm output.ColorMode) {
	useColor := cm.ShouldColor()

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

// ruleJSON is the editor/tooling view of one catalog entry, emitted by
// `cqrs-lint rules --json`. Field names are stable API for consumers.
type ruleJSON struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	Severity    string `json:"severity"`
	Confidence  string `json:"confidence"`
	Description string `json:"description"`
	AutoFix     bool   `json:"autoFix"`
	DocURL      string `json:"docUrl,omitempty"`
}

func renderRulesJSON() (string, error) {
	allRules := rules.AllRules()

	entries := make([]ruleJSON, 0, len(allRules))
	for _, r := range allRules {
		entries = append(entries, ruleJSON{
			ID:          r.ID,
			Name:        r.Name,
			Category:    r.Category,
			Severity:    r.Severity,
			Confidence:  r.Confidence,
			Description: r.Description,
			AutoFix:     r.AutoFix,
			DocURL:      r.DocURL,
		})
	}

	data, err := json.Marshal(entries)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

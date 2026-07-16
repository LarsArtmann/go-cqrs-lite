package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/larsartmann/go-finding"
)

// ANSI color codes for terminal output.
const (
	colorReset   = "\033[0m"
	colorRed     = "\033[31m"
	colorOrange  = "\033[33m"
	colorYellow  = "\033[93m"
	colorGreen   = "\033[32m"
	colorGray    = "\033[90m"
	colorBoldRed = "\033[1;31m"
)

// shouldUseColor determines if colored output should be used.
func shouldUseColor(colorMode string, w io.Writer) bool {
	switch strings.ToLower(colorMode) {
	case "always":
		return true
	case "never":
		return false
	default: // "auto"
		f, ok := w.(*os.File)
		if !ok {
			return false
		}
		info, err := f.Stat()
		if err != nil {
			return false
		}
		return (info.Mode() & os.ModeCharDevice) != 0
	}
}

// severityColor returns the ANSI color for a given severity.
func severityColor(sev finding.Severity) string {
	switch sev {
	case finding.SeverityCritical:
		return colorBoldRed
	case finding.SeverityError:
		return colorRed
	case finding.SeverityWarning:
		return colorYellow
	case finding.SeverityInfo:
		return colorGreen
	default:
		return colorReset
	}
}

// formatColoredText writes colored text output to the given writer.
func formatColoredText(w io.Writer, findings []finding.Finding, colorMode string) {
	useColor := shouldUseColor(colorMode, w)

	for _, f := range findings {
		sevStr := f.Severity.String()
		if useColor {
			color := severityColor(f.Severity)
			sevStr = color + strings.ToUpper(sevStr) + colorReset
		} else {
			sevStr = strings.ToUpper(sevStr)
		}

		location := fmt.Sprintf("%s:%d:%d", f.Position.File, f.Position.Line, f.Position.Column)
		if useColor {
			location = colorGray + location + colorReset
		}

		_, _ = fmt.Fprintf(w, "%s %s %s\n", sevStr, location, f.Message)

		if f.Rule != "" {
			ruleStr := fmt.Sprintf("  [%s]", f.Rule)
			if useColor {
				ruleStr = colorGray + ruleStr + colorReset
			}
			_, _ = fmt.Fprintln(w, ruleStr)
		}

		if f.Suggestion != "" {
			_, _ = fmt.Fprintf(w, "  Suggestion: %s\n", f.Suggestion)
		}

		if f.Snippet != "" {
			snippetStr := "  |> " + f.Snippet
			if useColor {
				snippetStr = colorGray + snippetStr + colorReset
			}
			_, _ = fmt.Fprintln(w, snippetStr)
		}

		_, _ = fmt.Fprintln(w)
	}
}

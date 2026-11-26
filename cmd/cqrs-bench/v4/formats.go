package main

import (
	"fmt"
	"slices"
	"strings"
)

// Per-command --format vocabularies. Handlers validate before any benchmark
// work starts; an unknown or unsupported value used to fall through the
// render switches' default branch and silently render plain text, after the
// benchmark had already run.

// runFormats are the formats of the run command.
func runFormats() []string {
	return []string{
		formatAuto, formatTable, formatText, formatJSON, formatCSV,
		formatTSV, formatMarkdown, formatBenchstat, formatManifest,
	}
}

// reportFormats are the formats of compare, sweep, and soak: every run
// format except the two that describe a single run's configuration.
func reportFormats() []string {
	return []string{
		formatAuto, formatTable, formatText, formatJSON, formatCSV,
		formatTSV, formatMarkdown,
	}
}

// layoutFormats are the formats of the layout command.
func layoutFormats() []string {
	return []string{formatAuto, formatTable, formatText, formatJSON}
}

// validateFormat rejects a format the command cannot render, naming the
// ones it can. Called before any work starts so a typo fails in
// milliseconds instead of after minutes of benchmarking.
func validateFormat(command, format string, allowed []string) error {
	if slices.Contains(allowed, format) {
		return nil
	}

	return fmt.Errorf("%w %q for %s (supported: %s)",
		errFormat, format, command, strings.Join(allowed, ", "))
}

package main

import (
	"slices"
	"strings"
)

// Per-subcommand --format vocabularies. Handlers validate before any
// benchmark work starts; previously an unknown or unsupported value fell
// through the render switches' default branch and silently rendered plain
// text — after the benchmark had already run.
var (
	//nolint:gochecknoglobals // static vocabulary
	runFormats = []string{
		formatAuto, formatTable, formatText, formatJSON, formatCSV,
		formatTSV, formatMarkdown, formatBenchstat, formatManifest,
	}
	//nolint:gochecknoglobals // static vocabulary
	compareFormats = []string{
		formatAuto, formatTable, formatText, formatJSON, formatCSV,
		formatTSV, formatMarkdown,
	}
	//nolint:gochecknoglobals // static vocabulary
	soakFormats = []string{
		formatAuto, formatTable, formatText, formatJSON, formatCSV,
		formatTSV, formatMarkdown,
	}
	//nolint:gochecknoglobals // static vocabulary
	layoutFormats = []string{formatAuto, formatTable, formatText, formatJSON}
)

// validateFormat fatals with a teaching message when format is not one of
// allowed. Called as the first statement of each handler so a typo fails in
// milliseconds instead of after minutes of benchmarking. compare and sweep
// share compareFormats — their renderers support the same set.
func validateFormat(command, format string, allowed []string) {
	if slices.Contains(allowed, format) {
		return
	}

	fatalf(
		"invalid --format %q for %s (supported: %s)",
		format, command, strings.Join(allowed, ", "),
	)
}

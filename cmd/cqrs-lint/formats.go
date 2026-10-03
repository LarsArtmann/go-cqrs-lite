package main

import "strings"

// Per-command --format vocabularies — the single source of truth for which
// output formats each command supports. Validation calls, command help
// strings, and the explain config table all derive from these slices, so
// adding a format to a command edits exactly one line and every surface
// follows. Order matters: it is the order used in error messages and help.
//
//nolint:gochecknoglobals // read-only lookup tables
var (
	formatsLint      = []string{"text", "json", "sarif", "markdown", "csv", "tsv"}
	formatsScorecard = []string{"text", "json", "markdown", "sarif"}
	formatsDoctor    = []string{"text", "json"}
	formatsRules     = []string{"text", "json", "markdown"}
)

// formatList renders a vocabulary as help text: "text, json".
func formatList(formats []string) string {
	return strings.Join(formats, ", ")
}

// withFormatsSuffix appends "; formats: a, b, c" to a command's short
// description so the advertised list can never drift from the validated one.
func withFormatsSuffix(short string, formats []string) string {
	return short + "; formats: " + formatList(formats)
}

package main

import (
	"fmt"
	"io"
	"strings"

	cqrsanalyzer "github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	cqrsversion "github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/version"
	"github.com/larsartmann/go-finding"
)

// deprecationReport runs the cqrs-lint V007 detector (v5-removed API usage)
// in-process over dir and prints a per-finding report. Best-effort: a load
// failure becomes a warning, never an upgrade abort.
func deprecationReport(w io.Writer, dir string) error {
	ctx, err := cqrsanalyzer.BuildContext(dir)
	if err != nil {
		fmt.Fprintf(w, "deprecation report skipped (analysis failed): %v\n", err)

		return nil
	}

	detector := cqrsversion.NewV007Detector(ctx)

	findings, err := detector.Detect(ctx)
	if err != nil {
		fmt.Fprintf(w, "deprecation report skipped (detector failed): %v\n", err)

		return nil
	}

	writeFindings(w, findings)

	return nil
}

// writeFindings prints V007 findings sorted by file for stable output.
func writeFindings(w io.Writer, findings []finding.Finding) {
	if len(findings) == 0 {
		fmt.Fprintln(w, "deprecation report: no v5-removed API usage detected")

		return
	}

	fmt.Fprintf(w, "deprecation report: %d finding(s) — APIs removed at go-cqrs-lite v5:\n", len(findings))

	for _, f := range findings {
		line := ""

		if f.Location() != nil {
			line = fmt.Sprintf("%s:%d", f.Location().Filename, f.Location().Line)
		}

		fmt.Fprintf(w, "  %s %s %s\n", line, strings.TrimSpace(f.Message()), f.Rule().ID())
	}
}

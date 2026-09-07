package main

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/larsartmann/go-finding"

	cqrsanalyzer "github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	cqrsversion "github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/version"
)

// deprecationReport runs the cqrs-lint V007 detector (v5-removed API usage)
// in-process over dir and prints a per-finding report. Best-effort: a load
// failure becomes a warning, never an upgrade abort.
func deprecationReport(w io.Writer, dir string) {
	ctx, err := cqrsanalyzer.BuildContext(dir)
	if err != nil {
		fmt.Fprintf(w, "deprecation report skipped (analysis failed): %v\n", err)

		return
	}

	findings, detErr := cqrsversion.NewV007Detector(ctx).Detect(context.Background())
	if detErr != nil {
		fmt.Fprintf(w, "deprecation report skipped (detector failed): %v\n", detErr)

		return
	}

	writeFindings(w, findings)
}

// writeFindings prints V007 findings sorted by position for stable output.
func writeFindings(w io.Writer, findings []finding.Finding) {
	if len(findings) == 0 {
		fmt.Fprintln(w, "deprecation report: no v5-removed API usage detected")

		return
	}

	fmt.Fprintf(
		w,
		"deprecation report: %d finding(s) — APIs removed at go-cqrs-lite v5:\n",
		len(findings),
	)

	sorted := append([]finding.Finding(nil), findings...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Position.String() < sorted[j].Position.String()
	})

	for _, f := range sorted {
		fmt.Fprintf(w, "  %s %s [%s]\n",
			f.Position.String(), strings.TrimSpace(f.Message), f.Rule)
	}
}

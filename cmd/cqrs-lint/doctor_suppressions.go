package main

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// renderDoctorSuppressions counts and displays inline //cqrs-lint:ignore comments.
func renderDoctorSuppressions(w io.Writer, actx *analyzer.AnalysisContext) {
	suppressionCounts := countSuppressions(actx)
	if len(suppressionCounts) == 0 {
		return
	}

	_, _ = fmt.Fprintln(w, "INLINE SUPPRESSIONS")
	_, _ = fmt.Fprintln(w, "───────────────────")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "  //cqrs-lint:ignore(RULE) counts per rule:")
	_, _ = fmt.Fprintln(w)

	type suppressionEntry struct {
		rule  string
		count int
	}

	entries := make([]suppressionEntry, 0, len(suppressionCounts))
	for rule, count := range suppressionCounts {
		entries = append(entries, suppressionEntry{rule, count})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].count > entries[j].count
	})

	for _, e := range entries {
		_, _ = fmt.Fprintf(w, "    %-8s  %d suppressed\n", e.rule, e.count)
	}

	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(
		w,
		"  High suppression counts may signal a rule heuristic that needs tuning.",
	)
	_, _ = fmt.Fprintln(w, "  Consider reporting these as false positives.")
	_, _ = fmt.Fprintln(w)
}

// countSuppressions scans all Go files for //cqrs-lint:ignore(RULE) comments
// and returns a map of rule ID to suppression count.
func countSuppressions(actx *analyzer.AnalysisContext) map[string]int {
	counts := make(map[string]int)

	for _, gf := range actx.GoFiles {
		if gf.AST == nil {
			continue
		}

		for _, group := range gf.AST.Comments {
			for _, c := range group.List {
				text := c.Text
				idx := strings.Index(text, "cqrs-lint:ignore(")
				if idx < 0 {
					continue
				}

				start := idx + len("cqrs-lint:ignore(")
				end := strings.Index(text[start:], ")")
				if end < 0 {
					continue
				}

				rule := strings.TrimSpace(text[start : start+end])
				if rule != "" {
					counts[rule]++
				}
			}
		}
	}

	return counts
}

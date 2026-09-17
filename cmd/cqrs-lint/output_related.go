package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/larsartmann/go-finding"
	output "github.com/larsartmann/go-output"
)

// printRelatedGroups renders the GroupID summary section after the flat
// finding list in text mode: one header per group (sorted by group ID) with
// every member location. Multi-site rules (e.g. C019's duplicate Repository
// instances for one state type) emit findings that only make sense together;
// the group section makes the shared root cause visible.
func printRelatedGroups(w io.Writer, findings []finding.Finding, cm output.ColorMode) {
	groups := relatedGroups(findings)
	if groups == nil {
		return
	}

	useColor := cm.ShouldColor()

	header := fmt.Sprintf(
		"Related findings (%d group(s) — findings share a root cause):",
		len(groups),
	)
	if useColor {
		header = ansiBold + header + ansiReset
	}

	_, _ = fmt.Fprintln(w, header)

	for _, g := range groups {
		title := fmt.Sprintf("  %s (%d finding(s))", g.ID, len(g.Findings))
		if useColor {
			title = ansiCyan + title + ansiReset
		}

		_, _ = fmt.Fprintln(w, title)

		for _, f := range g.Findings {
			_, _ = fmt.Fprintf(w, "    %s %s:%d:%d  %s\n",
				strings.ToUpper(f.Severity.String()),
				f.Position.File, f.Position.Line, f.Position.Column, f.Message)
		}
	}

	_, _ = fmt.Fprintln(w)
}

// relatedGroups returns the sorted GroupID groups, or nil when no finding
// carries a GroupID.
func relatedGroups(findings []finding.Finding) []finding.Group {
	report := finding.NewReportFromFindings(
		finding.ToolInfo{Name: "cqrs-lint"},
		findings,
	)

	return report.GroupFindingsSorted()
}

// relatedGroupsMarkdown renders the GroupID summary as a markdown section,
// or false when no finding carries a GroupID.
func relatedGroupsMarkdown(findings []finding.Finding) (string, bool) {
	groups := relatedGroups(findings)
	if groups == nil {
		return "", false
	}

	var b strings.Builder

	fmt.Fprintf(&b, "## Related findings (%d group(s))\n\n", len(groups))

	for _, g := range groups {
		fmt.Fprintf(&b, "### `%s` (%d finding(s))\n\n", g.ID, len(g.Findings))

		for _, f := range g.Findings {
			fmt.Fprintf(&b, "- `%s:%d:%d` — %s\n",
				f.Position.File, f.Position.Line, f.Position.Column, f.Message)
		}

		b.WriteString("\n")
	}

	return b.String(), true
}

// printCorrelations renders the --correlate summary: pairs of findings the
// pipeline considers related (same file, nearby lines), nearest-first. It is
// a triage aid on stderr; machine formats stay untouched so JSON/SARIF
// consumers see finding documents, not cross-references.
func printCorrelations(w io.Writer, correlations []finding.Correlation, findings []finding.Finding) {
	if len(correlations) == 0 {
		return
	}

	byID := make(map[finding.ID]finding.Finding, len(findings))
	for _, f := range findings {
		byID[f.ID] = f
	}

	fmt.Fprintf(w, "Correlations (%d pair(s) of related findings):\n", len(correlations))

	for _, c := range correlations {
		locations := make([]string, 0, len(c.FindingIDs))
		for _, id := range c.FindingIDs {
			f, ok := byID[id]
			if !ok {
				locations = append(locations, string(id))
				continue
			}
			locations = append(locations, fmt.Sprintf("%s:%d %s",
				f.Position.File, f.Position.Line, f.Rule))
		}

		fmt.Fprintf(w, "  %s\n    %s (score %.2f)\n",
			strings.Join(locations, " + "), c.Reason, c.Score)
	}

	fmt.Fprintln(w)
}

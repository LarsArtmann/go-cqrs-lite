package main

import (
	"testing"

	"github.com/larsartmann/go-finding"
)

// toolFinding builds a finding for the given tool at a line. go-finding's
// Correlate skips same-tool pairs by design (it exists to relate findings
// ACROSS tools), so the fixtures vary ToolName.
func toolFinding(tool, rule string, line int) finding.Finding {
	return finding.NewBuilder(
		finding.RuleName(rule), finding.ToolName(tool), "message "+rule,
		finding.SeverityWarning,
		finding.Pos("main.go", line, 1),
	).MustBuild()
}

// TestCorrelateNearestFirstOrdering pins the proximity-comparator contract
// that go-finding v1.11.0 fixed (8a9b7c8, released after v1.10.0 shipped
// with the bug): nearby-line pairs must surface in ascending line-distance
// order, nearest first. The previous subtraction comparator could wrap and
// silently scramble that order.
//
// This test runs against the go-finding version cqrs-lint is built against,
// so a future downgrade or regression in the dependency fails HERE, in the
// consumer's suite, not only upstream. Same-tool pairs are excluded by
// upstream design (Correlate relates findings across tools), hence the
// alternating tool names.
func TestCorrelateNearestFirstOrdering(t *testing.T) {
	t.Parallel()

	findings := []finding.Finding{
		toolFinding("cqrs-lint", "C001", 100),
		toolFinding("art-dupl", "DUP", 101),
		toolFinding("cqrs-lint", "C002", 103),
		toolFinding("art-dupl", "DUP2", 104),
	}

	correlations := finding.Correlate(findings)
	if len(correlations) < 3 {
		t.Fatalf("expected at least 3 nearby-line correlations, got %d", len(correlations))
	}

	lineOf := make(map[finding.ID]int, len(findings))
	for _, f := range findings {
		lineOf[f.ID] = f.Position.Line
	}

	prev := -1
	for _, c := range correlations {
		if len(c.FindingIDs) != 2 {
			t.Fatalf("expected 2 findings per correlation, got %v", c.FindingIDs)
		}

		dist := lineOf[c.FindingIDs[0]] - lineOf[c.FindingIDs[1]]
		if dist < 0 {
			dist = -dist
		}

		if dist > 5 {
			t.Fatalf("correlated pair beyond maxLineDiff=5: distance %d", dist)
		}

		if prev >= 0 && dist < prev {
			t.Fatalf(
				"correlations not nearest-first: distance %d after %d",
				dist, prev,
			)
		}
		prev = dist
	}
}

// TestCorrelateSkipsSameToolPairs pins the second upstream contract
// cqrs-lint relies on when deciding NOT to build a --correlate surface:
// same-tool findings (all of cqrs-lint's own output) never correlate, so
// intra-linter rule overlap must be solved by rule design, not correlation.
func TestCorrelateSkipsSameToolPairs(t *testing.T) {
	t.Parallel()

	findings := []finding.Finding{
		toolFinding("cqrs-lint", "C001", 100),
		toolFinding("cqrs-lint", "C010", 101),
		toolFinding("cqrs-lint", "C025", 103),
	}

	if got := finding.Correlate(findings); len(got) != 0 {
		t.Fatalf("same-tool findings must not correlate, got %+v", got)
	}
}

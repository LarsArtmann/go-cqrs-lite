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
// with the bug): findings are pre-sorted by line (cmp.Compare, no
// subtraction wrap), every emitted pair is within maxLineDiff=5 with the
// lower line first, and each pair's score equals 1 - dist/5 — so score
// order matches distance order for every anchor. A comparator that wraps
// or scrambles the sort breaks the score/distance agreement, which is
// exactly what a triage UI reading correlation scores depends on.
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

	for _, c := range correlations {
		if len(c.FindingIDs) != 2 {
			t.Fatalf("expected 2 findings per correlation, got %v", c.FindingIDs)
		}

		la, lb := lineOf[c.FindingIDs[0]], lineOf[c.FindingIDs[1]]
		dist := la - lb
		if dist < 0 {
			dist = -dist
		}

		if dist == 0 || dist > 5 {
			t.Fatalf("correlated pair outside (0, maxLineDiff=5]: distance %d", dist)
		}

		want := 1.0 - float64(dist)/5.0
		if diff := float64(c.Score) - want; diff > 1e-9 || diff < -1e-9 {
			t.Fatalf(
				"score %.2f inconsistent with distance %d (want %.2f) — comparator wrap?",
				c.Score, dist, want,
			)
		}
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

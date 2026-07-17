package main

import (
	"strings"
	"testing"

	"github.com/larsartmann/go-finding"
)

func TestComputeHealthScore_Empty(t *testing.T) {
	t.Parallel()

	hs := ComputeHealthScore(nil)
	if hs.Score != 100 {
		t.Errorf("empty findings: expected score 100, got %d", hs.Score)
	}
	if hs.Grade != "Excellent" {
		t.Errorf("empty findings: expected grade Excellent, got %s", hs.Grade)
	}
}

func TestComputeHealthScore_WithFindings(t *testing.T) {
	t.Parallel()

	findings := createTestFindings()
	hs := ComputeHealthScore(findings)

	if hs.Score >= 100 {
		t.Error("findings should reduce score below 100")
	}
	if hs.Score < 0 {
		t.Error("score should not go below 0")
	}
}

func TestComputeHealthScore_Floor(t *testing.T) {
	t.Parallel()

	var findings []finding.Finding
	for i := 0; i < 100; i++ {
		f, _ := finding.NewBuilder(
			"C001", "test",
			"critical issue",
			finding.SeverityCritical,
			finding.Pos(finding.FilePath("test.go"), 1, 1),
		).
			Build()
		findings = append(findings, f)
	}

	hs := ComputeHealthScore(findings)
	if hs.Score != 0 {
		t.Errorf("100 critical findings should floor to 0, got %d", hs.Score)
	}
	if hs.Grade != "Needs Improvement" {
		t.Errorf("score 0 should be Needs Improvement, got %s", hs.Grade)
	}
}

func TestRenderHealthScore(t *testing.T) {
	t.Parallel()

	hs := HealthScore{Score: 85, Grade: "Good", Breakdown: map[string]int{"error C001": 5}}
	out := renderHealthScore(hs, parseColorMode("never"))

	if !strings.Contains(out, "85/100") {
		t.Error("output should contain score")
	}
	if !strings.Contains(out, "Good") {
		t.Error("output should contain grade")
	}
	if !strings.Contains(out, "C001") {
		t.Error("output should contain rule from breakdown")
	}
}

func TestComputeHealthScore_Grades(t *testing.T) {
	t.Parallel()

	tests := []struct {
		score  int
		grade  string
		findFn func() HealthScore
	}{
		{95, "Excellent", func() HealthScore { return ComputeHealthScore(nil) }},
	}

	_ = tests

	cases := []struct {
		findingsCount int
		severity      finding.Severity
		wantGrade     string
	}{
		{0, finding.SeverityInfo, "Excellent"},
		{11, finding.SeverityCritical, "Needs Improvement"},
	}

	for _, tc := range cases {
		var findings []finding.Finding
		for i := 0; i < tc.findingsCount; i++ {
			f, _ := finding.NewBuilder(
				"TEST", "test", "test",
				tc.severity,
				finding.Pos(finding.FilePath("test.go"), 1, 1),
			).Build()
			findings = append(findings, f)
		}
		hs := ComputeHealthScore(findings)
		if hs.Grade != tc.wantGrade {
			t.Errorf("findings=%d severity=%s: expected grade %s, got %s",
				tc.findingsCount, tc.severity, tc.wantGrade, hs.Grade)
		}
	}
}

func createTestFindings() []finding.Finding {
	severities := []finding.Severity{
		finding.SeverityCritical,
		finding.SeverityError,
		finding.SeverityWarning,
		finding.SeverityInfo,
	}
	findings := make([]finding.Finding, 0, len(severities))

	for _, sev := range severities {
		f, _ := finding.NewBuilder(
			"TEST", "test",
			"test finding",
			sev,
			finding.Pos(finding.FilePath("test.go"), 1, 1),
		).
			Build()
		findings = append(findings, f)
	}

	return findings
}

// Confidence weighting: a Low-confidence finding costs less than a High one.
// Guards against a flood of Low-confidence heuristic matches dominating the
// score. Covers item f-16 in the DiscordSync feedback triage.
func TestComputeHealthScore_ConfidenceWeighting(t *testing.T) {
	t.Parallel()

	high, _ := finding.NewBuilder(
		"C001", "test", "sure bug",
		finding.SeverityCritical,
		finding.Pos(finding.FilePath("test.go"), 1, 1),
	).WithConfidence(finding.ConfidenceHigh).Build()

	low, _ := finding.NewBuilder(
		"C002", "test", "maybe bug",
		finding.SeverityCritical,
		finding.Pos(finding.FilePath("test.go"), 1, 1),
	).WithConfidence(finding.ConfidenceLow).Build()

	hs := ComputeHealthScore([]finding.Finding{high, low})

	// High (1.0): 10, Low (0.5): 5 → total 15 → score 85.
	if hs.Score != 85 {
		t.Errorf("High(10)+Low(5): expected score 85, got %d", hs.Score)
	}
}

// Info cap: many Info-severity findings can't tank the score. 50 info findings
// would naïvely cost -50; the 20% cap limits them to -20. Covers item f-15 in
// the DiscordSync feedback triage (18 D002 infos out-scoring real bugs).
func TestComputeHealthScore_InfoCap(t *testing.T) {
	t.Parallel()

	findings := make([]finding.Finding, 0, 50)

	for i := 0; i < 50; i++ {
		f, _ := finding.NewBuilder(
			"D002", "test", "style nit",
			finding.SeverityInfo,
			finding.Pos(finding.FilePath("test.go"), 1, 1),
		).Build()
		findings = append(findings, f)
	}

	hs := ComputeHealthScore(findings)

	// 50 info * 1.0 weight = 50, capped at 20 → score 80.
	if hs.Score != 80 {
		t.Errorf("50 info findings capped at 20: expected score 80, got %d", hs.Score)
	}
}

// The Info cap must NOT suppress Critical/Error/warning deductions. A project
// with a real bug AND lots of info noise should still feel the bug.
func TestComputeHealthScore_InfoCapDoesNotAffectHigherSeverities(t *testing.T) {
	t.Parallel()

	var findings []finding.Finding

	critical, _ := finding.NewBuilder(
		"C001", "test", "real bug",
		finding.SeverityCritical,
		finding.Pos(finding.FilePath("test.go"), 1, 1),
	).Build()
	findings = append(findings, critical)

	for i := 0; i < 50; i++ {
		f, _ := finding.NewBuilder(
			"D002", "test", "style nit",
			finding.SeverityInfo,
			finding.Pos(finding.FilePath("test.go"), 1, 1),
		).Build()
		findings = append(findings, f)
	}

	hs := ComputeHealthScore(findings)

	// Critical (10) + info capped (20) = 30 → score 70.
	if hs.Score != 70 {
		t.Errorf("critical + capped info: expected score 70, got %d", hs.Score)
	}
}

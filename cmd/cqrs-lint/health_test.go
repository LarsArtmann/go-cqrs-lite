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

// The tunable cap: consumers can raise/lower it via .cqrs-lint.json.
// A cap of 5 means 50 info findings cost only -5, not -20.
func TestComputeHealthScore_TunableCap(t *testing.T) {
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

	hsDefault := ComputeHealthScore(findings)
	hsTight := ComputeHealthScoreWithCap(findings, 5)

	if hsDefault.Score != 80 {
		t.Errorf("default cap: expected 80, got %d", hsDefault.Score)
	}
	// Cap 5 → 50 info capped at 5 → score 95.
	if hsTight.Score != 95 {
		t.Errorf("cap=5: expected 95, got %d", hsTight.Score)
	}
}

// When the Info cap truncates deductions, InfoCapped + InfoRawDeduction expose
// the uncapped total for verbose-mode transparency.
func TestComputeHealthScore_InfoCappedTransparency(t *testing.T) {
	t.Parallel()

	findings := make([]finding.Finding, 0, 30)
	for i := 0; i < 30; i++ {
		f, _ := finding.NewBuilder(
			"D002", "test", "style nit",
			finding.SeverityInfo,
			finding.Pos(finding.FilePath("test.go"), 1, 1),
		).Build()
		findings = append(findings, f)
	}

	hs := ComputeHealthScore(findings)

	if !hs.InfoCapped {
		t.Error("expected InfoCapped=true when 30 info findings exceed the 20 cap")
	}
	if hs.InfoRawDeduction != 30 {
		t.Errorf("InfoRawDeduction = %d, want 30", hs.InfoRawDeduction)
	}

	// Under the cap → no transparency fields set.
	small, _ := finding.NewBuilder(
		"D002", "test", "style nit",
		finding.SeverityInfo,
		finding.Pos(finding.FilePath("test.go"), 1, 1),
	).Build()
	hsSmall := ComputeHealthScore([]finding.Finding{small})
	if hsSmall.InfoCapped {
		t.Error("expected InfoCapped=false when info is under the cap")
	}
}

// When the Info cap is customized, the HealthScore must carry the actual cap
// used so renderHealthScore shows the correct value (not the hardcoded default).
func TestComputeHealthScore_InfoCapAppliedReflectsCustomCap(t *testing.T) {
	t.Parallel()

	findings := make([]finding.Finding, 0, 30)
	for i := 0; i < 30; i++ {
		f, _ := finding.NewBuilder(
			"D002", "test", "style nit",
			finding.SeverityInfo,
			finding.Pos(finding.FilePath("test.go"), 1, 1),
		).Build()
		findings = append(findings, f)
	}

	hsDefault := ComputeHealthScore(findings)
	hsCustom := ComputeHealthScoreWithCap(findings, 10)

	if hsDefault.InfoCapApplied != defaultInfoDeductionCap {
		t.Errorf("default cap: InfoCapApplied = %d, want %d",
			hsDefault.InfoCapApplied, defaultInfoDeductionCap)
	}
	if hsCustom.InfoCapApplied != 10 {
		t.Errorf("custom cap=10: InfoCapApplied = %d, want 10", hsCustom.InfoCapApplied)
	}

	out := renderHealthScore(hsCustom, parseColorMode("never"))
	if !strings.Contains(out, "capped at -10") {
		t.Errorf("display should show custom cap -10, got:\n%s", out)
	}
	if strings.Contains(out, "capped at -20") {
		t.Error("display should NOT show default cap -20 when custom cap=10 was used")
	}
}

// The health score must be computed on ALL unsuppressed findings, not the
// severity-filtered display set. This test proves the invariant matters:
// filtering out warnings/info before computing gives a misleadingly high score.
func TestComputeHealthScore_ProducesDifferentResultOnFilteredSet(t *testing.T) {
	t.Parallel()

	var findings []finding.Finding

	critical, _ := finding.NewBuilder(
		"C001", "test", "real bug",
		finding.SeverityCritical,
		finding.Pos(finding.FilePath("test.go"), 1, 1),
	).Build()
	findings = append(findings, critical)

	warning, _ := finding.NewBuilder(
		"C002", "test", "warning issue",
		finding.SeverityWarning,
		finding.Pos(finding.FilePath("test.go"), 1, 1),
	).Build()
	findings = append(findings, warning)

	info, _ := finding.NewBuilder(
		"D002", "test", "style nit",
		finding.SeverityInfo,
		finding.Pos(finding.FilePath("test.go"), 1, 1),
	).Build()
	findings = append(findings, info)

	// Full set (what the health score SHOULD see).
	fullScore := ComputeHealthScore(findings).Score

	// Simulate --min-severity=critical filtering out the warning+info.
	errorOnly := filterBySeverity(findings, "critical")
	filteredScore := ComputeHealthScore(errorOnly).Score

	if fullScore == filteredScore {
		t.Errorf("health score on full set (%d) should differ from filtered set (%d) "+
			"— if equal, the severity filter is leaking into health score",
			fullScore, filteredScore)
	}
	if filteredScore > fullScore {
		// Filtered set has fewer findings → higher score. This proves the fix
		// matters: without it, --min-severity would inflate the health score.
		return
	}

	t.Errorf("filtered score (%d) should be higher than full score (%d)", filteredScore, fullScore)
}

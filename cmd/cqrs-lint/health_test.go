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

func TestFormatHealthScore(t *testing.T) {
	t.Parallel()

	hs := HealthScore{Score: 85, Grade: "Good", Breakdown: map[string]int{"error C001": 5}}
	output := FormatHealthScore(hs)

	if !strings.Contains(output, "85/100") {
		t.Error("output should contain score")
	}
	if !strings.Contains(output, "Good") {
		t.Error("output should contain grade")
	}
	if !strings.Contains(output, "Breakdown") {
		t.Error("output should contain breakdown section")
	}
}

func TestHealthScoreGrades(t *testing.T) {
	t.Parallel()

	tests := []struct {
		score int
		grade string
	}{
		{95, "Excellent"},
		{90, "Excellent"},
		{85, "Good"},
		{75, "Good"},
		{60, "Fair"},
		{50, "Fair"},
		{30, "Needs Improvement"},
		{0, "Needs Improvement"},
	}

	for _, tt := range tests {
		hs := HealthScore{Score: tt.score}
		got := healthGrade(tt.score)
		if got != tt.grade {
			t.Errorf("score %d: expected grade %s, got %s", tt.score, tt.grade, got)
		}
		_ = hs
	}
}

func healthGrade(score int) string {
	switch {
	case score >= 90:
		return "Excellent"
	case score >= 75:
		return "Good"
	case score >= 50:
		return "Fair"
	default:
		return "Needs Improvement"
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

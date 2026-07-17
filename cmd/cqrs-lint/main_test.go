package main

import (
	"context"
	"strings"
	"testing"

	"github.com/larsartmann/go-finding"
)

func TestFilterBySeverity(t *testing.T) {
	t.Parallel()

	findings := []finding.Finding{
		{Severity: finding.SeverityCritical},
		{Severity: finding.SeverityError},
		{Severity: finding.SeverityWarning},
		{Severity: finding.SeverityInfo},
	}

	tests := []struct {
		minSev    string
		wantCount int
	}{
		{"info", 4},
		{"warning", 3},
		{"error", 2},
		{"critical", 1},
	}

	for _, tt := range tests {
		result := filterBySeverity(findings, tt.minSev)
		if len(result) != tt.wantCount {
			t.Errorf("filterBySeverity(%q): got %d, want %d", tt.minSev, len(result), tt.wantCount)
		}
	}
}

func TestFilterByConfidence(t *testing.T) {
	t.Parallel()

	findings := []finding.Finding{
		{Confidence: finding.ConfidenceHigh},
		{Confidence: finding.ConfidenceMedium},
		{Confidence: finding.ConfidenceLow},
	}

	tests := []struct {
		minConf   string
		wantCount int
	}{
		{"low", 3},
		{"medium", 2},
		{"high", 1},
	}

	for _, tt := range tests {
		result := filterByConfidence(findings, tt.minConf)
		if len(result) != tt.wantCount {
			t.Errorf(
				"filterByConfidence(%q): got %d, want %d",
				tt.minConf,
				len(result),
				tt.wantCount,
			)
		}
	}
}

func TestParseSeverity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  finding.Severity
	}{
		{"critical", finding.SeverityCritical},
		{"error", finding.SeverityError},
		{"warning", finding.SeverityWarning},
		{"info", finding.SeverityInfo},
		{"unknown", finding.SeverityInfo},
	}

	for _, tt := range tests {
		if got := parseSeverity(tt.input); got != tt.want {
			t.Errorf("parseSeverity(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestParseConfidence(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  finding.Confidence
	}{
		{"high", finding.ConfidenceHigh},
		{"medium", finding.ConfidenceMedium},
		{"low", finding.ConfidenceLow},
		{"unknown", finding.ConfidenceLow},
	}

	for _, tt := range tests {
		if got := parseConfidence(tt.input); got != tt.want {
			t.Errorf("parseConfidence(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestCollectFindingsDeduplicates(t *testing.T) {
	t.Parallel()

	f1 := finding.Finding{ID: finding.ID("test-id-1")}
	f2 := finding.Finding{ID: f1.ID}

	seen := make(map[finding.ID]bool)
	var unique []finding.Finding

	for _, f := range []finding.Finding{f1, f2} {
		if seen[f.ID] {
			continue
		}

		seen[f.ID] = true
		unique = append(unique, f)
	}

	if len(unique) != 1 {
		t.Errorf("expected 1 unique finding, got %d", len(unique))
	}
}

func TestOutputFindingsJSON(t *testing.T) {
	t.Parallel()

	f, err := finding.NewBuilder(
		"TEST", "cqrs-lint",
		"test message",
		finding.SeverityWarning,
		finding.Pos(finding.FilePath("test.go"), 1, 1),
	).
		WithCategory(finding.CategoryCorrectness).
		WithConfidence(finding.ConfidenceHigh).
		Build()
	if err != nil {
		t.Fatal(err)
	}

	cfg := &AppConfig{Format: "json"}
	err = outputFindings(context.Background(), []finding.Finding{f}, cfg)
	if err != nil {
		t.Errorf("outputFindings json: %v", err)
	}
}

func TestOutputFindingsEmpty(t *testing.T) {
	t.Parallel()

	cfg := &AppConfig{Format: "text", Quiet: true}
	err := outputFindings(context.Background(), nil, cfg)
	if err != nil {
		t.Errorf("outputFindings empty: %v", err)
	}
}

func TestVersionConstant(t *testing.T) {
	if version == "" {
		t.Error("version constant should not be empty")
	}
	if !strings.Contains(version, ".") {
		t.Error("version should contain a dot (semver)")
	}
}

func TestFilterSuppressed(t *testing.T) {
	t.Parallel()

	findings := []finding.Finding{
		{ID: finding.ID("active-1")},
		{ID: finding.ID("active-2")},
		{
			ID:          finding.ID("suppressed-1"),
			Suppression: &finding.Suppression{Kind: finding.SuppressionInSource},
		},
		{
			ID:          finding.ID("suppressed-2"),
			Suppression: &finding.Suppression{Kind: finding.SuppressionInSource},
		},
	}

	active, suppressedCount := filterSuppressed(findings)
	if len(active) != 2 {
		t.Errorf("active count: got %d, want 2", len(active))
	}
	if suppressedCount != 2 {
		t.Errorf("suppressed count: got %d, want 2", suppressedCount)
	}
	for _, f := range active {
		if f.Suppression != nil {
			t.Errorf("active finding %s should not be suppressed", f.ID)
		}
	}
}

func TestFilterSuppressed_AllActive(t *testing.T) {
	t.Parallel()

	findings := []finding.Finding{
		{ID: finding.ID("a")},
		{ID: finding.ID("b")},
	}

	active, suppressedCount := filterSuppressed(findings)
	if len(active) != 2 {
		t.Errorf("active count: got %d, want 2", len(active))
	}
	if suppressedCount != 0 {
		t.Errorf("suppressed count: got %d, want 0", suppressedCount)
	}
}

func TestFilterSuppressed_Empty(t *testing.T) {
	t.Parallel()

	active, suppressedCount := filterSuppressed(nil)
	if len(active) != 0 {
		t.Errorf("active count: got %d, want 0", len(active))
	}
	if suppressedCount != 0 {
		t.Errorf("suppressed count: got %d, want 0", suppressedCount)
	}
}

func TestFilterFPSuspects(t *testing.T) {
	t.Parallel()

	findings := []finding.Finding{
		{ID: finding.ID("high"), Confidence: finding.ConfidenceHigh},
		{ID: finding.ID("medium"), Confidence: finding.ConfidenceMedium},
		{ID: finding.ID("low1"), Confidence: finding.ConfidenceLow},
		{ID: finding.ID("low2"), Confidence: finding.ConfidenceLow},
		{ID: finding.ID("none"), Confidence: finding.ConfidenceNone},
	}

	suspects := filterFPSuspects(findings)
	if len(suspects) != 3 {
		t.Fatalf("suspects count: got %d, want 3 (low1, low2, none)", len(suspects))
	}

	seen := make(map[finding.ID]bool)
	for _, f := range suspects {
		seen[f.ID] = true
		if f.Confidence >= finding.ConfidenceMedium {
			t.Errorf("finding %s has confidence >= Medium, should be filtered out", f.ID)
		}
	}
	for _, id := range []finding.ID{"low1", "low2", "none"} {
		if !seen[id] {
			t.Errorf("finding %s should be in suspects", id)
		}
	}
}

func TestFilterFPSuspects_Empty(t *testing.T) {
	t.Parallel()

	suspects := filterFPSuspects(nil)
	if len(suspects) != 0 {
		t.Errorf("suspects count: got %d, want 0", len(suspects))
	}
}

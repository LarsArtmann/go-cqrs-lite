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

	duplicates := []finding.Finding{f1, f2}
	unique := deduplicate(duplicates)
	if len(unique) != 1 {
		t.Errorf("expected 1 unique finding, got %d", len(unique))
	}
}

func deduplicate(findings []finding.Finding) []finding.Finding {
	seen := make(map[finding.ID]bool)
	var unique []finding.Finding
	for _, f := range findings {
		if seen[f.ID] {
			continue
		}
		seen[f.ID] = true
		unique = append(unique, f)
	}

	return unique
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

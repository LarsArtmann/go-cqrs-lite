package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/larsartmann/go-finding"
)

func TestFormatFindingsText_Empty(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	formatFindingsText(&buf, nil, parseColorMode("never"))

	if buf.String() != "" {
		t.Errorf("expected empty output for no findings, got %q", buf.String())
	}
}

func TestFormatFindingsText_BasicFinding(t *testing.T) {
	t.Parallel()

	f, err := finding.NewBuilder(
		"C001", "cqrs-lint",
		"test message",
		finding.SeverityError,
		finding.Pos(finding.FilePath("example.go"), 10, 5),
	).
		WithCategory(finding.CategoryCorrectness).
		WithConfidence(finding.ConfidenceHigh).
		WithSuggestion("fix it").
		Build()
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	formatFindingsText(&buf, []finding.Finding{f}, parseColorMode("never"))

	output := buf.String()
	if !strings.Contains(output, "ERROR") {
		t.Error("expected ERROR in output")
	}

	if !strings.Contains(output, "example.go:10:5") {
		t.Error("expected file:line:col in output")
	}

	if !strings.Contains(output, "test message") {
		t.Error("expected message in output")
	}

	if !strings.Contains(output, "fix it") {
		t.Error("expected suggestion in output")
	}
}

func TestFormatFindingsText_WithSnippet(t *testing.T) {
	t.Parallel()

	f, err := finding.NewBuilder(
		"S001", "cqrs-lint",
		"hardcoded secret",
		finding.SeverityCritical,
		finding.Pos(finding.FilePath("config.go"), 5, 1),
	).
		WithSnippet(`secret := "sk-abc123"`).
		Build()
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	formatFindingsText(&buf, []finding.Finding{f}, parseColorMode("never"))

	output := buf.String()
	if !strings.Contains(output, "|>") {
		t.Error("expected snippet marker |> in output")
	}

	if !strings.Contains(output, "sk-abc123") {
		t.Error("expected snippet content in output")
	}
}

func TestParseColorMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"always", "always"},
		{"never", "never"},
		{"auto", "auto"},
		{"ALWAYS", "always"},
		{"", "auto"},
		{"invalid", "auto"},
	}

	for _, tt := range tests {
		got := parseColorMode(tt.input)
		if got.String() != tt.want && tt.want != "auto" {
			t.Errorf("parseColorMode(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestFilterByExcludedPaths(t *testing.T) {
	t.Parallel()

	findings := []finding.Finding{
		{Position: finding.Position{File: finding.FilePath("src/main.go")}},
		{Position: finding.Position{File: finding.FilePath("vendor/lib/db.go")}},
		{Position: finding.Position{File: finding.FilePath("src/handler.go")}},
		{Position: finding.Position{File: finding.FilePath("gen/generated.go")}},
	}

	filtered := filterByExcludedPaths(findings, []string{"vendor/", "gen"})

	if len(filtered) != 2 {
		t.Errorf("expected 2 findings after exclusion, got %d", len(filtered))
	}

	for _, f := range filtered {
		path := string(f.Position.File)
		if strings.Contains(path, "vendor/") || strings.Contains(path, "gen") {
			t.Errorf("finding from excluded path remained: %s", path)
		}
	}
}

func TestFilterByExcludedPaths_GlobPattern(t *testing.T) {
	t.Parallel()

	findings := []finding.Finding{
		{Position: finding.Position{File: finding.FilePath("src/main.go")}},
		{Position: finding.Position{File: finding.FilePath("src/generated_test.go")}},
	}

	filtered := filterByExcludedPaths(findings, []string{"*_test.go"})

	if len(filtered) != 1 {
		t.Errorf("expected 1 finding after glob exclusion, got %d", len(filtered))
	}

	if string(filtered[0].Position.File) != "src/main.go" {
		t.Errorf("expected main.go to remain, got %s", filtered[0].Position.File)
	}
}

func TestFilterByExcludedPaths_EmptyPatterns(t *testing.T) {
	t.Parallel()

	findings := []finding.Finding{
		{Position: finding.Position{File: finding.FilePath("a.go")}},
	}

	filtered := filterByExcludedPaths(findings, nil)
	if len(filtered) != 1 {
		t.Errorf("expected 1 finding with empty patterns, got %d", len(filtered))
	}

	filtered = filterByExcludedPaths(findings, []string{"", "  "})
	if len(filtered) != 1 {
		t.Errorf("expected 1 finding with blank patterns, got %d", len(filtered))
	}
}

func TestGroupFindingsByModule(t *testing.T) {
	t.Parallel()

	findings := []finding.Finding{
		{Position: finding.Position{File: finding.FilePath("services/api/main.go")}},
		{Position: finding.Position{File: finding.FilePath("services/api/handler.go")}},
		{Position: finding.Position{File: finding.FilePath("services/db/store.go")}},
		{Position: finding.Position{File: finding.FilePath("root.go")}},
	}

	groups := groupFindingsByModule(findings)

	if len(groups) != 3 {
		t.Errorf("expected 3 groups, got %d", len(groups))
	}

	// Check alphabetical order
	for i := 1; i < len(groups); i++ {
		if groups[i-1].name > groups[i].name {
			t.Error("groups should be sorted alphabetically")
		}
	}

	// Find the services/api group
	for _, g := range groups {
		if g.name == "services/api" {
			if len(g.findings) != 2 {
				t.Errorf("expected 2 findings in services/api, got %d", len(g.findings))
			}
		}
	}
}

func TestGroupFindingsByAggregate_GroupsByMetadata(t *testing.T) {
	t.Parallel()

	findings := []finding.Finding{
		{Rule: "C001", Metadata: map[string]string{"aggregate": "User"}},
		{Rule: "C002", Metadata: map[string]string{"aggregate": "User"}},
		{Rule: "C003", Metadata: map[string]string{"aggregate": "Order"}},
		{Rule: "C004"}, // Uncategorized
	}

	groups := groupFindingsByAggregate(findings)

	if len(groups) != 3 {
		t.Fatalf("expected 3 groups, got %d", len(groups))
	}

	// User has 2 findings → should be first (sorted by count desc).
	if groups[0].name != "User" || len(groups[0].findings) != 2 {
		t.Errorf("first group should be User (2 findings), got %s (%d)",
			groups[0].name, len(groups[0].findings))
	}

	// Order and Uncategorized both have 1 → alphabetical.
	var foundUncategorized, foundOrder bool
	for _, g := range groups {
		switch g.name {
		case "Order":
			foundOrder = true
		case "Uncategorized":
			foundUncategorized = true
		}
	}
	if !foundOrder {
		t.Error("Order group not found")
	}
	if !foundUncategorized {
		t.Error("Uncategorized group not found")
	}
}

func TestPrintFindingsByAggregate_RendersHeadersAndFindings(t *testing.T) {
	t.Parallel()

	findings := []finding.Finding{
		{
			Rule:     "C007",
			ToolName: "cqrs-lint",
			Message:  "time.Now in decider",
			Severity: finding.SeverityWarning,
			Position: finding.Position{File: finding.FilePath("user.go"), Line: 10, Column: 5},
			Metadata: map[string]string{"aggregate": "User"},
		},
		{
			Rule:     "C001",
			ToolName: "cqrs-lint",
			Message:  "missing event type",
			Severity: finding.SeverityError,
			Position: finding.Position{File: finding.FilePath("order.go"), Line: 20, Column: 1},
			Metadata: map[string]string{"aggregate": "Order"},
		},
	}

	var buf bytes.Buffer
	printFindingsByAggregate(&buf, findings, parseColorMode("never"))

	output := buf.String()

	if !strings.Contains(output, "--- User (1) ---") {
		t.Errorf("expected User group header, got: %s", output)
	}
	if !strings.Contains(output, "--- Order (1) ---") {
		t.Errorf("expected Order group header, got: %s", output)
	}
	if !strings.Contains(output, "time.Now in decider") {
		t.Error("expected User finding message in output")
	}
	if !strings.Contains(output, "missing event type") {
		t.Error("expected Order finding message in output")
	}
}

func TestPrintFindingsByAggregate_EmptyFindings(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	printFindingsByAggregate(&buf, nil, parseColorMode("never"))

	if buf.String() != "" {
		t.Errorf("expected empty output for no findings, got %q", buf.String())
	}
}

func TestPrintFindingsByAggregate_UntaggedFindingsGoToUncategorized(t *testing.T) {
	t.Parallel()

	findings := []finding.Finding{
		{
			Rule:     "C007",
			ToolName: "cqrs-lint",
			Message:  "no aggregate tag",
			Severity: finding.SeverityWarning,
			Position: finding.Position{File: finding.FilePath("main.go"), Line: 1, Column: 1},
		},
	}

	var buf bytes.Buffer
	printFindingsByAggregate(&buf, findings, parseColorMode("never"))

	output := buf.String()
	if !strings.Contains(output, "--- Uncategorized (1) ---") {
		t.Errorf("expected Uncategorized group header, got: %s", output)
	}
}

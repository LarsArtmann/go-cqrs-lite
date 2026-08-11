package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/larsartmann/go-finding"
	"github.com/larsartmann/go-output/delimited"
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

// sampleErrorFinding returns a single ERROR finding reused by the color-mode
// regression tests below.
func sampleErrorFinding(t *testing.T) finding.Finding {
	t.Helper()

	f, err := finding.NewBuilder(
		"C001", "cqrs-lint", "test message",
		finding.SeverityError,
		finding.Pos(finding.FilePath("example.go"), 10, 5),
	).Build()
	if err != nil {
		t.Fatal(err)
	}

	return f
}

const ansiEscape = "\x1b"

func hasANSI(s string) bool { return strings.Contains(s, ansiEscape) }

// TestFormatFindingsText_HonorsNoColor locks the NO_COLOR regression.
// formatFindingsText now delegates to cm.ShouldColor(), which honors NO_COLOR.
// The deleted hand-rolled shouldColor only checked os.ModeCharDevice and ignored
// NO_COLOR entirely, producing colored findings while go-output tables (which
// honored NO_COLOR) were colorless — an inconsistent single run.
func TestFormatFindingsText_HonorsNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	t.Setenv("FORCE_COLOR", "")

	var buf bytes.Buffer
	formatFindingsText(&buf, []finding.Finding{sampleErrorFinding(t)}, parseColorMode("auto"))

	if hasANSI(buf.String()) {
		t.Errorf("NO_COLOR=1 must suppress ANSI in findings text, got: %q", buf.String())
	}
}

// TestFormatFindingsText_HonorsCIEnv locks the CI regression: CI providers
// (GitHub Actions, GitLab CI, etc.) must get colorless findings text under
// ColorModeAuto, matching go-output table behavior.
func TestFormatFindingsText_HonorsCIEnv(t *testing.T) {
	t.Setenv("CI", "true")
	t.Setenv("FORCE_COLOR", "")

	var buf bytes.Buffer
	formatFindingsText(&buf, []finding.Finding{sampleErrorFinding(t)}, parseColorMode("auto"))

	if hasANSI(buf.String()) {
		t.Errorf("CI=true must suppress ANSI in findings text, got: %q", buf.String())
	}
}

// TestFormatFindingsText_HonorsForceColor locks the FORCE_COLOR gain: even when
// stdout is not a terminal (as in this test), FORCE_COLOR must produce colored
// findings text under ColorModeAuto. The old shouldColor checked ModeCharDevice
// on the writer and returned false for non-terminals, silently dropping color.
func TestFormatFindingsText_HonorsForceColor(t *testing.T) {
	t.Setenv("FORCE_COLOR", "1")
	t.Setenv("NO_COLOR", "")
	t.Setenv("CI", "")
	t.Setenv("TERM", "xterm")

	var buf bytes.Buffer
	formatFindingsText(&buf, []finding.Finding{sampleErrorFinding(t)}, parseColorMode("auto"))

	if !hasANSI(buf.String()) {
		t.Errorf("FORCE_COLOR=1 must produce ANSI in findings text, got plain: %q", buf.String())
	}
}

func TestFindingsToTable_CSV(t *testing.T) {
	t.Parallel()

	out, err := delimited.RenderCSV(findingsToTable([]finding.Finding{sampleErrorFinding(t)}))
	if err != nil {
		t.Fatalf("RenderCSV failed: %v", err)
	}

	for _, col := range []string{"Rule", "Severity", "File", "Line", "Column", "Message", "Suggestion", "Category", "Confidence"} {
		if !strings.Contains(out, col) {
			t.Errorf("CSV output missing column %q, got: %q", col, out)
		}
	}

	if !strings.Contains(out, "C001") {
		t.Error("expected rule ID C001 in CSV output")
	}

	if !strings.Contains(out, "example.go") {
		t.Error("expected file path example.go in CSV output")
	}
}

func TestFindingsToTable_TSV_UsesTabs(t *testing.T) {
	t.Parallel()

	out, err := delimited.RenderTSV(findingsToTable([]finding.Finding{sampleErrorFinding(t)}))
	if err != nil {
		t.Fatalf("RenderTSV failed: %v", err)
	}

	if !strings.Contains(out, "\t") {
		t.Errorf("TSV output must contain tab separators, got: %q", out)
	}
}

func TestFindingsToTable_EmptyFindings_OnlyHeader(t *testing.T) {
	t.Parallel()

	csvOut, err := delimited.RenderCSV(findingsToTable(nil))
	if err != nil {
		t.Fatalf("RenderCSV failed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(csvOut), "\n")
	if len(lines) != 1 {
		t.Errorf(
			"expected header-only CSV (1 line) for no findings, got %d lines: %q",
			len(lines),
			csvOut,
		)
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

func TestFilterByExcludedPaths_DoubleStarGlob(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		pattern  string
		path     string
		expected bool
	}{
		{"** matches nested templ file", "**/*_templ.go", "src/views/timeline_templ.go", true},
		{"** matches root-level templ file", "**/*_templ.go", "main_templ.go", true},
		{"** does not match non-templ file", "**/*_templ.go", "src/views/timeline.go", false},
		{"** in middle matches deep path", "src/**/generated.go", "src/a/b/c/generated.go", true},
		{"** in middle matches shallow path", "src/**/generated.go", "src/generated.go", true},
		{"vendor/** matches nested vendor file", "vendor/**", "vendor/lib/db.go", true},
		{"vendor/** does not match non-vendor", "vendor/**", "src/main.go", false},
		{"trailing ** matches everything under dir", "gen/**", "gen/a/b/c.go", true},
		{"** at end matches any depth", "**", "any/deep/path/file.go", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := matchExcludePattern(tc.pattern, tc.path)
			if got != tc.expected {
				t.Errorf("matchExcludePattern(%q, %q) = %v, want %v",
					tc.pattern, tc.path, got, tc.expected)
			}
		})
	}
}

func TestFilterByExcludedPaths_BackwardCompatSubstrings(t *testing.T) {
	t.Parallel()

	findings := []finding.Finding{
		{Position: finding.Position{File: finding.FilePath("vendor/lib/db.go")}},
		{Position: finding.Position{File: finding.FilePath("src/handler.go")}},
		{Position: finding.Position{File: finding.FilePath("gen/generated.go")}},
	}

	filtered := filterByExcludedPaths(findings, []string{"vendor/", "gen"})
	if len(filtered) != 1 {
		t.Errorf("expected 1 finding after substring exclusion, got %d", len(filtered))
	}
	if string(filtered[0].Position.File) != "src/handler.go" {
		t.Errorf("expected handler.go to remain, got %s", filtered[0].Position.File)
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

func TestPrintMarkdownGrouped_AggregateMode(t *testing.T) {
	t.Parallel()

	findings := []finding.Finding{
		{
			Rule:     "C001",
			ToolName: "cqrs-lint",
			Message:  "user issue",
			Severity: finding.SeverityError,
			Position: finding.Position{File: finding.FilePath("user.go"), Line: 1, Column: 1},
			Metadata: map[string]string{"aggregate": "User"},
		},
		{
			Rule:     "C002",
			ToolName: "cqrs-lint",
			Message:  "order issue",
			Severity: finding.SeverityWarning,
			Position: finding.Position{File: finding.FilePath("order.go"), Line: 2, Column: 3},
			Metadata: map[string]string{"aggregate": "Order"},
		},
		{
			Rule:     "C003",
			ToolName: "cqrs-lint",
			Message:  "another user issue",
			Severity: finding.SeverityWarning,
			Position: finding.Position{File: finding.FilePath("user2.go"), Line: 5, Column: 1},
			Metadata: map[string]string{"aggregate": "User"},
		},
	}

	var buf bytes.Buffer
	printMarkdownGrouped(&buf, findings, "aggregate")

	out := buf.String()
	// User group should appear first (2 findings > 1 finding).
	userIdx := strings.Index(out, "## User (2)")
	orderIdx := strings.Index(out, "## Order (1)")
	if userIdx < 0 {
		t.Errorf("expected '## User (2)' header in markdown output, got:\n%s", out)
	}
	if orderIdx < 0 {
		t.Errorf("expected '## Order (1)' header in markdown output, got:\n%s", out)
	}
	if userIdx > orderIdx && userIdx >= 0 && orderIdx >= 0 {
		t.Error("User group (more findings) should appear before Order group")
	}
}

func TestPrintMarkdownGrouped_ModuleMode(t *testing.T) {
	t.Parallel()

	findings := []finding.Finding{
		{
			Rule:     "C001",
			ToolName: "cqrs-lint",
			Message:  "issue in pkg/a",
			Severity: finding.SeverityError,
			Position: finding.Position{File: finding.FilePath("pkg/a/file.go"), Line: 1, Column: 1},
		},
		{
			Rule:     "C002",
			ToolName: "cqrs-lint",
			Message:  "issue in pkg/b",
			Severity: finding.SeverityWarning,
			Position: finding.Position{File: finding.FilePath("pkg/b/file.go"), Line: 2, Column: 3},
		},
	}

	var buf bytes.Buffer
	printMarkdownGrouped(&buf, findings, "module")

	out := buf.String()
	if !strings.Contains(out, "## pkg/a (1)") {
		t.Errorf("expected '## pkg/a (1)' header, got:\n%s", out)
	}
	if !strings.Contains(out, "## pkg/b (1)") {
		t.Errorf("expected '## pkg/b (1)' header, got:\n%s", out)
	}
}

func TestPrintMarkdownGrouped_DefaultMode(t *testing.T) {
	t.Parallel()

	findings := []finding.Finding{
		{
			Rule:     "C001",
			ToolName: "cqrs-lint",
			Message:  "test",
			Severity: finding.SeverityError,
			Position: finding.Position{File: finding.FilePath("main.go"), Line: 1, Column: 1},
		},
	}

	var buf bytes.Buffer
	printMarkdownGrouped(&buf, findings, "none")

	out := buf.String()
	if !strings.Contains(out, "## Findings (1)") {
		t.Errorf("expected '## Findings (1)' header for default mode, got:\n%s", out)
	}
}

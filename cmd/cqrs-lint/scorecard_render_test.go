package main

import (
	"encoding/json/v2"
	"strings"
	"testing"

	output "github.com/larsartmann/go-output"
)

func makeTestScorecard() ScorecardResult {
	return ScorecardResult{
		Summary: ScorecardSummary{
			UsedCount:       5,
			RelevantTotal:   10,
			IrrelevantCount: 2,
			CoveragePercent: 50,
			Grade:           "Fair",
		},
		Used: []ScorecardModule{
			{Key: "otel", DisplayName: "OpenTelemetry", Category: "Observability", Status: "used"},
			{
				Key:         "encryption",
				DisplayName: "Event Encryption",
				Category:    "Security",
				Status:      "used",
			},
		},
		Missing: []ScorecardModule{
			{
				Key:         "signing",
				DisplayName: "Event Signing",
				Category:    "Security",
				Status:      "missing",
				Suggestion:  "Tamper-proof event streams with HMAC or Ed25519 signing",
			},
			{
				Key:         "scheduling",
				DisplayName: "Scheduling",
				Category:    "Workflow",
				Status:      "missing",
				Suggestion:  "Durable deadline timers for time-based business rules",
			},
		},
		Irrelevant: []ScorecardModule{
			{
				Key:         "transport/http",
				DisplayName: "HTTP Transport",
				Category:    "Messaging",
				Status:      "n/a",
			},
		},
		Recommendations: []string{
			"Tamper-proof event streams with HMAC or Ed25519 signing",
			"Durable deadline timers for time-based business rules",
		},
	}
}

func TestRenderText_HasSummary(t *testing.T) {
	t.Parallel()

	result := makeTestScorecard()
	out := renderScorecardText(result, output.ColorModeNever)

	if !strings.Contains(out, "5/10") {
		t.Errorf("expected '5/10' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "50%") {
		t.Errorf("expected '50%%' in output")
	}
	if !strings.Contains(out, "Fair") {
		t.Errorf("expected 'Fair' grade in output")
	}
}

func TestRenderText_HasTables(t *testing.T) {
	t.Parallel()

	result := makeTestScorecard()
	out := renderScorecardText(result, output.ColorModeNever)

	if !strings.Contains(out, "USED") {
		t.Errorf("expected 'USED' section header")
	}
	if !strings.Contains(out, "MISSING") {
		t.Errorf("expected 'MISSING' section header")
	}
	if !strings.Contains(out, "OpenTelemetry") {
		t.Errorf("expected 'OpenTelemetry' in Used table")
	}
	if !strings.Contains(out, "Event Signing") {
		t.Errorf("expected 'Event Signing' in Missing table")
	}
}

func TestRenderText_HasRecommendations(t *testing.T) {
	t.Parallel()

	result := makeTestScorecard()
	out := renderScorecardText(result, output.ColorModeNever)

	if !strings.Contains(out, "RECOMMENDATIONS") {
		t.Errorf("expected 'RECOMMENDATIONS' section")
	}
	if !strings.Contains(out, "Tamper-proof") {
		t.Errorf("expected recommendation text in output")
	}
}

func TestRenderText_NoMissing_AllUsed(t *testing.T) {
	t.Parallel()

	result := ScorecardResult{
		Summary: ScorecardSummary{
			UsedCount:       10,
			RelevantTotal:   10,
			CoveragePercent: 100,
			Grade:           "Excellent",
		},
		Used: []ScorecardModule{
			{Key: "otel", DisplayName: "OpenTelemetry", Category: "Observability", Status: "used"},
		},
	}
	out := renderScorecardText(result, output.ColorModeNever)

	if strings.Contains(out, "MISSING") {
		t.Error("should not have MISSING section when nothing is missing")
	}
	if strings.Contains(out, "RECOMMENDATIONS") {
		t.Error("should not have RECOMMENDATIONS when nothing is missing")
	}
}

func TestRenderJSON_Valid(t *testing.T) {
	t.Parallel()

	result := makeTestScorecard()
	out, err := renderScorecardJSON(result)
	if err != nil {
		t.Fatalf("renderScorecardJSON error: %v", err)
	}

	// Must be valid JSON.
	var parsed ScorecardResult
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}

	// Must have expected top-level values.
	if parsed.Summary.UsedCount != 5 {
		t.Errorf("expected UsedCount 5, got %d", parsed.Summary.UsedCount)
	}
	if parsed.Summary.RelevantTotal != 10 {
		t.Errorf("expected RelevantTotal 10, got %d", parsed.Summary.RelevantTotal)
	}
	if parsed.Summary.CoveragePercent != 50 {
		t.Errorf("expected CoveragePercent 50, got %d", parsed.Summary.CoveragePercent)
	}
	if len(parsed.Used) != 2 {
		t.Errorf("expected 2 used modules, got %d", len(parsed.Used))
	}
	if len(parsed.Missing) != 2 {
		t.Errorf("expected 2 missing modules, got %d", len(parsed.Missing))
	}
}

func TestRenderScorecard_DispatchByFormat(t *testing.T) {
	t.Parallel()

	result := makeTestScorecard()

	// Text format.
	textOut, err := renderScorecard(result, "text", output.ColorModeNever)
	if err != nil {
		t.Fatalf("text render error: %v", err)
	}
	if !strings.Contains(textOut, "USED") {
		t.Error("text format should contain table headers")
	}

	// JSON format.
	jsonOut, err := renderScorecard(result, "json", output.ColorModeNever)
	if err != nil {
		t.Fatalf("json render error: %v", err)
	}
	if !strings.HasPrefix(jsonOut, "{") {
		t.Error("json format should start with '{'")
	}
}

func TestRenderText_ColorModeNever(t *testing.T) {
	t.Parallel()

	result := makeTestScorecard()
	out := renderScorecardText(result, output.ColorModeNever)

	// Should not contain ANSI escape codes.
	if strings.Contains(out, "\x1b[") {
		t.Error("ColorModeNever should not produce ANSI codes")
	}
}

func TestRenderMarkdown_HasSummary(t *testing.T) {
	t.Parallel()

	result := makeTestScorecard()
	out := renderScorecardMarkdown(result)

	if !strings.Contains(out, "5/10") {
		t.Errorf("expected '5/10' in markdown output, got:\n%s", out)
	}
	if !strings.Contains(out, "50%") {
		t.Errorf("expected '50%%' in markdown output")
	}
	if !strings.Contains(out, "Fair") {
		t.Errorf("expected 'Fair' grade in markdown output")
	}
}

func TestRenderMarkdown_HasTables(t *testing.T) {
	t.Parallel()

	result := makeTestScorecard()
	out := renderScorecardMarkdown(result)

	if !strings.Contains(out, "### Used") {
		t.Errorf("expected '### Used' section header")
	}
	if !strings.Contains(out, "### Missing") {
		t.Errorf("expected '### Missing' section header")
	}
	if !strings.Contains(out, "| Module | Category |") {
		t.Errorf("expected markdown table header row")
	}
	if !strings.Contains(out, "|--------|") {
		t.Errorf("expected markdown table separator row")
	}
	if !strings.Contains(out, "OpenTelemetry") {
		t.Errorf("expected 'OpenTelemetry' in Used table")
	}
	if !strings.Contains(out, "Event Signing") {
		t.Errorf("expected 'Event Signing' in Missing table")
	}
}

func TestRenderMarkdown_HasRecommendations(t *testing.T) {
	t.Parallel()

	result := makeTestScorecard()
	out := renderScorecardMarkdown(result)

	if !strings.Contains(out, "### Recommendations") {
		t.Errorf("expected '### Recommendations' section")
	}
	if !strings.Contains(out, "- Tamper-proof") {
		t.Errorf("expected bullet-list recommendation text")
	}
}

func TestRenderMarkdown_NoMissing(t *testing.T) {
	t.Parallel()

	result := ScorecardResult{
		Summary: ScorecardSummary{
			UsedCount:       10,
			RelevantTotal:   10,
			CoveragePercent: 100,
			Grade:           "Excellent",
		},
		Used: []ScorecardModule{
			{Key: "otel", DisplayName: "OpenTelemetry", Category: "Observability", Status: "used"},
		},
	}
	out := renderScorecardMarkdown(result)

	if strings.Contains(out, "### Missing") {
		t.Error("should not have Missing section when nothing is missing")
	}
	if strings.Contains(out, "### Recommendations") {
		t.Error("should not have Recommendations when nothing is missing")
	}
}

func TestRenderScorecard_MarkdownDispatch(t *testing.T) {
	t.Parallel()

	result := makeTestScorecard()

	mdOut, err := renderScorecard(result, "markdown", output.ColorModeNever)
	if err != nil {
		t.Fatalf("markdown render error: %v", err)
	}
	if !strings.Contains(mdOut, "| Module |") {
		t.Error("markdown format should contain table header row")
	}

	mdAliasOut, err := renderScorecard(result, "md", output.ColorModeNever)
	if err != nil {
		t.Fatalf("md alias render error: %v", err)
	}
	if mdAliasOut != mdOut {
		t.Error("'md' should produce identical output to 'markdown'")
	}
}

func TestRenderSARIF_Valid(t *testing.T) {
	t.Parallel()

	result := makeTestScorecard()
	out, err := renderScorecardSARIF(result)
	if err != nil {
		t.Fatalf("renderScorecardSARIF error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("invalid SARIF JSON: %v\n%s", err, out)
	}

	if parsed["version"] != "2.1.0" {
		t.Errorf("expected SARIF version 2.1.0, got %v", parsed["version"])
	}
}

func TestRenderSARIF_HasSummary(t *testing.T) {
	t.Parallel()

	result := makeTestScorecard()
	out, err := renderScorecardSARIF(result)
	if err != nil {
		t.Fatalf("renderScorecardSARIF error: %v", err)
	}

	var parsed sarifReport
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("invalid SARIF JSON: %v", err)
	}

	if len(parsed.Runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(parsed.Runs))
	}

	props := parsed.Runs[0].Properties
	if props == nil {
		t.Fatal("expected run.properties to be non-nil")
	}

	if props["coveragePercent"] != float64(50) {
		t.Errorf("expected coveragePercent 50, got %v", props["coveragePercent"])
	}
	if props["grade"] != "Fair" {
		t.Errorf("expected grade Fair, got %v", props["grade"])
	}
	if props["usedCount"] != float64(5) {
		t.Errorf("expected usedCount 5, got %v", props["usedCount"])
	}
	if props["relevantTotal"] != float64(10) {
		t.Errorf("expected relevantTotal 10, got %v", props["relevantTotal"])
	}
}

func TestRenderSARIF_MissingModulesAsResults(t *testing.T) {
	t.Parallel()

	result := makeTestScorecard()
	out, err := renderScorecardSARIF(result)
	if err != nil {
		t.Fatalf("renderScorecardSARIF error: %v", err)
	}

	var parsed sarifReport
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("invalid SARIF JSON: %v", err)
	}

	if len(parsed.Runs[0].Results) != 2 {
		t.Fatalf("expected 2 results (missing modules), got %d", len(parsed.Runs[0].Results))
	}

	first := parsed.Runs[0].Results[0]
	if first.RuleID != "scorecard/missing-module" {
		t.Errorf("expected ruleId scorecard/missing-module, got %s", first.RuleID)
	}
	if first.Level != "info" {
		t.Errorf("expected level info, got %s", first.Level)
	}
	if !strings.Contains(first.Message.Text, "Event Signing") {
		t.Errorf("expected first result to mention 'Event Signing', got %s", first.Message.Text)
	}
	if len(first.Locations) != 1 || first.Locations[0].PhysicalLocation.ArtifactLocation.URI != "go.mod" {
		t.Errorf("expected location go.mod, got %+v", first.Locations)
	}
}

func TestRenderSARIF_NoMissingEmptyResults(t *testing.T) {
	t.Parallel()

	result := ScorecardResult{
		Summary: ScorecardSummary{
			UsedCount:       10,
			RelevantTotal:   10,
			CoveragePercent: 100,
			Grade:           "Excellent",
		},
		Used: []ScorecardModule{
			{Key: "otel", DisplayName: "OpenTelemetry", Category: "Observability", Status: "used"},
		},
	}
	out, err := renderScorecardSARIF(result)
	if err != nil {
		t.Fatalf("renderScorecardSARIF error: %v", err)
	}

	var parsed sarifReport
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("invalid SARIF JSON: %v", err)
	}

	if len(parsed.Runs[0].Results) != 0 {
		t.Errorf("expected 0 results when nothing missing, got %d", len(parsed.Runs[0].Results))
	}
}

func TestRenderScorecard_SARIFDispatch(t *testing.T) {
	t.Parallel()

	result := makeTestScorecard()

	sarifOut, err := renderScorecard(result, "sarif", output.ColorModeNever)
	if err != nil {
		t.Fatalf("sarif render error: %v", err)
	}
	if !strings.HasPrefix(sarifOut, "{") {
		t.Error("sarif format should start with '{'")
	}
}

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
	if len(first.Locations) != 1 ||
		first.Locations[0].PhysicalLocation.ArtifactLocation.URI != "go.mod" {
		t.Errorf("expected location go.mod, got %+v", first.Locations)
	}
}

func TestRenderSARIF_LogicalLocationsPopulated(t *testing.T) {
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

	lls := parsed.Runs[0].LogicalLocations
	// makeTestScorecard has 2 used + 2 missing = 4 modules.
	if len(lls) != 4 {
		t.Fatalf("expected 4 logicalLocations (2 used + 2 missing), got %d", len(lls))
	}

	// Every logicalLocation must have kind "module".
	byFQN := make(map[string]sarifLogicalLocation, len(lls))
	for _, ll := range lls {
		if ll.Kind != "module" {
			t.Errorf(
				"logicalLocation %q has kind %q, want \"module\"",
				ll.FullyQualifiedName,
				ll.Kind,
			)
		}
		byFQN[ll.FullyQualifiedName] = ll
	}

	// Verify all expected modules appear.
	for _, key := range []string{"otel", "encryption", "signing", "scheduling"} {
		if _, ok := byFQN[key]; !ok {
			t.Errorf("expected logicalLocation with fullyQualifiedName %q", key)
		}
	}

	// Verify index mapping: missing-module results must reference the correct
	// logicalLocation index for their module.
	for _, res := range parsed.Runs[0].Results {
		if len(res.Locations) == 0 || len(res.Locations[0].LogicalLocations) == 0 {
			t.Fatal("missing-module result has no logicalLocation reference")
		}
		idx := res.Locations[0].LogicalLocations[0].Index
		if idx < 0 || idx >= len(lls) {
			t.Fatalf("result logicalLocation index %d out of range [0, %d)", idx, len(lls))
		}
		referenced := lls[idx]
		if !strings.Contains(res.Message.Text, referenced.Name) {
			t.Errorf("result message %q does not reference module %q at index %d",
				res.Message.Text, referenced.Name, idx)
		}
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

func TestRenderSARIF_MetaengineProperties(t *testing.T) {
	t.Parallel()

	result := makeTestScorecard()
	result.Metaengine = &ScorecardMetaengine{
		Detected:        true,
		Engines:         []string{"sqlite", "pebble"},
		PushdownAdopted: true,
	}

	out, err := renderScorecardSARIF(result)
	if err != nil {
		t.Fatalf("renderScorecardSARIF error: %v", err)
	}

	var parsed sarifReport
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("invalid SARIF JSON: %v", err)
	}

	props := parsed.Runs[0].Properties
	if props["metaengineDetected"] != true {
		t.Errorf("expected metaengineDetected true, got %v", props["metaengineDetected"])
	}
	if props["metaenginePushdownAdopted"] != true {
		t.Errorf(
			"expected metaenginePushdownAdopted true, got %v",
			props["metaenginePushdownAdopted"],
		)
	}
	engines, ok := props["metaengineEngines"].([]any)
	if !ok {
		t.Fatalf("expected metaengineEngines to be a slice, got %T", props["metaengineEngines"])
	}
	if len(engines) != 2 {
		t.Fatalf("expected 2 engines, got %d", len(engines))
	}
}

func TestRenderSARIF_NoMetaengineProperties(t *testing.T) {
	t.Parallel()

	result := makeTestScorecard()
	// Metaengine is nil — SARIF properties should NOT contain metaengine keys.

	out, err := renderScorecardSARIF(result)
	if err != nil {
		t.Fatalf("renderScorecardSARIF error: %v", err)
	}

	var parsed sarifReport
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("invalid SARIF JSON: %v", err)
	}

	props := parsed.Runs[0].Properties
	if _, exists := props["metaengineDetected"]; exists {
		t.Error("should not have metaengineDetected when Metaengine is nil")
	}
}

func TestScorecard_CrossFormat_MetaengineConsistency(t *testing.T) {
	t.Parallel()

	result := makeTestScorecard()
	result.Metaengine = &ScorecardMetaengine{
		Detected:        true,
		Engines:         []string{"sqlite", "pebble"},
		PushdownAdopted: true,
		Suggestion:      "add FilterOnField for query pushdown",
	}

	formats := []struct {
		name   string
		format string
	}{
		{"text", "text"},
		{"json", "json"},
		{"markdown", "markdown"},
		{"sarif", "sarif"},
	}

	rendered := make(map[string]string, len(formats))

	for _, f := range formats {
		out, err := renderScorecard(result, f.format, output.ColorModeNever)
		if err != nil {
			t.Fatalf("%s render error: %v", f.name, err)
		}

		rendered[f.name] = out
	}

	t.Run("text_contains_metaengine", func(t *testing.T) {
		t.Parallel()

		s := rendered["text"]
		for _, want := range []string{"METAENGINE", "Detected: yes", "Pushdown: adopted", "sqlite", "pebble"} {
			if !strings.Contains(s, want) {
				t.Errorf("text output missing %q", want)
			}
		}
	})

	t.Run("json_contains_metaengine", func(t *testing.T) {
		t.Parallel()

		var parsed ScorecardResult

		if err := json.Unmarshal([]byte(rendered["json"]), &parsed); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}

		if parsed.Metaengine == nil {
			t.Fatal("expected non-nil metaengine in JSON")
		}

		if !parsed.Metaengine.Detected {
			t.Error("expected metaengine.detected true")
		}

		if !parsed.Metaengine.PushdownAdopted {
			t.Error("expected metaengine.pushdown_adopted true")
		}

		if len(parsed.Metaengine.Engines) != 2 {
			t.Fatalf("expected 2 engines, got %d", len(parsed.Metaengine.Engines))
		}
	})

	t.Run("markdown_contains_metaengine", func(t *testing.T) {
		t.Parallel()

		s := rendered["markdown"]
		for _, want := range []string{"Metaengine", "**Detected:** yes", "**Pushdown:** adopted", "sqlite", "pebble"} {
			if !strings.Contains(s, want) {
				t.Errorf("markdown output missing %q", want)
			}
		}
	})

	t.Run("sarif_contains_metaengine", func(t *testing.T) {
		t.Parallel()

		var parsed sarifReport

		if err := json.Unmarshal([]byte(rendered["sarif"]), &parsed); err != nil {
			t.Fatalf("invalid SARIF JSON: %v", err)
		}

		props := parsed.Runs[0].Properties
		if props["metaengineDetected"] != true {
			t.Errorf("expected metaengineDetected true, got %v", props["metaengineDetected"])
		}

		if props["metaenginePushdownAdopted"] != true {
			t.Errorf(
				"expected metaenginePushdownAdopted true, got %v",
				props["metaenginePushdownAdopted"],
			)
		}

		engines, ok := props["metaengineEngines"].([]any)
		if !ok {
			t.Fatalf("expected metaengineEngines to be a slice, got %T", props["metaengineEngines"])
		}

		if len(engines) != 2 {
			t.Fatalf("expected 2 engines, got %d", len(engines))
		}
	})
}

func TestScorecard_CrossFormat_NoMetaengineConsistency(t *testing.T) {
	t.Parallel()

	result := makeTestScorecard()
	// Metaengine is nil.

	for _, format := range []string{"text", "json", "markdown", "sarif"} {
		out, err := renderScorecard(result, format, output.ColorModeNever)
		if err != nil {
			t.Fatalf("%s render error: %v", format, err)
		}

		if strings.Contains(strings.ToLower(out), "metaengine") {
			t.Errorf("%s output should not mention metaengine when nil", format)
		}
	}
}

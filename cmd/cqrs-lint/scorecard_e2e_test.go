package main

import (
	"strings"
	"testing"

	output "github.com/larsartmann/go-output"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// TestScorecard_E2E_FromSource builds an AnalysisContext from Go source
// with known imports and verifies the full scorecard pipeline: detect →
// compute → render. This is the integration test for the scorecard feature.
func TestScorecard_E2E_FromSource(t *testing.T) {
	t.Parallel()

	// Source simulating a project that imports several go-cqrs-lite modules.
	src := map[string]string{
		"main.go": `package main

import (
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/otel/v4"
	"github.com/larsartmann/go-cqrs-lite/signing/v4"
	"github.com/larsartmann/go-cqrs-lite/scheduling/v4"
	"github.com/larsartmann/go-cqrs-lite/kv/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	"github.com/larsartmann/go-cqrs-lite/codec/v4"
)

func main() {}
`,
	}

	actx := analyzer.BuildContextFromSource(t, src)

	usage := analyzer.DetectUsedModules(actx.Packages, actx.GoFiles, analyzer.DefaultCatalog)

	result := ComputeScorecard(
		analyzer.DefaultCatalog, usage,
		actx.FeatureProfile, analyzer.PresetNone,
	)

	// Verify used modules: otel, signing, scheduling, kv, projectionhost, codec
	// (event, command, decider, id are Core — excluded from scoring).
	usedKeys := make(map[string]bool)
	for _, m := range result.Used {
		usedKeys[m.Key] = true
	}

	expectedUsed := []string{"otel", "signing", "scheduling", "kv", "projectionhost", "codec"}
	for _, k := range expectedUsed {
		if !usedKeys[k] {
			t.Errorf("expected module %s in Used, but it's missing. Used: %v", k, usedKeys)
		}
	}

	// Core modules should NOT appear in Used (they're not scored).
	if usedKeys["event"] {
		t.Error("core module 'event' should not appear in Used (it's excluded from scoring)")
	}

	// Verify summary math: UsedCount + len(Missing) + len(Irrelevant) == total scored.
	total := len(analyzer.DefaultCatalog.Scored())
	got := result.Summary.UsedCount + len(result.Missing) + len(result.Irrelevant)
	if got != total {
		t.Errorf("Used(%d) + Missing(%d) + Irrelevant(%d) = %d, want %d",
			result.Summary.UsedCount, len(result.Missing), len(result.Irrelevant), got, total)
	}

	// Verify coverage math.
	expected := len(expectedUsed) * 100 / result.Summary.RelevantTotal
	if result.Summary.CoveragePercent != expected {
		t.Errorf("expected %d%% coverage, got %d%%", expected, result.Summary.CoveragePercent)
	}

	// Render text and verify output shape.
	textOut := renderScorecardText(result, output.ColorModeNever)
	if !strings.Contains(textOut, "Adoption:") {
		t.Error("text output should contain 'Adoption:' summary line")
	}
	for _, k := range expectedUsed {
		entry, _ := analyzer.DefaultCatalog.Get(analyzer.ModuleKey(k))
		if !strings.Contains(textOut, entry.DisplayName) {
			t.Errorf("text output should contain '%s' in Used table", entry.DisplayName)
		}
	}

	// Render JSON and verify round-trip.
	jsonOut, err := renderScorecardJSON(result)
	if err != nil {
		t.Fatalf("JSON render error: %v", err)
	}
	if !strings.HasPrefix(strings.TrimSpace(jsonOut), "{") {
		t.Error("JSON output should start with '{'")
	}
}

// TestScorecard_E2E_LocalCLIProfile verifies that a local-CLI project
// (no server) gets transport/server modules excluded from its denominator.
func TestScorecard_E2E_LocalCLIProfile(t *testing.T) {
	t.Parallel()

	src := map[string]string{
		"main.go": `package main

import (
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/kv/v4"
	"github.com/larsartmann/go-cqrs-lite/codec/v4"
)

func main() {}
`,
	}

	actx := analyzer.BuildContextFromSource(t, src)

	usage := analyzer.DetectUsedModules(actx.Packages, actx.GoFiles, analyzer.DefaultCatalog)

	result := ComputeScorecard(
		analyzer.DefaultCatalog, usage,
		analyzer.FeatureProfile{HasServer: false, ServerLocal: false},
		analyzer.PresetLocalCLI,
	)

	// Transport and server-infra modules should be Irrelevant.
	irrelevantKeys := make(map[string]bool)
	for _, m := range result.Irrelevant {
		irrelevantKeys[m.Key] = true
	}
	for _, k := range []string{"transport/http", "transport/grpc", "prometheus", "watermill"} {
		if !irrelevantKeys[k] {
			t.Errorf("module %s should be Irrelevant for local-cli profile", k)
		}
	}

	// The denominator should NOT include irrelevant modules.
	// So: UsedCount + len(Missing) == RelevantTotal.
	if result.Summary.UsedCount+len(result.Missing) != result.Summary.RelevantTotal {
		t.Errorf("UsedCount(%d) + Missing(%d) != RelevantTotal(%d)",
			result.Summary.UsedCount, len(result.Missing), result.Summary.RelevantTotal)
	}

	// Denominator should be less than total scored (since some are irrelevant).
	if result.Summary.RelevantTotal >= len(analyzer.DefaultCatalog.Scored()) {
		t.Errorf("local-cli denominator (%d) should be < total scored (%d) — transport excluded",
			result.Summary.RelevantTotal, len(analyzer.DefaultCatalog.Scored()))
	}
}

// TestScorecard_E2E_ProductionProfile verifies that a production server
// project includes transport modules in its denominator.
func TestScorecard_E2E_ProductionProfile(t *testing.T) {
	t.Parallel()

	src := map[string]string{
		"main.go": `package main

import (
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/transport/http/v4"
	"github.com/larsartmann/go-cqrs-lite/otel/v4"
	"github.com/larsartmann/go-cqrs-lite/prometheus/v4"
)

func main() {}
`,
	}

	actx := analyzer.BuildContextFromSource(t, src)

	usage := analyzer.DetectUsedModules(actx.Packages, actx.GoFiles, analyzer.DefaultCatalog)

	result := ComputeScorecard(
		analyzer.DefaultCatalog, usage,
		analyzer.FeatureProfile{HasServer: true, ServerLocal: false},
		analyzer.PresetProduction,
	)

	// Transport and prometheus should be in Used, not Irrelevant.
	usedKeys := make(map[string]bool)
	for _, m := range result.Used {
		usedKeys[m.Key] = true
	}
	if !usedKeys["transport/http"] {
		t.Error("transport/http should be Used for production profile with HTTP import")
	}
	if !usedKeys["prometheus"] {
		t.Error("prometheus should be Used for production profile with prometheus import")
	}

	// No transport modules should be Irrelevant.
	irrelevantKeys := make(map[string]bool)
	for _, m := range result.Irrelevant {
		irrelevantKeys[m.Key] = true
	}
	if irrelevantKeys["transport/http"] {
		t.Error("transport/http should NOT be Irrelevant for production profile")
	}
	if irrelevantKeys["transport/grpc"] {
		t.Error("transport/grpc should NOT be Irrelevant for production profile")
	}
}

// TestScorecard_E2E_EmptyProject verifies the scorecard for a project with
// no go-cqrs-lite imports at all.
func TestScorecard_E2E_EmptyProject(t *testing.T) {
	t.Parallel()

	src := map[string]string{
		"main.go": `package main

import "fmt"

func main() { fmt.Println("hello") }
`,
	}

	actx := analyzer.BuildContextFromSource(t, src)

	usage := analyzer.DetectUsedModules(actx.Packages, actx.GoFiles, analyzer.DefaultCatalog)

	result := ComputeScorecard(
		analyzer.DefaultCatalog, usage,
		actx.FeatureProfile, analyzer.PresetNone,
	)

	if result.Summary.UsedCount != 0 {
		t.Errorf("expected 0 used modules for empty project, got %d", result.Summary.UsedCount)
	}
	if result.Summary.CoveragePercent != 0 {
		t.Errorf(
			"expected 0%% coverage for empty project, got %d%%",
			result.Summary.CoveragePercent,
		)
	}
	if len(result.Recommendations) == 0 {
		t.Error("empty project should have recommendations")
	}
}

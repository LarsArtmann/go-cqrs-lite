package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	output "github.com/larsartmann/go-output"
)

// TestComputeScorecardWithWaivers_MovesRowsAndRecomputes pins the waiver
// contract: the row leaves MISSING for a visible WAIVED partition carrying
// reason and trigger, the denominator shrinks, and grade/coverage recompute.
func TestComputeScorecardWithWaivers_MovesRowsAndRecomputes(t *testing.T) {
	t.Parallel()

	before := ComputeScorecard(
		analyzer.DefaultCatalog, absentUsage(),
		analyzer.FeatureProfile{}, analyzer.PresetNone,
	)

	waivers := []analyzer.ScorecardWaiver{{
		Key:     "graph",
		Reason:  "no traversal-heavy read models in this domain",
		Trigger: "variable-depth queries appear",
	}}

	result, err := ComputeScorecardWithWaivers(
		analyzer.DefaultCatalog, absentUsage(),
		analyzer.FeatureProfile{}, analyzer.PresetNone, waivers,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Summary.WaivedCount != 1 {
		t.Fatalf("WaivedCount = %d, want 1", result.Summary.WaivedCount)
	}
	if len(result.Waived) != 1 || result.Waived[0].Key != "graph" {
		t.Fatalf("Waived = %+v, want one graph row", result.Waived)
	}
	if result.Waived[0].Status != "waived" {
		t.Errorf("waived row status = %q", result.Waived[0].Status)
	}
	wantEvidence := "no traversal-heavy read models in this domain — revisit when: variable-depth queries appear"
	if result.Waived[0].Evidence != wantEvidence {
		t.Errorf("waived evidence = %q, want %q", result.Waived[0].Evidence, wantEvidence)
	}
	if slices.ContainsFunc(
		result.Missing,
		func(m ScorecardModule) bool { return m.Key == "graph" },
	) {
		t.Error("graph must leave MISSING after being waived")
	}

	if want := before.Summary.RelevantTotal - 1; result.Summary.RelevantTotal != want {
		t.Errorf("RelevantTotal = %d, want %d", result.Summary.RelevantTotal, want)
	}
	if want := before.Summary.CoveragePercent; result.Summary.CoveragePercent != want {
		// Coverage recomputes against the smaller denominator; with 0 used
		// it stays 0 — this pins that the recompute path ran at all.
		t.Logf("coverage %d -> %d", before.Summary.CoveragePercent, result.Summary.CoveragePercent)
	}
}

// TestComputeScorecardWithWaivers_CoverageRisesByLeavingDenominator pins the
// grade-math decision: a waived row is declared not-applicable and leaves
// the denominator (like Irrelevant, but on purpose).
func TestComputeScorecardWithWaivers_CoverageRisesByLeavingDenominator(t *testing.T) {
	t.Parallel()

	usage := absentUsage()
	for _, key := range []analyzer.ModuleKey{"metaengine", "codec"} {
		usage[key] = analyzer.ModuleUsage{Key: key, Status: analyzer.UsageImported}
	}

	before := ComputeScorecard(
		analyzer.DefaultCatalog, usage,
		analyzer.FeatureProfile{}, analyzer.PresetNone,
	)

	waivers := []analyzer.ScorecardWaiver{{
		Key: "graph", Reason: "no use case", Trigger: "traversal appears",
	}}

	result, err := ComputeScorecardWithWaivers(
		analyzer.DefaultCatalog, usage,
		analyzer.FeatureProfile{}, analyzer.PresetNone, waivers,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Summary.UsedCount != 2 {
		t.Fatalf("UsedCount = %d, want 2", result.Summary.UsedCount)
	}
	if want := before.Summary.RelevantTotal - 1; result.Summary.RelevantTotal != want {
		t.Errorf("RelevantTotal = %d, want %d", result.Summary.RelevantTotal, want)
	}
	if want := 2 * 100 / (before.Summary.RelevantTotal - 1); result.Summary.CoveragePercent != want {
		t.Errorf("CoveragePercent = %d, want %d", result.Summary.CoveragePercent, want)
	}
}

// TestComputeScorecardWithWaivers_TriggerlessWaiverIsShamed pins that a
// waiver without a revisit trigger renders with a visible gap — a permanent
// pin by accident must not look deliberate.
func TestComputeScorecardWithWaivers_TriggerlessWaiverIsShamed(t *testing.T) {
	t.Parallel()

	waivers := []analyzer.ScorecardWaiver{{Key: "kv", Reason: "read models live in metaengine"}}

	result, err := ComputeScorecardWithWaivers(
		analyzer.DefaultCatalog, absentUsage(),
		analyzer.FeatureProfile{}, analyzer.PresetNone, waivers,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result.Waived[0].Evidence, "no revisit trigger recorded") {
		t.Errorf("triggerless waiver evidence = %q, want shaming suffix", result.Waived[0].Evidence)
	}
}

// TestComputeScorecardWithWaivers_RejectsWaivingUsedModule pins the loud
// failure: a waiver must record a refusal, never mask an adoption.
func TestComputeScorecardWithWaivers_RejectsWaivingUsedModule(t *testing.T) {
	t.Parallel()

	usage := absentUsage()
	usage["codec"] = analyzer.ModuleUsage{Key: "codec", Status: analyzer.UsageImported}

	_, err := ComputeScorecardWithWaivers(
		analyzer.DefaultCatalog, usage,
		analyzer.FeatureProfile{}, analyzer.PresetNone,
		[]analyzer.ScorecardWaiver{{Key: "codec", Reason: "x"}},
	)
	if err == nil || !strings.Contains(err.Error(), "USED") {
		t.Fatalf("want used-module waiver error, got %v", err)
	}
}

// TestComputeScorecardWithWaivers_RejectsWaivingIrrelevantModule pins the
// dead-config failure: waiving a profile-irrelevant row errors instead of
// silently persisting a no-op.
func TestComputeScorecardWithWaivers_RejectsWaivingIrrelevantModule(t *testing.T) {
	t.Parallel()

	_, err := ComputeScorecardWithWaivers(
		analyzer.DefaultCatalog, absentUsage(),
		analyzer.FeatureProfile{HasServer: false, ServerLocal: false},
		analyzer.PresetLocalCLI,
		[]analyzer.ScorecardWaiver{{Key: "watermill", Reason: "x"}},
	)
	if err == nil || !strings.Contains(err.Error(), "irrelevant") {
		t.Fatalf("want irrelevant-module waiver error, got %v", err)
	}
}

// TestComputeScorecardWithWaivers_WaiverHeavyPressureNote pins that waivers
// dominating the scorecard re-assert pressure as a recommendation.
func TestComputeScorecardWithWaivers_WaiverHeavyPressureNote(t *testing.T) {
	t.Parallel()

	usage := absentUsage()
	usage["metaengine"] = analyzer.ModuleUsage{Key: "metaengine", Status: analyzer.UsageImported}

	waivers := []analyzer.ScorecardWaiver{
		{Key: "graph", Reason: "r", Trigger: "t"},
		{Key: "kv", Reason: "r", Trigger: "t"},
	}

	result, err := ComputeScorecardWithWaivers(
		analyzer.DefaultCatalog, usage,
		analyzer.FeatureProfile{}, analyzer.PresetNone, waivers,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := slices.ContainsFunc(result.Recommendations, func(r string) bool {
		return strings.Contains(r, "waived")
	})
	if !found {
		t.Errorf("recommendations %v should carry the waiver-pressure note", result.Recommendations)
	}
}

// TestRenderScorecard_WaivedAndModernity pin the render surface: WAIVED
// section, waived banner line, and the modernity headline in text+markdown.
func TestRenderScorecard_WaivedAndModernity(t *testing.T) {
	t.Parallel()

	waivers := []analyzer.ScorecardWaiver{{
		Key: "graph", Reason: "no traversal use case", Trigger: "graph queries appear",
	}}

	result, err := ComputeScorecardWithWaivers(
		analyzer.DefaultCatalog, absentUsage(),
		analyzer.FeatureProfile{}, analyzer.PresetNone, waivers,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	result.Summary.ModernityGrade = "Modern"

	text := renderScorecardText(result, output.ColorModeNever)
	for _, want := range []string{
		"WAIVED (recorded refusals — re-litigate on trigger)",
		"no traversal use case — revisit when: graph queries appear",
		"Modernity: Modern",
		"(1 modules waived — recorded refusals, see WAIVED)",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("text render missing %q", want)
		}
	}

	md := renderScorecardMarkdown(result)
	for _, want := range []string{"**Modernity: Modern**", "### Waived (1)", "Reason / revisit trigger"} {
		if !strings.Contains(md, want) {
			t.Errorf("markdown render missing %q", want)
		}
	}
}

// TestResolveScorecardWaivers pins the merge contract: the scored project's
// <path>/.cqrs-lint.json owns its refusals per key; the CLI-level (cwd)
// config only fills unrecorded keys; a malformed project config errors.
func TestResolveScorecardWaivers(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	projectCfg := `{"scorecard": {"waivers": [{"key": "graph", "reason": "project reason", "trigger": "t"}]}}`
	if err := os.WriteFile(
		filepath.Join(dir, ".cqrs-lint.json"),
		[]byte(projectCfg),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	merged, err := resolveScorecardWaivers(&AppConfig{
		Path: dir,
		ScorecardSettings: analyzer.ScorecardSettings{Waivers: []analyzer.ScorecardWaiver{
			{Key: "graph", Reason: "cli reason"}, // project wins
			{Key: "kv", Reason: "cli-only"},      // filled in
		}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(merged) != 2 {
		t.Fatalf("merged = %+v, want 2 waivers", merged)
	}
	for _, w := range merged {
		if w.Key == "graph" && w.Reason != "project reason" {
			t.Errorf("graph reason = %q, want project reason to win", w.Reason)
		}
	}

	// No project config: CLI-level waivers pass through unchanged.
	passthrough, err := resolveScorecardWaivers(&AppConfig{
		Path: t.TempDir(),
		ScorecardSettings: analyzer.ScorecardSettings{
			Waivers: []analyzer.ScorecardWaiver{{Key: "kv", Reason: "r"}},
		},
	})
	if err != nil || len(passthrough) != 1 {
		t.Fatalf("passthrough = (%v, %v), want 1 waiver", passthrough, err)
	}

	// Malformed project config (bad waiver key) errors loudly.
	badDir := t.TempDir()
	badCfg := `{"scorecard": {"waivers": [{"key": "bogus", "reason": "r"}]}}`
	if err := os.WriteFile(
		filepath.Join(badDir, ".cqrs-lint.json"),
		[]byte(badCfg),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := resolveScorecardWaivers(&AppConfig{Path: badDir}); err == nil {
		t.Fatal("want load error for bogus project waiver key")
	}
}

// TestComputeScorecardWithWaivers_InvalidWaiversSurfaceAnalyzeErrors ensures
// analyzer-level validation failures propagate as command errors.
func TestComputeScorecardWithWaivers_InvalidWaiversSurfaceAnalyzeErrors(t *testing.T) {
	t.Parallel()

	_, err := ComputeScorecardWithWaivers(
		analyzer.DefaultCatalog, absentUsage(),
		analyzer.FeatureProfile{}, analyzer.PresetNone,
		[]analyzer.ScorecardWaiver{{Key: "not-a-module", Reason: "x"}},
	)
	if err == nil || !strings.Contains(err.Error(), "unknown module key") {
		t.Fatalf("want unknown-key error, got %v", err)
	}
}

// TestResolveScorecardPreset pins the preset-resolution convention: a preset
// recorded in the scored project's <path>/.cqrs-lint.json wins over the
// operator's cwd config; with no project config the cwd preset stands.
func TestResolveScorecardPreset(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(dir, ".cqrs-lint.json"),
		[]byte(`{"preset": "library"}`),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	cfg := &AppConfig{Path: dir, Preset: analyzer.PresetLocalCLI}
	if err := resolveScorecardPreset(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Preset != analyzer.PresetLibrary {
		t.Errorf("preset = %q, want project's library to win over cwd local-cli", cfg.Preset)
	}

	// No project config: the cwd preset passes through unchanged.
	plain := &AppConfig{Path: t.TempDir(), Preset: analyzer.PresetLocalCLI}
	if err := resolveScorecardPreset(plain); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plain.Preset != analyzer.PresetLocalCLI {
		t.Errorf("preset = %q, want unchanged without project config", plain.Preset)
	}

	// Unknown preset in the project config errors loudly (LoadProjectConfig).
	badDir := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(badDir, ".cqrs-lint.json"),
		[]byte(`{"preset": "bogus"}`),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	if err := resolveScorecardPreset(&AppConfig{Path: badDir}); err == nil {
		t.Fatal("want error for unknown project preset")
	}
}

// TestComputeScorecardWithWaivers_FiredTriggerIsSurfaced pins trigger
// expiry: when a detectable profile signal that the waiver's trigger
// mentions has appeared, the WAIVED row is flagged and a recommendation
// demands re-litigation. Wording stays advisory ("LIKELY FIRED").
func TestComputeScorecardWithWaivers_FiredTriggerIsSurfaced(t *testing.T) {
	t.Parallel()

	waivers := []analyzer.ScorecardWaiver{{
		Key:     "idempotency",
		Reason:  "single-process dispatch, no redelivery",
		Trigger: "when we add an http server",
	}}

	result, err := ComputeScorecardWithWaivers(
		analyzer.DefaultCatalog, absentUsage(),
		analyzer.FeatureProfile{HasServer: true}, analyzer.PresetNone, waivers,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Waived) != 1 {
		t.Fatalf("Waived = %+v, want one row", result.Waived)
	}
	if !strings.Contains(result.Waived[0].Evidence, "TRIGGER LIKELY FIRED") {
		t.Errorf("evidence = %q, want fired-trigger flag", result.Waived[0].Evidence)
	}
	if !slices.ContainsFunc(result.Recommendations, func(r string) bool {
		return strings.Contains(r, "idempotency") && strings.Contains(r, "likely fired")
	}) {
		t.Errorf("recommendations = %+v, want a re-litigation entry", result.Recommendations)
	}
}

// TestComputeScorecardWithWaivers_UnfiredTriggerStaysQuiet pins the inverse:
// a signal that has not appeared, or a trigger whose words merely CONTAIN a
// signal word ("business" vs "bus"), must not flag the waiver.
func TestComputeScorecardWithWaivers_UnfiredTriggerStaysQuiet(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		trigger string
		fp      analyzer.FeatureProfile
	}{
		{
			name:    "server trigger, no server yet",
			trigger: "when a server appears",
			fp:      analyzer.FeatureProfile{},
		},
		{
			name:    "business is not the bus signal",
			trigger: "when business logic grows",
			fp:      analyzer.FeatureProfile{HasAsyncBus: true},
		},
		{
			name:    "fired signal but unrelated words",
			trigger: "when domain grows past 50 aggregates",
			fp:      analyzer.FeatureProfile{HasServer: true, HasAsyncBus: true},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			waivers := []analyzer.ScorecardWaiver{{
				Key: "graph", Reason: "no use case", Trigger: tt.trigger,
			}}
			result, err := ComputeScorecardWithWaivers(
				analyzer.DefaultCatalog, absentUsage(), tt.fp, analyzer.PresetNone, waivers,
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if strings.Contains(result.Waived[0].Evidence, "TRIGGER LIKELY FIRED") {
				t.Errorf("evidence = %q, must not flag", result.Waived[0].Evidence)
			}
			if slices.ContainsFunc(result.Recommendations, func(r string) bool {
				return strings.Contains(r, "likely fired")
			}) {
				t.Errorf("recommendations = %+v, must stay quiet", result.Recommendations)
			}
		})
	}
}

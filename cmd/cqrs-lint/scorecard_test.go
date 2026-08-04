package main

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

func TestComputeScorecard_AllUsed(t *testing.T) {
	t.Parallel()

	// Build a usage map where every scored module is imported.
	usage := make(map[analyzer.ModuleKey]analyzer.ModuleUsage)
	for _, e := range analyzer.DefaultCatalog.Scored() {
		usage[e.Key] = analyzer.ModuleUsage{
			Key:    e.Key,
			Status: analyzer.UsageImported,
		}
	}

	result := ComputeScorecard(
		analyzer.DefaultCatalog, usage,
		analyzer.FeatureProfile{}, analyzer.PresetNone,
	)

	if result.Summary.CoveragePercent != 100 {
		t.Errorf("expected 100%% coverage, got %d%%", result.Summary.CoveragePercent)
	}
	if len(result.Missing) != 0 {
		t.Errorf("expected 0 missing, got %d", len(result.Missing))
	}
	if len(result.Recommendations) != 0 {
		t.Errorf("expected 0 recommendations, got %d", len(result.Recommendations))
	}
	if result.Summary.Grade != "Excellent" {
		t.Errorf("expected Excellent grade, got %s", result.Summary.Grade)
	}
}

func TestComputeScorecard_NoneUsed(t *testing.T) {
	t.Parallel()

	// Empty usage map — all modules absent.
	usage := make(map[analyzer.ModuleKey]analyzer.ModuleUsage)
	for _, e := range analyzer.DefaultCatalog.Scored() {
		usage[e.Key] = analyzer.ModuleUsage{
			Key:    e.Key,
			Status: analyzer.UsageAbsent,
		}
	}

	result := ComputeScorecard(
		analyzer.DefaultCatalog, usage,
		analyzer.FeatureProfile{}, analyzer.PresetNone,
	)

	if result.Summary.CoveragePercent != 0 {
		t.Errorf("expected 0%% coverage, got %d%%", result.Summary.CoveragePercent)
	}
	if result.Summary.Grade != "Minimal" {
		t.Errorf("expected Minimal grade, got %s", result.Summary.Grade)
	}
	if len(result.Used) != 0 {
		t.Errorf("expected 0 used, got %d", len(result.Used))
	}
	// Missing should include all universally-relevant modules (no profile filter).
	if len(result.Missing) == 0 {
		t.Error("expected non-empty Missing list")
	}
	// Recommendations should have up to 3 entries.
	if len(result.Recommendations) > 3 {
		t.Errorf("expected ≤3 recommendations, got %d", len(result.Recommendations))
	}
	if len(result.Recommendations) == 0 {
		t.Error("expected at least 1 recommendation when modules are missing")
	}
}

func TestComputeScorecard_ProfileFilterExcludesTransport(t *testing.T) {
	t.Parallel()

	// local-cli profile: no server. Transport/server-infra modules should
	// be in Irrelevant, not in Missing.
	usage := make(map[analyzer.ModuleKey]analyzer.ModuleUsage)
	for _, e := range analyzer.DefaultCatalog.Scored() {
		usage[e.Key] = analyzer.ModuleUsage{
			Key:    e.Key,
			Status: analyzer.UsageAbsent,
		}
	}

	result := ComputeScorecard(
		analyzer.DefaultCatalog, usage,
		analyzer.FeatureProfile{HasServer: false, ServerLocal: false},
		analyzer.PresetLocalCLI,
	)

	// Transport and server-infra modules should be Irrelevant, NOT Missing.
	irrelevantKeys := make(map[string]bool)
	for _, m := range result.Irrelevant {
		irrelevantKeys[m.Key] = true
	}
	missingKeys := make(map[string]bool)
	for _, m := range result.Missing {
		missingKeys[m.Key] = true
	}

	serverOnlyModules := []string{
		"transport/http", "transport/grpc",
		"prometheus", "watermill",
		"stack/postgres", "stack/mysql", "stack/turso",
	}
	for _, k := range serverOnlyModules {
		if !irrelevantKeys[k] {
			t.Errorf("module %s should be Irrelevant for local-cli, but it isn't", k)
		}
		if missingKeys[k] {
			t.Errorf("module %s should NOT be in Missing for local-cli (it's irrelevant)", k)
		}
	}

	// Summary math: UsedCount + len(Missing) + len(Irrelevant) should equal total scored.
	total := len(analyzer.DefaultCatalog.Scored())
	got := result.Summary.UsedCount + len(result.Missing) + len(result.Irrelevant)
	if got != total {
		t.Errorf("Used(%d) + Missing(%d) + Irrelevant(%d) = %d, want %d (total scored)",
			result.Summary.UsedCount, len(result.Missing), len(result.Irrelevant), got, total)
	}
}

func TestComputeScorecard_RecommendationsSortedByCategory(t *testing.T) {
	t.Parallel()

	usage := make(map[analyzer.ModuleKey]analyzer.ModuleUsage)
	for _, e := range analyzer.DefaultCatalog.Scored() {
		usage[e.Key] = analyzer.ModuleUsage{
			Key:    e.Key,
			Status: analyzer.UsageAbsent,
		}
	}

	result := ComputeScorecard(
		analyzer.DefaultCatalog, usage,
		analyzer.FeatureProfile{}, analyzer.PresetNone,
	)

	// Recommendations should be sorted by category priority. Security should
	// come first (signing, encryption), then Reliability, etc.
	if len(result.Recommendations) == 0 {
		t.Fatal("expected recommendations")
	}

	// The first recommendation should be a Security module.
	missingCats := make([]string, 0, len(result.Missing))
	for _, m := range result.Missing {
		missingCats = append(missingCats, m.Category)
	}
	if len(missingCats) > 0 && missingCats[0] != "Security" {
		t.Errorf("expected first missing category to be Security, got %s", missingCats[0])
	}
}

func TestComputeScorecard_MixedUsage(t *testing.T) {
	t.Parallel()

	// Some used, some missing, some irrelevant.
	usage := make(map[analyzer.ModuleKey]analyzer.ModuleUsage)
	for _, e := range analyzer.DefaultCatalog.Scored() {
		status := analyzer.UsageAbsent
		// Mark a few as used.
		switch string(e.Key) {
		case "otel", "encryption", "signing", "scheduling", "kv", "codec":
			status = analyzer.UsageImported
		}
		usage[e.Key] = analyzer.ModuleUsage{Key: e.Key, Status: status}
	}

	result := ComputeScorecard(
		analyzer.DefaultCatalog, usage,
		analyzer.FeatureProfile{HasServer: false}, analyzer.PresetLocalCLI,
	)

	if result.Summary.UsedCount != 6 {
		t.Errorf("expected 6 used, got %d", result.Summary.UsedCount)
	}
	// Summary: used + missing = relevant total (denominator).
	if result.Summary.RelevantTotal != result.Summary.UsedCount+len(result.Missing) {
		t.Errorf("RelevantTotal (%d) != UsedCount (%d) + Missing (%d)",
			result.Summary.RelevantTotal, result.Summary.UsedCount, len(result.Missing))
	}
	// 6 used out of relevant total.
	expected := 6 * 100 / result.Summary.RelevantTotal
	if result.Summary.CoveragePercent != expected {
		t.Errorf("expected %d%%, got %d%%", expected, result.Summary.CoveragePercent)
	}
}

func TestScoreGrade(t *testing.T) {
	t.Parallel()

	tests := []struct {
		pct  int
		want string
	}{
		{100, "Excellent"},
		{80, "Excellent"},
		{79, "Good"},
		{60, "Good"},
		{59, "Fair"},
		{40, "Fair"},
		{39, "Sparse"},
		{20, "Sparse"},
		{19, "Minimal"},
		{0, "Minimal"},
	}

	for _, tt := range tests {
		if got := scoreGrade(tt.pct); got != tt.want {
			t.Errorf("scoreGrade(%d) = %q, want %q", tt.pct, got, tt.want)
		}
	}
}

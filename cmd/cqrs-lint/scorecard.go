package main

import (
	"context"
	"sort"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/adoption"
	lintversion "github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/version"
	"github.com/larsartmann/go-finding"
)

// ScorecardSummary is the headline math for the scorecard.
//
//nolint:tagliatelle // snake_case JSON for CLI tool consumers
type ScorecardSummary struct {
	UsedCount       int    `json:"used_count"`
	RelevantTotal   int    `json:"relevant_total"`
	IrrelevantCount int    `json:"irrelevant_count"`
	CoveragePercent int    `json:"coverage_percent"`
	Grade           string `json:"grade"`
	// WaivedCount is the number of MISSING rows moved to the Waived
	// partition by recorded waivers (.cqrs-lint.json scorecard.waivers).
	WaivedCount    int    `json:"waived_count,omitempty"`
	ModernityGrade string `json:"modernity_grade,omitempty"`
}

// ScorecardResult is the computed adoption scorecard. It partitions the
// catalog into Used / Missing / Irrelevant based on the detected usage and
// the project's feature profile.
type ScorecardResult struct {
	Summary         ScorecardSummary     `json:"summary"`
	Used            []ScorecardModule    `json:"used"`
	Missing         []ScorecardModule    `json:"missing"`
	Waived          []ScorecardModule    `json:"waived,omitempty"`
	Irrelevant      []ScorecardModule    `json:"irrelevant,omitempty"`
	Recommendations []string             `json:"recommendations,omitempty"`
	Metaengine      *ScorecardMetaengine `json:"metaengine,omitempty"`
	Deprecated      *ScorecardDeprecated `json:"deprecated,omitempty"`
}

// ScorecardMetaengine holds detected metaengine usage details — which engines
// are imported and whether SQL pushdown (FilterOnField/SortOnField) is adopted.
//
//nolint:tagliatelle // snake_case JSON for CLI tool consumers
type ScorecardMetaengine struct {
	Detected        bool     `json:"detected"`
	Engines         []string `json:"engines,omitempty"`
	PushdownAdopted bool     `json:"pushdown_adopted"`
	Suggestion      string   `json:"suggestion,omitempty"`
}

// ScorecardDeprecated reports usage of surfaces that v5 removes: V007
// (v5-removed-API references), F030 (deprecated transport/http SSE), and
// stack-surface imports (engine presets + stack.Bundle, deleted per
// ADR-0123). The panel answers "is this project already clean for the v5
// cut?" from the same detectors the lint run uses, independent of preset
// disables.
//
//nolint:tagliatelle // snake_case JSON for CLI tool consumers
type ScorecardDeprecated struct {
	RemovedAPIUses      int    `json:"removed_api_uses"`
	DeprecatedTransport int    `json:"deprecated_transport_uses"`
	StackPresetUses     int    `json:"stack_preset_uses"`
	Suggestion          string `json:"suggestion,omitempty"`
}

// ScorecardModule is one row in the scorecard output.
//
//nolint:tagliatelle // snake_case JSON for CLI tool consumers
type ScorecardModule struct {
	Key         string `json:"key"`
	DisplayName string `json:"display_name"`
	Category    string `json:"category"`
	Status      string `json:"status"`
	Evidence    string `json:"evidence,omitempty"`
	Suggestion  string `json:"suggestion,omitempty"`
}

// ComputeScorecard partitions the catalog into Used / Missing / Irrelevant
// based on the detected usage map and the project's feature profile. It
// computes the coverage percentage against the profile-relative denominator
// and generates up to 3 recommendations sorted by category priority.
func ComputeScorecard(
	catalog analyzer.Catalog,
	usage map[analyzer.ModuleKey]analyzer.ModuleUsage,
	fp analyzer.FeatureProfile,
	preset analyzer.ConfigPreset,
) ScorecardResult {
	relevant := catalog.RelevantFor(fp, preset)
	relevantSet := make(map[analyzer.ModuleKey]bool, len(relevant))
	for _, e := range relevant {
		relevantSet[e.Key] = true
	}

	// Composition credit: backends wired through metaengine engines or
	// system.New are in use even without direct stack imports — crediting
	// them keeps the remaining MISSING rows honest.
	usage = creditComposition(usage, fp, relevantSet)

	var result ScorecardResult

	// Partition into Used / Missing / Irrelevant.
	for _, e := range catalog.Scored() {
		row := ScorecardModule{
			Key:         string(e.Key),
			DisplayName: e.DisplayName,
			Category:    string(e.Category),
			Suggestion:  e.Suggestion,
		}

		u, isUsed := usage[e.Key]
		if isUsed && u.Status >= analyzer.UsageImported {
			row.Status = "used"
			row.Evidence = u.Evidence
			result.Used = append(result.Used, row)
		} else if relevantSet[e.Key] {
			row.Status = "missing"
			result.Missing = append(result.Missing, row)
		} else {
			row.Status = "n/a"
			result.Irrelevant = append(result.Irrelevant, row)
		}
	}

	// Sort Used by category priority then key.
	sort.SliceStable(result.Used, func(i, j int) bool {
		return scorecardLess(result.Used[i], result.Used[j])
	})
	// Sort Missing by category priority then key (recommendations are the first 3).
	sort.SliceStable(result.Missing, func(i, j int) bool {
		return scorecardLess(result.Missing[i], result.Missing[j])
	})
	sort.SliceStable(result.Irrelevant, func(i, j int) bool {
		return scorecardLess(result.Irrelevant[i], result.Irrelevant[j])
	})

	// Compute summary.
	usedCount := len(result.Used)
	relevantTotal := usedCount + len(result.Missing)
	result.Summary = ScorecardSummary{
		UsedCount:       usedCount,
		RelevantTotal:   relevantTotal,
		IrrelevantCount: len(result.Irrelevant),
	}
	if relevantTotal > 0 {
		result.Summary.CoveragePercent = usedCount * 100 / relevantTotal
	}
	result.Summary.Grade = scoreGrade(result.Summary.CoveragePercent)

	// Generate up to 3 recommendations from the Missing list.
	result.Recommendations = generateRecommendations(result.Missing, 3)

	// Metaengine detection section.
	if fp.HasMetaengine {
		me := &ScorecardMetaengine{
			Detected:        true,
			Engines:         fp.MetaengineEngines,
			PushdownAdopted: fp.MetaenginePushdown,
		}
		if !fp.MetaenginePushdown {
			me.Suggestion = "Adopt FilterOnField/SortOnField for SQL filter/sort " +
				"pushdown (50x faster at scale vs in-memory scan)"
		}
		result.Metaengine = me
	}

	return result
}

// ComputeDeprecatedPanel runs the two deprecated-surface detectors against
// the analysis context and reports their finding counts. It is computed
// separately from ComputeScorecard because it needs findings data, not just
// the module usage map. Zero counts mean "checked, clean" — the panel is
// the v5-readiness bill of health for the project.
func ComputeDeprecatedPanel(
	ctx context.Context,
	actx *analyzer.AnalysisContext,
) *ScorecardDeprecated {
	count := func(det finding.Detector) int {
		fs, err := det.Detect(ctx)
		if err != nil {
			return 0
		}

		return len(fs)
	}

	panel := &ScorecardDeprecated{
		RemovedAPIUses:      count(lintversion.NewV007Detector(actx)),
		DeprecatedTransport: count(adoption.NewF030Detector(actx)),
		StackPresetUses:     stackPresetUseCount(actx),
	}

	switch {
	case panel.RemovedAPIUses > 0:
		panel.Suggestion = "Migrate off APIs removed at v5 before the major " +
			"bump — each use becomes a compile error at the cut (RULES.md#v007)"
	case panel.DeprecatedTransport > 0:
		panel.Suggestion = "Replace transport/http SSE with go-sse or watermill " +
			"before the v5 cut"
	case panel.StackPresetUses > 0:
		panel.Suggestion = "Stack presets and stack.Bundle are removed at the " +
			"v5 cut (ADR-0123) — migrate to system.New composition"
	}

	return panel
}

// stackPresetUseCount returns the number of DISTINCT stack surfaces in use
// project-wide: the primary module's profile unioned with every per-module
// profile. The v007/F030 detector counts beside it scan the whole project,
// so a stack import hidden in a submodule (cqrs-htmx keeps its stack wiring
// in usermgmt/) must stay visible to the v5-readiness panel too — the
// primary-only view BuildContext exposes would silently zero it.
func stackPresetUseCount(actx *analyzer.AnalysisContext) int {
	seen := make(map[string]struct{}, len(actx.FeatureProfile.StackPresets))
	add := func(names []string) {
		for _, name := range names {
			seen[name] = struct{}{}
		}
	}
	add(actx.FeatureProfile.StackPresets)
	for _, fp := range actx.FeatureProfiles {
		add(fp.StackPresets)
	}
	return len(seen)
}

// compositionWideProfile returns the primary module's profile with the
// canonical-path composition signals — system.New wiring, metaengine and
// declarative pushdown — ORed across every per-module profile. Composition
// often lives in a dedicated submodule (cqrs-htmx/systemadapter,
// go-appkit/cqrs); the primary-only view BuildContext exposes would grade
// those projects as if the composition did not exist. Only the composition
// booleans are unioned — store resolution keeps its documented primary-wins
// semantics (feature_detect.go T20-3).
func compositionWideProfile(actx *analyzer.AnalysisContext) analyzer.FeatureProfile {
	fp := actx.FeatureProfile
	for _, sub := range actx.FeatureProfiles {
		if sub.HasSystemComposition {
			fp.HasSystemComposition = true
		}
		if sub.HasMetaengine {
			fp.HasMetaengine = true
		}
		if sub.MetaenginePushdown {
			fp.MetaenginePushdown = true
		}
	}
	return fp
}

// scorecardLess defines the sort order for scorecard rows: by category
// priority (lower = first), then by key for determinism.
func scorecardLess(a, b ScorecardModule) bool {
	priA := analyzer.CategoryPriority(analyzer.ModuleCategory(a.Category))
	priB := analyzer.CategoryPriority(analyzer.ModuleCategory(b.Category))
	if priA != priB {
		return priA < priB
	}
	return a.Key < b.Key
}

// generateRecommendations extracts up to N suggestions from the missing list.
// The list is already sorted by category priority.
func generateRecommendations(missing []ScorecardModule, n int) []string {
	if len(missing) == 0 {
		return nil
	}
	limit := min(n, len(missing))
	recs := make([]string, 0, limit)
	for i := range limit {
		if missing[i].Suggestion != "" {
			recs = append(recs, missing[i].Suggestion)
		}
	}
	return recs
}

// scoreGrade maps a coverage percentage to a qualitative grade.
func scoreGrade(pct int) string {
	switch {
	case pct >= 80:
		return "Excellent"
	case pct >= 60:
		return "Good"
	case pct >= 40:
		return "Fair"
	case pct >= 20:
		return "Sparse"
	default:
		return "Minimal"
	}
}

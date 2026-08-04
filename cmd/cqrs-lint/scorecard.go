package main

import (
	"sort"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// ScorecardSummary is the headline math for the scorecard.
type ScorecardSummary struct {
	UsedCount       int    `json:"used_count"`
	RelevantTotal   int    `json:"relevant_total"`
	IrrelevantCount int    `json:"irrelevant_count"`
	CoveragePercent int    `json:"coverage_percent"`
	Grade           string `json:"grade"`
}

// ScorecardResult is the computed adoption scorecard. It partitions the
// catalog into Used / Missing / Irrelevant based on the detected usage and
// the project's feature profile.
type ScorecardResult struct {
	Summary         ScorecardSummary  `json:"summary"`
	Used            []ScorecardModule `json:"used"`
	Missing         []ScorecardModule `json:"missing"`
	Irrelevant      []ScorecardModule `json:"irrelevant,omitempty"`
	Recommendations []string          `json:"recommendations,omitempty"`
}

// ScorecardModule is one row in the scorecard output.
type ScorecardModule struct {
	Key         string `json:"key"`
	DisplayName string `json:"display_name"`
	Category    string `json:"category"`
	Status      string `json:"status"`
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

	return result
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

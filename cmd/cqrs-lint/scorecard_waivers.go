package main

import (
	"fmt"
	"slices"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

// ComputeScorecardWithWaivers computes the scorecard and then applies the
// project's recorded waivers: rows matching a waiver move from MISSING to a
// visible WAIVED partition carrying the recorded reason and revisit trigger,
// and leave the coverage denominator — a waiver declares the module
// not-applicable, like the automatic Irrelevant partition but on purpose.
//
// Waiving is loud by design: waiving a USED row or a profile-irrelevant row
// is an error, reasons are mandatory, and a trigger-less waiver renders with
// a shaming suffix. The waiver list lives in the committed .cqrs-lint.json,
// so every reviewer sees the recorded refusals.
func ComputeScorecardWithWaivers(
	catalog analyzer.Catalog,
	usage map[analyzer.ModuleKey]analyzer.ModuleUsage,
	fp analyzer.FeatureProfile,
	preset analyzer.ConfigPreset,
	waivers []analyzer.ScorecardWaiver,
) (ScorecardResult, error) {
	if err := analyzer.ValidateScorecardWaivers(waivers); err != nil {
		return ScorecardResult{}, err
	}

	result := ComputeScorecard(catalog, usage, fp, preset)
	if err := applyWaivers(&result, waivers); err != nil {
		return ScorecardResult{}, err
	}

	return result, nil
}

// applyWaivers partitions MISSING into kept Missing and Waived, then
// recomputes the summary against the shrunken denominator.
func applyWaivers(result *ScorecardResult, waivers []analyzer.ScorecardWaiver) error {
	if err := checkWaivable(result, waivers); err != nil {
		return err
	}

	waivedBy := make(map[string]analyzer.ScorecardWaiver, len(waivers))
	for _, w := range waivers {
		waivedBy[w.Key] = w
	}

	kept := result.Missing[:0]
	for _, row := range result.Missing {
		w, ok := waivedBy[row.Key]
		if !ok {
			kept = append(kept, row)
			continue
		}
		row.Status = "waived"
		row.Evidence = waiverEvidence(w)
		result.Waived = append(result.Waived, row)
	}
	result.Missing = kept

	recomputeWaivedSummary(result)
	return nil
}

// checkWaivable refuses waivers that do not apply to a MISSING row: waiving
// a used module records a refusal of something the project already adopted,
// and waiving an irrelevant module is dead config under the current profile.
func checkWaivable(result *ScorecardResult, waivers []analyzer.ScorecardWaiver) error {
	for _, w := range waivers {
		if slices.ContainsFunc(result.Used, func(m ScorecardModule) bool { return m.Key == w.Key }) {
			return fmt.Errorf(
				"scorecard waiver for %q: module is USED — waivers record refusals, not redundancies",
				w.Key,
			)
		}
		if slices.ContainsFunc(result.Irrelevant, func(m ScorecardModule) bool { return m.Key == w.Key }) {
			return fmt.Errorf(
				"scorecard waiver for %q: module is irrelevant for this profile — remove the waiver",
				w.Key,
			)
		}
	}
	return nil
}

// recomputeWaivedSummary adjusts counts, coverage, grade, and recommendations
// for the waived rows leaving the denominator, and re-asserts pressure when
// waivers dominate the scorecard.
func recomputeWaivedSummary(result *ScorecardResult) {
	waived := len(result.Waived)
	result.Summary.WaivedCount = waived
	result.Summary.RelevantTotal -= waived
	if result.Summary.RelevantTotal > 0 {
		result.Summary.CoveragePercent = result.Summary.UsedCount * 100 / result.Summary.RelevantTotal
	}
	result.Summary.Grade = scoreGrade(result.Summary.CoveragePercent)

	result.Recommendations = generateRecommendations(result.Missing, 3)
	if waived > 0 && waived >= result.Summary.UsedCount {
		result.Recommendations = append(result.Recommendations, fmt.Sprintf(
			"%d modules are waived — review .cqrs-lint.json justifications and revisit triggers",
			waived,
		))
	}
}

// waiverEvidence renders the recorded justification plus its revisit
// trigger. A missing trigger is surfaced instead of hidden — a waiver
// without a trigger is a permanent pin by accident.
func waiverEvidence(w analyzer.ScorecardWaiver) string {
	if w.Trigger == "" {
		return w.Reason + " — no revisit trigger recorded"
	}
	return fmt.Sprintf("%s — revisit when: %s", w.Reason, w.Trigger)
}

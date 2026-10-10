package main

import (
	"fmt"
	"slices"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

// resolveScorecardWaivers merges the two waiver sources: the scored
// project's config (<path>/.cqrs-lint.json, the doctor/toolspec convention)
// wins per key, and the CLI-level config (cwd, cmdguard loader) fills keys
// the project did not record. The scored project owns its refusals.
func resolveScorecardWaivers(cfg *AppConfig) ([]analyzer.ScorecardWaiver, error) {
	project, found, err := analyzer.LoadProjectConfig(cfg.Path)
	if err != nil {
		return nil, err
	}
	if !found {
		return cfg.ScorecardSettings.Waivers, nil
	}

	merged := slices.Clone(project.Scorecard.Waivers)
	for _, w := range cfg.ScorecardSettings.Waivers {
		if !slices.ContainsFunc(
			merged,
			func(m analyzer.ScorecardWaiver) bool { return m.Key == w.Key },
		) {
			merged = append(merged, w)
		}
	}
	return merged, nil
}

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
	if err := applyWaivers(&result, waivers, fp); err != nil {
		return ScorecardResult{}, err
	}

	return result, nil
}

// applyWaivers partitions MISSING into kept Missing and Waived, then
// recomputes the summary against the shrunken denominator.
func applyWaivers(
	result *ScorecardResult,
	waivers []analyzer.ScorecardWaiver,
	fp analyzer.FeatureProfile,
) error {
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
		row.Evidence = waiverEvidence(w, fp)
		result.Waived = append(result.Waived, row)
	}
	result.Missing = kept

	recomputeWaivedSummary(result)
	flagFiredWaiverTriggers(result, waivers, fp)
	return nil
}

// checkWaivable refuses waivers that do not apply to a MISSING row: waiving
// a used module records a refusal of something the project already adopted,
// and waiving an irrelevant module is dead config under the current profile.
func checkWaivable(result *ScorecardResult, waivers []analyzer.ScorecardWaiver) error {
	for _, w := range waivers {
		if slices.ContainsFunc(
			result.Used,
			func(m ScorecardModule) bool { return m.Key == w.Key },
		) {
			return fmt.Errorf(
				"scorecard waiver for %q: module is USED — waivers record refusals, not redundancies",
				w.Key,
			)
		}
		if slices.ContainsFunc(
			result.Irrelevant,
			func(m ScorecardModule) bool { return m.Key == w.Key },
		) {
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
// without a trigger is a permanent pin by accident. When a detectable
// profile signal that the trigger mentions has since appeared, the waiver
// is flagged for re-litigation: a recorded refusal whose revisit condition
// plausibly arrived must not silently persist.
func waiverEvidence(w analyzer.ScorecardWaiver, fp analyzer.FeatureProfile) string {
	var b strings.Builder
	b.WriteString(w.Reason)
	if w.Trigger == "" {
		b.WriteString(" — no revisit trigger recorded")
	} else {
		fmt.Fprintf(&b, " — revisit when: %s", w.Trigger)
	}
	if fired := firedTriggerSignals(w, fp); len(fired) > 0 {
		fmt.Fprintf(&b, " — TRIGGER LIKELY FIRED (profile now has %s): re-litigate this waiver",
			strings.Join(fired, "; "))
	}
	return b.String()
}

// waiverTriggerSignals maps detectable FeatureProfile signals to the
// trigger words that make them reviewable. Matching is word-exact on the
// lowercased trigger text ("bus" does not match "business") — the wording
// stays advisory ("LIKELY FIRED") because triggers are free text the tool
// cannot fully interpret.
//
//nolint:gochecknoglobals // read-only signal table
var waiverTriggerSignals = []struct {
	signal string
	words  []string
	fired  func(analyzer.FeatureProfile) bool
}{
	{
		signal: "network server (server=true)",
		words:  []string{"server", "http", "endpoint"},
		fired:  func(fp analyzer.FeatureProfile) bool { return fp.HasServer },
	},
	{
		signal: "async bus (async-bus=true)",
		words:  []string{"async", "bus", "watermill", "broker", "distributed"},
		fired:  func(fp analyzer.FeatureProfile) bool { return fp.HasAsyncBus },
	},
	{
		signal: "external transport (transport=true)",
		words:  []string{"transport", "sse"},
		fired:  func(fp analyzer.FeatureProfile) bool { return fp.HasTransport },
	},
}

// firedTriggerSignals returns the detectable profile signals that are now
// PRESENT and whose trigger word the waiver's free-text revisit trigger
// mentions. Surfacing is advisory — never a silent waiver removal.
func firedTriggerSignals(w analyzer.ScorecardWaiver, fp analyzer.FeatureProfile) []string {
	if w.Trigger == "" {
		return nil
	}
	tokens := triggerTokens(w.Trigger)
	var fired []string
	for _, s := range waiverTriggerSignals {
		if !s.fired(fp) {
			continue
		}
		if slices.ContainsFunc(s.words, func(word string) bool { return tokens[word] }) {
			fired = append(fired, s.signal)
		}
	}
	return fired
}

// triggerTokens lowercases the free-text trigger and splits it into word
// tokens with surrounding punctuation stripped.
func triggerTokens(trigger string) map[string]bool {
	tokens := make(map[string]bool)
	for field := range strings.FieldsSeq(strings.ToLower(trigger)) {
		token := strings.Trim(field, ",.;:!?()[]\"'")
		if token != "" {
			tokens[token] = true
		}
	}
	return tokens
}

// flagFiredWaiverTriggers appends one recommendation per waiver whose
// revisit trigger plausibly fired, keeping the re-litigation pressure in
// the summary-level output (not just the WAIVED row evidence).
func flagFiredWaiverTriggers(
	result *ScorecardResult,
	waivers []analyzer.ScorecardWaiver,
	fp analyzer.FeatureProfile,
) {
	for _, w := range waivers {
		fired := firedTriggerSignals(w, fp)
		if len(fired) == 0 {
			continue
		}
		result.Recommendations = append(result.Recommendations, fmt.Sprintf(
			"waiver for %q: revisit trigger likely fired (%s) — re-litigate or re-record it",
			w.Key, strings.Join(fired, "; ")))
	}
}

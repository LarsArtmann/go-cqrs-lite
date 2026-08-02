package main

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/larsartmann/go-finding"
)

type breakdownEntry struct {
	Deduction int
	Severity  string
	Rule      string
}

// HealthScore computes a 0-100 score from findings.
type HealthScore struct {
	Score     int
	Grade     string
	Breakdown map[string]int
	// RawScore is the unclamped score (may be negative). When Score is 0
	// because deductions exceed 100, RawScore shows how far below zero the
	// project actually is — e.g., "0/100 (clamped from -43)" motivates the
	// user by showing that even a few fixes would move the needle.
	RawScore int
	// InfoCapped is true when the raw Info deductions exceeded the cap and were
	// truncated. Exposed for --verbose transparency.
	InfoCapped bool
	// InfoRawDeduction is the uncapped Info total (only meaningful when InfoCapped).
	InfoRawDeduction int
	// InfoCapApplied is the actual cap value used in the computation. Exposed so
	// renderHealthScore can display the correct cap (which may differ from the
	// default when the consumer set a custom value via HealthConfig.InfoCap).
	InfoCapApplied int
}

// defaultInfoDeductionCap caps the total health-score penalty attributable to
// Info-severity findings. Info is the noisiest tier (style nits, low-confidence
// heuristic matches) and without a cap a single chatty rule can dominate the
// score: 18 D002 info findings used to cost -18, more than a Critical
// correctness bug. Capping Info at 20% keeps heuristic noise from drowning real
// issues. See item f-15 in the DiscordSync feedback triage.
//
// The cap is tunable via .cqrs-lint.json -> "health": {"info-cap": N}.
// ComputeHealthScore uses this default; ComputeHealthScoreWithCap accepts an
// explicit value.
const defaultInfoDeductionCap = 20

// ComputeHealthScore calculates the health score using the default Info cap.
func ComputeHealthScore(findings []finding.Finding) HealthScore {
	return ComputeHealthScoreWithCap(findings, defaultInfoDeductionCap)
}

// ComputeHealthScoreWithCap calculates the health score with a caller-specified
// Info-deduction cap. Pass defaultInfoDeductionCap for the standard behavior, or
// a custom value from .cqrs-lint.json -> "health": {"info-cap": N}.
//
// Two fairness adjustments over a naive severity sum:
//   - Confidence weighting: each finding's deduction is scaled by how certain
//     the rule is. High/Full confidence pays the full deduction; Medium pays
//     75%; Low pays 50%. No confidence info (the zero value) is treated as full
//     weight so findings that don't declare a confidence keep their original
//     impact.
//   - Info cap: total Info-severity deductions are capped at infoCap so a flood
//     of style nits can't outweigh real bugs.
//
// The Breakdown map reports per-rule weighted deductions (rounded once at the
// end from the accumulated float64 totals, so the breakdown is consistent with
// the Score). When the Info cap applies, InfoCapped and InfoRawDeduction expose
// the uncapped total for verbose-mode transparency.
func ComputeHealthScoreWithCap(findings []finding.Finding, infoCap int) HealthScore {
	if infoCap < 0 {
		infoCap = defaultInfoDeductionCap
	}

	var totalCritical, totalError, totalWarning, totalInfo float64
	// Accumulate unrounded per-rule deductions to avoid per-finding rounding
	// drift (round-2 self-critique d-5).
	rawBreakdown := make(map[string]float64)

	for _, f := range findings {
		if f.Suppression != nil {
			continue
		}

		weight := confidenceWeight(f.Confidence)
		key := fmt.Sprintf("%s %s", f.Severity.String(), f.Rule)

		switch f.Severity {
		case finding.SeverityCritical:
			d := 10 * weight
			totalCritical += d
			rawBreakdown[key] += d
		case finding.SeverityError:
			d := 5 * weight
			totalError += d
			rawBreakdown[key] += d
		case finding.SeverityWarning:
			d := 2 * weight
			totalWarning += d
			rawBreakdown[key] += d
		case finding.SeverityInfo:
			d := weight
			totalInfo += d
			rawBreakdown[key] += d
		}
	}

	uncappedInfo := totalInfo
	if totalInfo > float64(infoCap) {
		totalInfo = float64(infoCap)
	}

	total := totalCritical + totalError + totalWarning + totalInfo
	rawScore := 100 - int(math.Round(total))
	score := max(rawScore, 0)

	// Round the breakdown once from accumulated floats for consistency.
	breakdown := make(map[string]int, len(rawBreakdown))
	for key, d := range rawBreakdown {
		breakdown[key] = int(math.Round(d))
	}

	grade := "Needs Improvement"

	switch {
	case score >= 90:
		grade = "Excellent"
	case score >= 75:
		grade = "Good"
	case score >= 50:
		grade = "Fair"
	}

	hs := HealthScore{
		Score:          score,
		RawScore:       rawScore,
		Grade:          grade,
		Breakdown:      breakdown,
		InfoCapApplied: infoCap,
	}
	if uncappedInfo > float64(infoCap) {
		hs.InfoCapped = true
		hs.InfoRawDeduction = int(math.Round(uncappedInfo))
	}

	return hs
}

// confidenceWeight maps a finding's confidence to a health-score multiplier.
// Uncertain findings cost less: Low confidence halves the deduction, Medium
// reduces it to 75%, High/Full pay full price. The zero value (no confidence
// information) is treated as full weight so findings that don't declare a
// confidence keep their pre-weighting impact.
func confidenceWeight(c finding.Confidence) float64 {
	switch {
	case c >= finding.ConfidenceHigh: // 0.75, 1.0
		return 1.0
	case c >= finding.ConfidenceMedium: // 0.5
		return 0.75
	case c >= finding.ConfidenceLow: // 0.25
		return 0.5
	default: // 0.0 — no confidence info
		return 1.0
	}
}

func (hs HealthScore) sortedBreakdown() []breakdownEntry {
	entries := make([]breakdownEntry, 0, len(hs.Breakdown))

	for key, ded := range hs.Breakdown {
		parts := strings.SplitN(key, " ", 2)

		sev := ""
		rule := ""
		if len(parts) > 0 {
			sev = parts[0]
		}

		if len(parts) > 1 {
			rule = parts[1]
		}

		entries = append(entries, breakdownEntry{
			Deduction: ded,
			Severity:  sev,
			Rule:      rule,
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Deduction != entries[j].Deduction {
			return entries[i].Deduction > entries[j].Deduction
		}

		return entries[i].Severity < entries[j].Severity
	})

	return entries
}

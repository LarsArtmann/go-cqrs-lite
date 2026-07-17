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
}

// maxInfoDeduction caps the total health-score penalty attributable to
// Info-severity findings. Info is the noisiest tier (style nits, low-confidence
// heuristic matches) and without a cap a single chatty rule can dominate the
// score: 18 D002 info findings used to cost -18, more than a Critical
// correctness bug. Capping Info at 20% keeps heuristic noise from drowning real
// issues. See item f-15 in the DiscordSync feedback triage.
const maxInfoDeduction = 20

// ComputeHealthScore calculates the health score from findings.
//
// Two fairness adjustments over a naive severity sum:
//   - Confidence weighting: each finding's deduction is scaled by how certain
//     the rule is. High/Full confidence pays the full deduction; Medium pays
//     75%; Low pays 50%. No confidence info (the zero value) is treated as full
//     weight so findings that don't declare a confidence keep their original
//     impact.
//   - Info cap: total Info-severity deductions are capped at
//     maxInfoDeduction so a flood of style nits can't outweigh real bugs.
//
// The Breakdown map reports per-rule weighted deductions (rounded) for display;
// when the Info cap applies, the displayed Info deductions may sum to more than
// the capped contribution reflected in Score. That's intentional — the
// breakdown shows "what's noisy", the score shows "the actual penalty".
func ComputeHealthScore(findings []finding.Finding) HealthScore {
	var totalCritical, totalError, totalWarning, totalInfo float64
	breakdown := make(map[string]int)

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
			breakdown[key] += int(math.Round(d))
		case finding.SeverityError:
			d := 5 * weight
			totalError += d
			breakdown[key] += int(math.Round(d))
		case finding.SeverityWarning:
			d := 2 * weight
			totalWarning += d
			breakdown[key] += int(math.Round(d))
		case finding.SeverityInfo:
			d := weight
			totalInfo += d
			breakdown[key] += int(math.Round(d))
		}
	}

	if totalInfo > maxInfoDeduction {
		totalInfo = maxInfoDeduction
	}

	total := totalCritical + totalError + totalWarning + totalInfo
	score := max(100-int(math.Round(total)), 0)

	grade := "Needs Improvement"

	switch {
	case score >= 90:
		grade = "Excellent"
	case score >= 75:
		grade = "Good"
	case score >= 50:
		grade = "Fair"
	}

	return HealthScore{
		Score:     score,
		Grade:     grade,
		Breakdown: breakdown,
	}
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

package main

import (
	"fmt"
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

// ComputeHealthScore calculates the health score from findings.
// Starts at 100 and deducts points based on severity.
func ComputeHealthScore(findings []finding.Finding) HealthScore {
	score := 100
	breakdown := make(map[string]int)

	for _, f := range findings {
		if f.Suppression != nil {
			continue
		}

		deduction := 0

		switch f.Severity {
		case finding.SeverityCritical:
			deduction = 10
		case finding.SeverityError:
			deduction = 5
		case finding.SeverityWarning:
			deduction = 2
		case finding.SeverityInfo:
			deduction = 1
		}

		score -= deduction
		key := fmt.Sprintf("%s %s", f.Severity.String(), f.Rule)
		breakdown[key] += deduction
	}

	if score < 0 {
		score = 0
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

	return HealthScore{
		Score:     score,
		Grade:     grade,
		Breakdown: breakdown,
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

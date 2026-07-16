package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/larsartmann/go-finding"
)

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

// FormatHealthScore formats the health score for display.
func FormatHealthScore(hs HealthScore) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Score: %d/100 (%s)\n\n", hs.Score, hs.Grade)

	if len(hs.Breakdown) > 0 {
		sb.WriteString("Breakdown:\n")

		keys := make([]string, 0, len(hs.Breakdown))
		for k := range hs.Breakdown {
			keys = append(keys, k)
		}

		sort.Strings(keys)

		for _, k := range keys {
			fmt.Fprintf(&sb, "  -%-4d %s\n", hs.Breakdown[k], k)
		}
	}

	return sb.String()
}

package main

import (
	"fmt"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/benchkit/v4"
)

// strictNoiseGate returns a non-empty failure message when any headline metric
// (the set scripts/benchmark-regression.sh gates by default — kept in sync
// via benchkit.HeadlineMetricNames) was NOISY across the repeat runs; empty
// means the run is decision-grade (or the gate does not apply).
//
// This is --strict's second half: phase coverage is enforced by
// benchkit.Config.Strict (ErrSkippedPhases), noise coverage here — a median
// whose headline metrics moved more than the threshold between runs must
// fail a CI consumer instead of being trusted.
func strictNoiseGate(strict bool, repeated *benchkit.RepeatedResult) string {
	if !strict || repeated == nil || repeated.Median == nil {
		return ""
	}

	headline := make(map[string]bool)
	for _, name := range benchkit.HeadlineMetricNames() {
		headline[name] = true
	}

	var noisy []string

	for _, v := range repeated.Median.MetricVariation {
		if headline[v.Name] && !v.Reliable {
			noisy = append(noisy, fmt.Sprintf("%s CoV=%.1f%%", v.Name, v.CoV*100))
		}
	}

	if len(noisy) == 0 {
		return ""
	}

	return fmt.Sprintf(
		"--strict: NOISY headline metric(s): %s — medians from %d runs are not decision-grade; re-run on a quieter machine or raise --repeat",
		strings.Join(noisy, ", "), len(repeated.Runs))
}

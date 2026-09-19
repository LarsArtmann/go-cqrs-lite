package benchkit

import (
	"fmt"
	"io"
	"sort"
)

// noisyVariations returns the metrics whose cross-run CoV exceeded
// VariationThreshold, worst (highest CoV) first.
func noisyVariations(variations []MetricVariation) []MetricVariation {
	var noisy []MetricVariation

	for _, v := range variations {
		if !v.Reliable {
			noisy = append(noisy, v)
		}
	}

	sort.SliceStable(noisy, func(i, j int) bool { return noisy[i].CoV > noisy[j].CoV })

	return noisy
}

// printMetricVariation prints per-metric cross-run dispersion for a repeated
// run. A reader comparing runs needs the exceptions, not the full table: the
// summary line states how many metrics stayed stable, and only the noisy ones
// are listed, worst first.
func printMetricVariation(w io.Writer, r *Result) {
	if len(r.MetricVariation) == 0 {
		return
	}

	noisy := noisyVariations(r.MetricVariation)

	fmt.Fprintf(w, "Variation: %d/%d metrics stable (CoV < %.0f%%)\n",
		len(r.MetricVariation)-len(noisy), len(r.MetricVariation), VariationThreshold*100)

	for _, v := range noisy {
		fmt.Fprintf(w, "  NOISY  %-30s CoV=%.1f%%  n=%d  mean=%.0f %s\n",
			v.Name, v.CoV*100, len(v.Samples), v.Mean, v.Unit)
	}

	fmt.Fprintln(w)
}

// PrintComparisonVariation writes the cross-run dispersion summary for a
// backend comparison: one line per backend that carries MetricVariation data
// (i.e. was run with repeats), stating how many metrics stayed stable and
// naming the noisy ones with their CoV. Backends compared from single runs
// are omitted — a single run has no dispersion to report, and pretending a
// comparison is stable because nobody measured dispersion is exactly the
// dishonesty this section exists to prevent.
func PrintComparisonVariation(w io.Writer, results map[string]*Result) {
	printed := false

	for _, name := range sortedKeys(results) {
		r := results[name]
		if r == nil || len(r.MetricVariation) == 0 {
			continue
		}

		if !printed {
			fmt.Fprintf(w, "Variation (CoV >= %.0f%% = noisy, not decision-grade):\n",
				VariationThreshold*100)

			printed = true
		}

		noisy := noisyVariations(r.MetricVariation)

		fmt.Fprintf(w, "  %-10s %d/%d stable",
			name, len(r.MetricVariation)-len(noisy), len(r.MetricVariation))

		for i, v := range noisy {
			if i == 0 {
				fmt.Fprint(w, "  noisy: ")
			} else {
				fmt.Fprint(w, ", ")
			}

			fmt.Fprintf(w, "%s (%.1f%%)", v.Name, v.CoV*100)
		}

		fmt.Fprintln(w)
	}

	if printed {
		fmt.Fprintln(w)
	}
}

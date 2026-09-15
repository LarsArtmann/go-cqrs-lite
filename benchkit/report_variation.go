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

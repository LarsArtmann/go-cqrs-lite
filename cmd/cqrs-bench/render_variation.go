package main

import (
	"fmt"

	"github.com/larsartmann/go-output"

	"github.com/larsartmann/go-cqrs-lite/benchkit/v4"
)

// addVariationRows extends the run summary table with cross-run dispersion for
// repeated runs: the throughput headline plus the every-metric reliability
// verdict. Noisy metric names live in the text report's Variation section;
// the table stays one line high.
func addVariationRows(t *output.Table, r *benchkit.Result) {
	if r.RepeatCount <= 1 {
		return
	}

	t.AddRow([]string{"Repeat",
		fmt.Sprintf("median of %d (CoV %.1f%%)", r.RepeatCount, r.RepeatCoV*100)})

	if len(r.MetricVariation) == 0 {
		return
	}

	noisy := len(benchkit.NoisyMetricNames(r.MetricVariation))

	verdict := fmt.Sprintf("all %d metrics stable", len(r.MetricVariation))
	if noisy > 0 {
		verdict = fmt.Sprintf("%d/%d metrics noisy (CoV >= %.0f%%), see text report",
			noisy, len(r.MetricVariation), benchkit.VariationThreshold*100)
	}

	t.AddRow([]string{"Variation", verdict})
}

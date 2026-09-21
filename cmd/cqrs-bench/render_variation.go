package main

import (
	"fmt"

	"github.com/larsartmann/go-output"

	"github.com/larsartmann/go-cqrs-lite/benchkit/v4"
)

// addVariationRows extends the run summary table with cross-run dispersion for
// repeated runs: the throughput headline, per-headline-metric CoV rows (these
// rows are what --format csv/tsv export as variation columns), and the
// every-metric reliability verdict. Noisy metric names live in the text
// report's Variation section; the table stays one line per metric.
func addVariationRows(t *output.Table, r *benchkit.Result) {
	if r.RepeatCount <= 1 {
		return
	}

	t.AddRow([]string{
		"Repeat",
		fmt.Sprintf("median of %d (CoV %.1f%%)", r.RepeatCount, r.RepeatCoV*100),
	})

	headline := make(map[string]bool)
	for _, name := range benchkit.HeadlineMetricNames() {
		headline[name] = true
	}

	for _, v := range r.MetricVariation {
		if !headline[v.Name] {
			continue
		}

		verdict := "stable"
		if !v.Reliable {
			verdict = "NOISY"
		}

		t.AddRow([]string{
			v.Name + " CoV",
			fmt.Sprintf("%.1f%% (%s)", v.CoV*100, verdict),
		})
	}

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

package benchkit

import (
	"encoding/json/v2"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// PrintComparison writes a side-by-side comparison table of multiple results.
// The table includes latency percentiles, GC pause, write amplification, and
// integrity — the metrics that actually drive backend selection decisions.
func PrintComparison(w io.Writer, results map[string]*Result) {
	names := sortedKeys(results)

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Backend Comparison")
	fmt.Fprintln(w, strings.Repeat("=", 140))

	header := fmt.Sprintf(
		"%-10s %10s %10s %10s %10s %10s %10s %6s %8s %8s %8s %6s %10s %10s %10s",
		"Backend",
		"WriteP50",
		"WriteP99",
		"LoadP50",
		"LoadP99",
		"ColdP50",
		"GCMaxPau",
		"TailR",
		"A/op",
		"WrtAmp",
		"CoV%",
		"Noisy",
		"RAM",
		"Heap",
		"Disk",
	)
	fmt.Fprintln(w, header)
	fmt.Fprintln(w, strings.Repeat("-", len(header)))

	for _, name := range names {
		r := results[name]
		printComparisonRow(w, name, r)
	}

	// Cross-run dispersion footer: a comparison built from repeated runs must
	// state which backends' numbers are decision-grade, not just the median.
	PrintComparisonVariation(w, results)

	hasIntegrity := false

	for _, name := range names {
		if r := results[name]; r != nil && r.IntegrityErrors > 0 {
			hasIntegrity = true

			break
		}
	}

	if hasIntegrity {
		fmt.Fprintln(w)

		for _, name := range names {
			if r := results[name]; r != nil && r.IntegrityErrors > 0 {
				fmt.Fprintf(w, "  ⚠ %s: %d integrity errors!\n", name, r.IntegrityErrors)
			}
		}
	}

	hasWarnings := false

	for _, name := range names {
		if r := results[name]; r != nil && len(r.Warnings) > 0 {
			hasWarnings = true

			break
		}
	}

	if hasWarnings {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Phase Coverage:")

		for _, name := range names {
			r := results[name]
			if r == nil {
				continue
			}

			skipped := len(r.SkippedPhases)
			if skipped > 0 {
				fmt.Fprintf(w, "  ⚠ %s: %d phase(s) skipped (%s)\n",
					name, skipped, strings.Join(r.SkippedPhases, ", "))
			}
		}

		fmt.Fprintln(w)
		fmt.Fprintln(w, "Warnings:")

		for _, name := range names {
			r := results[name]
			if r == nil || len(r.Warnings) == 0 {
				continue
			}

			for _, msg := range r.Warnings {
				fmt.Fprintf(w, "  ⚠ %s: %s\n", name, msg)
			}
		}
	}

	fmt.Fprintln(w)
}

func printComparisonRow(w io.Writer, name string, r *Result) {
	if r.Error != "" {
		fmt.Fprintf(w, "%-10s %s\n", name, "FAILED: "+Truncate(r.Error, 60))

		return
	}

	covStr := "-"
	if r.RepeatCoV > 0 {
		covStr = fmt.Sprintf("%.1f%%", r.RepeatCoV*100)
	}

	noisyStr := "-"
	if len(r.MetricVariation) > 0 {
		noisyStr = strconv.Itoa(r.NoisyMetricCount())
	}

	wrtAmpStr := "-"
	if r.Disk.WriteAmplification > 0 {
		wrtAmpStr = fmt.Sprintf("%.1fx", r.Disk.WriteAmplification)
	}

	gcStr := roundDuration(r.GCMaxPause).String()
	if r.GCMaxPause == 0 {
		gcStr = "-"
	}

	tailStr := "-"
	if r.TailRatio > 0 {
		tailStr = fmt.Sprintf("%.1fx", r.TailRatio)
	}

	allocStr := "-"
	if r.AllocsPerOp > 0 {
		allocStr = fmt.Sprintf("%.0f", r.AllocsPerOp)
	}

	fmt.Fprintf(
		w, "%-10s %10s %10s %10s %10s %10s %10s %6s %8s %8s %8s %6s %10s %10s %10s\n",
		name,
		roundDuration(r.WriteLatency.P50),
		roundDuration(r.WriteLatency.P99),
		roundDuration(r.LoadLatency.P50),
		roundDuration(r.LoadLatency.P99),
		roundDuration(r.ColdReadLatency.P50),
		gcStr,
		tailStr,
		allocStr,
		wrtAmpStr,
		covStr,
		noisyStr,
		formatBytes(r.Memory.Resident),
		formatBytes(r.Memory.After),
		formatBytes(uint64(r.Disk.DatabaseBytes)),
	)
}

// WriteComparisonJSON serializes all results as a JSON object.
func WriteComparisonJSON(w io.Writer, results map[string]*Result) error {
	return json.MarshalWrite(w, results, jsonOpts)
}

// PrintMarkdown writes a markdown comparison table with evidence-grade metrics.
func PrintMarkdown(w io.Writer, results map[string]*Result) {
	names := sortedKeys(results)

	fmt.Fprintln(
		w,
		"| Backend | Write P50 | Write P99 | Load P50 | Load P99 | Cold P50 | GC Max | Write Amp | CoV | Noisy | RAM | Heap | Disk | Integrity |",
	)
	fmt.Fprintln(
		w,
		"|---------|----------:|----------:|---------:|---------:|---------:|-------:|----------:|----:|------:|----:|-----:|-----:|:---------:|",
	)

	for _, name := range names {
		r := results[name]
		if r.Error != "" {
			fmt.Fprintf(w, "| %s | FAILED | | | | | | | | | | |", name)

			continue
		}

		cov := "-"
		if r.RepeatCoV > 0 {
			cov = fmt.Sprintf("%.1f%%", r.RepeatCoV*100)
		}

		noisy := "-"
		if len(r.MetricVariation) > 0 {
			noisy = strconv.Itoa(r.NoisyMetricCount())
		}

		wrtAmp := "-"
		if r.Disk.WriteAmplification > 0 {
			wrtAmp = fmt.Sprintf("%.1fx", r.Disk.WriteAmplification)
		}

		gcMax := roundDuration(r.GCMaxPause).String()
		if r.GCMaxPause == 0 {
			gcMax = "-"
		}

		integrity := "✓"
		if r.IntegrityErrors > 0 {
			integrity = fmt.Sprintf("⚠ %d", r.IntegrityErrors)
		}

		fmt.Fprintf(
			w, "| %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s |\n",
			name,
			roundDuration(r.WriteLatency.P50),
			roundDuration(r.WriteLatency.P99),
			roundDuration(r.LoadLatency.P50),
			roundDuration(r.LoadLatency.P99),
			roundDuration(r.ColdReadLatency.P50),
			gcMax,
			wrtAmp,
			cov,
			noisy,
			formatBytes(r.Memory.Resident),
			formatBytes(r.Memory.After),
			formatBytes(uint64(r.Disk.DatabaseBytes)),
			integrity,
		)
	}

	// Variation summary: the honest-comparison rule applied to markdown —
	// state which backends' medians rest on stable repeats and which metrics
	// moved too much to trust. Single-run backends stay silent here; the
	// missing row IS the caveat.
	printed := false

	for _, name := range names {
		r := results[name]
		if r == nil || len(r.MetricVariation) == 0 {
			continue
		}

		if !printed {
			fmt.Fprintf(w, "\n**Variation** (CoV >= %.0f%% = noisy, not decision-grade)\n",
				VariationThreshold*100)

			printed = true
		}

		noisyVars := noisyVariations(r.MetricVariation)

		fmt.Fprintf(w, "\n- %s: %d/%d metrics stable", name,
			len(r.MetricVariation)-len(noisyVars), len(r.MetricVariation))

		for _, v := range noisyVars {
			fmt.Fprintf(w, ", `%s` CoV %.1f%%", v.Name, v.CoV*100)
		}
	}

	if printed {
		fmt.Fprintln(w)
	}

	for _, name := range names {
		r := results[name]
		if r == nil || len(r.Warnings) == 0 {
			continue
		}

		for _, msg := range r.Warnings {
			fmt.Fprintf(w, "\n> ⚠ **%s**: %s", name, msg)
		}
	}
}

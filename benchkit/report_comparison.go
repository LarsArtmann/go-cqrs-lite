package benchkit

import (
	"encoding/json/v2"
	"fmt"
	"io"
	"strings"
)

// PrintComparison writes a side-by-side comparison table of multiple results.
// The table includes latency percentiles, GC pause, write amplification, and
// integrity — the metrics that actually drive backend selection decisions.
func PrintComparison(w io.Writer, results map[string]*Result) {
	names := sortedKeys(results)

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Backend Comparison")
	fmt.Fprintln(w, strings.Repeat("=", 120))

	header := fmt.Sprintf(
		"%-10s %10s %10s %10s %10s %10s %10s %6s %8s %8s %8s %10s %10s",
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
		"Heap",
		"Disk",
	)
	fmt.Fprintln(w, header)
	fmt.Fprintln(w, strings.Repeat("-", len(header)))

	for _, name := range names {
		r := results[name]
		printComparisonRow(w, name, r)
	}

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

	fmt.Fprintln(w)
}

func printComparisonRow(w io.Writer, name string, r *Result) {
	if r.Error != "" {
		fmt.Fprintf(w, "%-10s %s\n", name, "FAILED: "+truncate(r.Error, 60))

		return
	}

	covStr := "-"
	if r.RepeatCoV > 0 {
		covStr = fmt.Sprintf("%.1f%%", r.RepeatCoV*100)
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
		w, "%-10s %10s %10s %10s %10s %10s %10s %6s %8s %8s %8s %10s %10s\n",
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
		"| Backend | Write P50 | Write P99 | Load P50 | Load P99 | Cold P50 | GC Max | Write Amp | CoV | Heap | Disk | Integrity |",
	)
	fmt.Fprintln(
		w,
		"|---------|----------:|----------:|---------:|---------:|---------:|-------:|----------:|----:|-----:|-----:|:---------:|",
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
			w, "| %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s |\n",
			name,
			roundDuration(r.WriteLatency.P50),
			roundDuration(r.WriteLatency.P99),
			roundDuration(r.LoadLatency.P50),
			roundDuration(r.LoadLatency.P99),
			roundDuration(r.ColdReadLatency.P50),
			gcMax,
			wrtAmp,
			cov,
			formatBytes(r.Memory.After),
			formatBytes(uint64(r.Disk.DatabaseBytes)),
			integrity,
		)
	}
}

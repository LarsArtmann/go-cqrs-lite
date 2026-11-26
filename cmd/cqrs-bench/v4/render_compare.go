package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/larsartmann/go-output"
	"github.com/larsartmann/go-output/delimited"

	"github.com/larsartmann/go-cqrs-lite/benchkit/v4"
)

// failureCellWidth is how much of a backend's error fits into a FAILED cell.
const failureCellWidth = 30

// renderComparison writes the comparison of several backends in format.
func renderComparison(out reportWriter, format string, results map[string]*benchkit.Result) error {
	switch format {
	case formatJSON:
		if err := benchkit.WriteComparisonJSON(out.w, results); err != nil {
			return writeError(err)
		}
	case formatTable:
		if err := out.table(buildComparisonTable(results)); err != nil {
			return err
		}

		if summary := comparisonWinnerSummary(results); summary != "" {
			fmt.Fprintf(out.w, "\n%s\n", summary)
		}

		benchkit.PrintComparisonVariation(out.w, results)
	case formatMarkdown:
		if err := out.markdownTable(buildComparisonTable(results)); err != nil {
			return err
		}

		benchkit.PrintComparisonVariation(out.w, results)
	case formatCSV:
		if err := delimited.WriteCSV(out.w, buildComparisonTable(results)); err != nil {
			return writeError(err)
		}
	case formatTSV:
		if err := delimited.WriteTSV(out.w, buildComparisonTable(results)); err != nil {
			return writeError(err)
		}
	default:
		benchkit.PrintComparison(out.w, results)
	}

	return nil
}

// comparisonHeaders are the columns of the comparison table.
func comparisonHeaders() []string {
	return []string{
		"Backend", "Write P50", "Write P99", "Load P50", "Load P99",
		"Cold P50", "GC Max Pause", "Tail Ratio", "Allocs/Op",
		"Write Amp", "CoV %", "Noisy", "RAM", "Heap", "Disk",
	}
}

func buildComparisonTable(results map[string]*benchkit.Result) *output.Table {
	headers := comparisonHeaders()
	table := output.NewTable(headers)

	for _, name := range sortedResultKeys(results) {
		table.AddRow(comparisonRow(headers, name, results[name]))
	}

	return table
}

// comparisonRow is one backend's line: its metrics, or a FAILED cell with
// the error.
func comparisonRow(headers []string, name string, result *benchkit.Result) []string {
	switch {
	case result == nil:
		return dashRow(headers, name, "FAILED: no result")
	case result.Error != "":
		return dashRow(headers, name, "FAILED: "+benchkit.Truncate(result.Error, failureCellWidth))
	}

	return []string{
		name,
		fmtDur(result.WriteLatency.P50),
		fmtDur(result.WriteLatency.P99),
		fmtDur(result.LoadLatency.P50),
		fmtDur(result.LoadLatency.P99),
		fmtDur(result.ColdReadLatency.P50),
		fmtGCDash(result.GCMaxPause),
		fmtRatioDash(result.TailRatio),
		fmtAllocDash(result.AllocsPerOp),
		fmtRatioDash(result.Disk.WriteAmplification),
		fmtCoVDash(result.RepeatCoV),
		fmtNoisyCount(result),
		fmtBytes(result.Memory.Resident),
		fmtBytes(result.Memory.After),
		fmtBytes(uint64(result.Disk.DatabaseBytes)),
	}
}

// fmtNoisyCount renders a backend's noisy-metric count (metrics whose
// cross-run CoV exceeded the threshold). "-" when the run carries no
// dispersion data: a single-run compare has no stability verdict to show,
// and showing 0 would fake one.
func fmtNoisyCount(result *benchkit.Result) string {
	if len(result.MetricVariation) == 0 {
		return "-"
	}

	return fmtInt(result.NoisyMetricCount())
}

// metricPick is one "best backend" entry of the winner summary: the metric
// in a comparable unit and how to print its winning value.
type metricPick struct {
	label  string
	value  func(*benchkit.Result) float64
	format func(float64) string
}

// metricWinner is the backend with the lowest value of a metric.
type metricWinner struct {
	name  string
	value float64
}

// fastest finds the backend with the lowest positive value among the runs
// that did not fail.
func (p metricPick) fastest(results map[string]*benchkit.Result) (metricWinner, bool) {
	winner := metricWinner{}
	found := false

	for _, name := range sortedResultKeys(results) {
		result := results[name]
		if result == nil || result.Error != "" {
			continue
		}

		value := p.value(result)
		if value <= 0 {
			continue
		}

		if !found || value < winner.value {
			winner = metricWinner{name: name, value: value}
			found = true
		}
	}

	return winner, found
}

// inNanos reads a latency metric as nanoseconds, the unit metricPick
// compares.
func inNanos(latency func(*benchkit.Result) time.Duration) func(*benchkit.Result) float64 {
	return func(result *benchkit.Result) float64 { return float64(latency(result)) }
}

// asDuration prints a nanosecond value as a duration.
func asDuration(nanos float64) string {
	return fmtDur(time.Duration(nanos))
}

// comparisonWinnerSummary finds the best backend per key metric and returns
// a one-line summary. Returns empty when fewer than 2 valid results.
func comparisonWinnerSummary(results map[string]*benchkit.Result) string {
	picks := []metricPick{
		{
			label: "writes", format: asDuration,
			value: inNanos(func(r *benchkit.Result) time.Duration { return r.WriteLatency.P50 }),
		},
		{
			label: "reads", format: asDuration,
			value: inNanos(func(r *benchkit.Result) time.Duration { return r.LoadLatency.P50 }),
		},
		{
			label: "allocs", format: func(v float64) string { return fmt.Sprintf("%.0f", v) },
			value: func(r *benchkit.Result) float64 { return r.AllocsPerOp },
		},
		{
			label: "heap", format: func(v float64) string { return fmtBytes(uint64(v)) },
			value: func(r *benchkit.Result) float64 { return float64(r.Memory.After) },
		},
		{
			label: "GC pause", format: asDuration,
			value: inNanos(func(r *benchkit.Result) time.Duration { return r.GCMaxPause }),
		},
	}

	var parts []string

	for _, pick := range picks {
		if winner, ok := pick.fastest(results); ok {
			parts = append(parts, fmt.Sprintf("%s: %s (%s)", pick.label, winner.name, pick.format(winner.value)))
		}
	}

	if len(parts) == 0 {
		return ""
	}

	return "Best — " + strings.Join(parts, " | ")
}

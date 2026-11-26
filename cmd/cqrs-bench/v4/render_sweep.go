package main

import (
	"slices"
	"strconv"

	"github.com/larsartmann/go-output"
	"github.com/larsartmann/go-output/delimited"

	"github.com/larsartmann/go-cqrs-lite/benchkit/v4"
)

// covHeader is the sweep table column that appears when points carry repeat
// dispersion.
const covHeader = "CoV %"

// renderSweep writes the scaling-sweep results in format.
func renderSweep(out reportWriter, format string, results []benchkit.SweepResult) error {
	switch format {
	case formatJSON:
		if err := benchkit.WriteSweepJSON(out.w, results); err != nil {
			return writeError(err)
		}
	case formatTable:
		return out.table(buildSweepTable(results))
	case formatMarkdown:
		return out.markdownTable(buildSweepTable(results))
	case formatCSV:
		if err := delimited.WriteCSV(out.w, buildSweepTable(results)); err != nil {
			return writeError(err)
		}
	case formatTSV:
		if err := delimited.WriteTSV(out.w, buildSweepTable(results)); err != nil {
			return writeError(err)
		}
	default:
		benchkit.PrintSweep(out.w, results)
	}

	return nil
}

func buildSweepTable(results []benchkit.SweepResult) *output.Table {
	if len(results) == 0 {
		return output.NewTable(nil)
	}

	headers := sweepHeaders(results)
	table := output.NewTable(headers)

	for _, sweep := range results {
		table.AddRow(sweepRow(headers, sweep))
	}

	return table
}

// sweepHeaders are the sweep table columns. The CoV column appears exactly
// when the sweep ran with --repeat N: it is the cross-run dispersion of each
// point's own repeats, not a new metric.
func sweepHeaders(results []benchkit.SweepResult) []string {
	headers := []string{
		benchkit.TitleCase(results[0].Parameter), "Write P50", "Write P99", "Load P50",
		"GC Max Pause", "Allocs/Op", "Write Amp", "Heap",
	}

	if withCoV(results) {
		headers = append(headers, covHeader)
	}

	return headers
}

// sweepRow is one data point: its metrics, or a FAILED cell with the error.
func sweepRow(headers []string, sweep benchkit.SweepResult) []string {
	value := strconv.Itoa(sweep.Value)
	result := sweep.Result

	switch {
	case result == nil:
		return dashRow(headers, value, "FAILED: no result")
	case result.Error != "":
		return dashRow(headers, value, "FAILED: "+benchkit.Truncate(result.Error, failureCellWidth))
	}

	row := []string{
		value,
		fmtDur(result.WriteLatency.P50),
		fmtDur(result.WriteLatency.P99),
		fmtDur(result.LoadLatency.P50),
		fmtGCDash(result.GCMaxPause),
		fmtAllocDash(result.AllocsPerOp),
		fmtRatioDash(result.Disk.WriteAmplification),
		fmtBytes(result.Memory.After),
	}

	if slices.Contains(headers, covHeader) {
		row = append(row, fmtCoVDash(result.RepeatCoV))
	}

	return row
}

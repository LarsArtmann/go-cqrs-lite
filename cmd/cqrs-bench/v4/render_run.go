package main

import (
	"fmt"
	"io"
	"strconv"

	"github.com/larsartmann/go-output"
	"github.com/larsartmann/go-output/delimited"

	"github.com/larsartmann/go-cqrs-lite/benchkit/v4"
)

// runFailureWidth is how much of a run error fits into the summary table.
const runFailureWidth = 60

// renderRunResult writes the result of a run command in format.
func renderRunResult(
	out reportWriter,
	format string,
	config benchkit.Config,
	result *benchkit.Result,
	repeated *benchkit.RepeatedResult,
	includeRuns bool,
) error {
	switch format {
	case formatTable:
		if err := out.table(buildRunSummaryTable(result)); err != nil {
			return err
		}

		if result.IntegrityErrors > 0 {
			fmt.Fprintf(out.w, "\n⚠ CORRUPTION: %d integrity errors detected!\n", result.IntegrityErrors)
		}
	case formatJSON:
		if err := benchkit.WriteJSON(out.w, result); err != nil {
			return writeError(err)
		}
	case formatBenchstat:
		// With repeats, emit one line per run: benchstat needs n>1 samples
		// per benchmark before it reports a confidence interval.
		if repeated != nil && len(repeated.Runs) > 1 {
			benchkit.WriteBenchstatRepeated(out.w, repeated)
		} else {
			benchkit.WriteBenchstat(out.w, result)
		}
	case formatManifest:
		if err := writeManifest(out.w, config, result, repeated, includeRuns); err != nil {
			return err
		}
	case formatCSV:
		if err := delimited.WriteCSV(out.w, buildRunSummaryTable(result)); err != nil {
			return writeError(err)
		}
	case formatTSV:
		if err := delimited.WriteTSV(out.w, buildRunSummaryTable(result)); err != nil {
			return writeError(err)
		}
	case formatMarkdown:
		return out.markdownTable(buildRunSummaryTable(result))
	default:
		benchkit.PrintReport(out.w, result)
	}

	return nil
}

// writeManifest writes config, environment, and result as JSON.
// --include-runs opts into serializing every repeat run: N repeats multiply
// the manifest N-fold, so the median-only default stays the
// machine-readable standard and raw per-run data is a deliberate ask.
func writeManifest(
	w io.Writer,
	config benchkit.Config,
	result *benchkit.Result,
	repeated *benchkit.RepeatedResult,
	includeRuns bool,
) error {
	if includeRuns && repeated != nil {
		if err := benchkit.WriteManifestRepeated(w, config, repeated); err != nil {
			return writeError(err)
		}

		return nil
	}

	if err := benchkit.WriteManifest(w, config, result); err != nil {
		return writeError(err)
	}

	return nil
}

// buildRunSummaryTable creates the two-column metric/value summary table of
// the run command's table, CSV, TSV, and markdown output. It shows the key
// metrics without the full text report.
func buildRunSummaryTable(result *benchkit.Result) *output.Table {
	if result.Error != "" {
		table := output.NewTable([]string{"Status", "Message"})
		table.AddRow([]string{"FAILED", benchkit.Truncate(result.Error, runFailureWidth)})

		return table
	}

	return runSummaryTable(result)
}

// runSummaryTable lists the key metrics of a successful run.
func runSummaryTable(result *benchkit.Result) *output.Table {
	table := output.NewTable([]string{"Metric", "Value"})

	table.AddRow([]string{"Backend", result.Backend})
	table.AddRow([]string{"Profile", result.Profile})
	table.AddRow([]string{"Codec", result.Codec})
	table.AddRow([]string{"Events", fmt.Sprintf("%s streams × %d = %s",
		fmtInt(result.Streams), result.EventsPerStream, fmtInt(result.TotalEvents))})

	addRowIf(table, "Payload", fmt.Sprintf("%d bytes/event", result.PayloadBytes), result.PayloadBytes > 0)

	table.AddRow([]string{"Duration", fmtDur(result.Duration)})
	table.AddRow([]string{"Workers", strconv.Itoa(result.Workers)})

	addRowIf(table, "Load1 (start)", fmt.Sprintf("%.1f", result.Environment.LoadAvg1),
		result.Environment.LoadAvg1 > 0)

	if result.WriteLatency.Count > 0 {
		table.AddRow([]string{"Write P50", fmtDur(result.WriteLatency.P50)})
		table.AddRow([]string{"Write P99", fmtDur(result.WriteLatency.P99)})
		table.AddRow([]string{"Write Min", fmtDur(result.WriteLatency.Min) + " (exact fastest)"})
	}

	addRowIf(table, "Write Throughput", fmtFloat(result.WriteThroughput)+" events/s", result.WriteThroughput > 0)

	if result.LoadLatency.Count > 0 {
		table.AddRow([]string{"Load P50", fmtDur(result.LoadLatency.P50)})
		table.AddRow([]string{"Load P99", fmtDur(result.LoadLatency.P99)})
	}

	addRowIf(table, "Cold Load P50", fmtDur(result.ColdReadLatency.P50), result.ColdReadLatency.Count > 0)
	addRowIf(table, "GC Max Pause", fmtDur(result.GCMaxPause), result.GCMaxPause > 0)
	addRowIf(table, "Allocs/Op", fmt.Sprintf("%.0f", result.AllocsPerOp), result.AllocsPerOp > 0)

	table.AddRow([]string{"RAM Resident", fmtBytes(result.Memory.Resident)})
	table.AddRow([]string{"Heap Peak", fmtBytes(result.Memory.After)})

	addRowIf(table, "Disk", fmtBytes(uint64(result.Disk.DatabaseBytes)), result.Disk.DatabaseBytes > 0)

	addVariationRows(table, result)

	addRowIf(table, "Integrity Errors", strconv.Itoa(result.IntegrityErrors), result.IntegrityErrors > 0)

	return table
}

// addRowIf adds a metric row only when the metric was measured.
func addRowIf(table *output.Table, label, value string, measured bool) {
	if measured {
		table.AddRow([]string{label, value})
	}
}

package main

import (
	"fmt"
	"io"
	"strconv"

	"github.com/larsartmann/go-output"
	"github.com/larsartmann/go-output/delimited"
	gotable "github.com/larsartmann/go-output/table"

	"github.com/larsartmann/go-cqrs-lite/benchkit/v4"
)

// ── run result rendering ──

func renderRunResult(
	w io.Writer,
	format string,
	config benchkit.Config,
	result *benchkit.Result,
	repeated *benchkit.RepeatedResult,
	includeRuns bool,
) {
	switch format {
	case formatTable:
		data := buildRunSummaryTable(result)
		if err := gotable.Write(w, data, gotable.WithColorMode(output.ColorModeAuto)); err != nil {
			fatalf("render table: %v", err)
		}

		if result.IntegrityErrors > 0 {
			fmt.Fprintf(
				w,
				"\n⚠ CORRUPTION: %d integrity errors detected!\n",
				result.IntegrityErrors,
			)
		}
	case formatJSON:
		if err := benchkit.WriteJSON(w, result); err != nil {
			fatalf("write JSON: %v", err)
		}
	case formatBenchstat:
		// With repeats, emit one line per run: benchstat needs n>1 samples
		// per benchmark before it reports a confidence interval.
		if repeated != nil && len(repeated.Runs) > 1 {
			benchkit.WriteBenchstatRepeated(w, repeated)
		} else {
			benchkit.WriteBenchstat(w, result)
		}
	case formatManifest:
		// --include-runs opts into serializing every repeat run: N repeats
		// multiply the manifest N-fold, so the median-only default stays the
		// machine-readable standard and raw per-run data is a deliberate ask.
		if includeRuns && repeated != nil {
			if err := benchkit.WriteManifestRepeated(w, config, repeated); err != nil {
				fatalf("write manifest: %v", err)
			}
		} else if err := benchkit.WriteManifest(w, config, result); err != nil {
			fatalf("write manifest: %v", err)
		}
	case formatCSV:
		data := buildRunSummaryTable(result)

		if err := delimited.WriteCSV(w, data); err != nil {
			fatalf("render CSV: %v", err)
		}
	case formatTSV:
		data := buildRunSummaryTable(result)

		if err := delimited.WriteTSV(w, data); err != nil {
			fatalf("render TSV: %v", err)
		}
	default:
		benchkit.PrintReport(w, result)
	}
}

// buildRunSummaryTable creates a 2-column metric/value summary table for
// the run command's --format table output. Shows the key metrics without
// the full detailed report.
func buildRunSummaryTable(r *benchkit.Result) *output.Table {
	if r.Error != "" {
		t := output.NewTable([]string{"Status", "Message"})
		t.AddRow([]string{"FAILED", benchkit.Truncate(r.Error, 60)})

		return t
	}

	t := output.NewTable([]string{"Metric", "Value"})

	t.AddRow([]string{"Backend", r.Backend})
	t.AddRow([]string{"Profile", r.Profile})
	t.AddRow([]string{"Codec", r.Codec})
	t.AddRow([]string{"Events", fmt.Sprintf("%s streams × %d = %s",
		fmtInt(r.Streams), r.EventsPerStream, fmtInt(r.TotalEvents))})

	if r.PayloadBytes > 0 {
		t.AddRow([]string{"Payload", fmt.Sprintf("%d bytes/event", r.PayloadBytes)})
	}

	t.AddRow([]string{"Duration", fmtDur(r.Duration)})
	t.AddRow([]string{"Workers", strconv.Itoa(r.Workers)})

	if r.WriteLatency.Count > 0 {
		t.AddRow([]string{"Write P50", fmtDur(r.WriteLatency.P50)})
		t.AddRow([]string{"Write P99", fmtDur(r.WriteLatency.P99)})
	}

	if r.WriteThroughput > 0 {
		t.AddRow(
			[]string{"Write Throughput", fmtFloat(r.WriteThroughput) + " events/s"},
		)
	}

	if r.LoadLatency.Count > 0 {
		t.AddRow([]string{"Load P50", fmtDur(r.LoadLatency.P50)})
		t.AddRow([]string{"Load P99", fmtDur(r.LoadLatency.P99)})
	}

	if r.ColdReadLatency.Count > 0 {
		t.AddRow([]string{"Cold Load P50", fmtDur(r.ColdReadLatency.P50)})
	}

	if r.GCMaxPause > 0 {
		t.AddRow([]string{"GC Max Pause", fmtDur(r.GCMaxPause)})
	}

	if r.AllocsPerOp > 0 {
		t.AddRow([]string{"Allocs/Op", fmt.Sprintf("%.0f", r.AllocsPerOp)})
	}

	t.AddRow([]string{"RAM Resident", fmtBytes(r.Memory.Resident)})
	t.AddRow([]string{"Heap Peak", fmtBytes(r.Memory.After)})

	if r.Disk.DatabaseBytes > 0 {
		t.AddRow([]string{"Disk", fmtBytes(uint64(r.Disk.DatabaseBytes))})
	}

	addVariationRows(t, r)

	if r.IntegrityErrors > 0 {
		t.AddRow([]string{"Integrity Errors", strconv.Itoa(r.IntegrityErrors)})
	}

	return t
}

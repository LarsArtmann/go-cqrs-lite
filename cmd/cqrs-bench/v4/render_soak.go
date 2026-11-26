package main

import (
	"fmt"
	"io"
	"strconv"

	"github.com/larsartmann/go-output"
	"github.com/larsartmann/go-output/delimited"

	"github.com/larsartmann/go-cqrs-lite/benchkit/v4"
)

// renderSoakResult writes a soak result in format.
func renderSoakResult(out reportWriter, format string, result *benchkit.SoakResult) error {
	switch format {
	case formatJSON:
		if err := benchkit.WriteSoakJSON(out.w, result); err != nil {
			return writeError(err)
		}
	case formatTable:
		if err := out.table(buildSoakTable(result)); err != nil {
			return err
		}

		printSoakSummary(out.w, result)
	case formatCSV:
		if err := delimited.WriteCSV(out.w, buildSoakTable(result)); err != nil {
			return writeError(err)
		}
	case formatTSV:
		if err := delimited.WriteTSV(out.w, buildSoakTable(result)); err != nil {
			return writeError(err)
		}
	case formatMarkdown:
		if err := out.markdownTable(buildSoakTable(result)); err != nil {
			return err
		}

		printSoakSummary(out.w, result)
	default:
		benchkit.PrintSoakReport(out.w, result)
	}

	return nil
}

// buildSoakTable is one row per soak iteration.
func buildSoakTable(result *benchkit.SoakResult) *output.Table {
	table := output.NewTable([]string{"Iter", "Throughput", "Heap", "Write P50", "Write P99"})

	for _, sample := range result.Samples {
		table.AddRow([]string{
			strconv.Itoa(sample.Iteration + 1),
			fmtFloat(sample.Throughput) + "/s",
			fmtBytes(sample.HeapBytes),
			fmtDur(sample.WriteP50),
			fmtDur(sample.WriteP99),
		})
	}

	return table
}

// printSoakSummary reports the first-to-last drift of a soak.
func printSoakSummary(w io.Writer, result *benchkit.SoakResult) {
	if len(result.Samples) == 0 {
		return
	}

	first, last, _ := result.FirstAndLastSample()

	fmt.Fprintf(w, "\nDrift over %s (%d iterations):\n", fmtDur(result.Duration), result.Iterations)
	fmt.Fprintf(w, "  Throughput: %s/s → %s/s (%+.1f%%)\n",
		fmtFloat(first.Throughput), fmtFloat(last.Throughput), result.ThroughputDriftPct)
	fmt.Fprintf(w, "  Write P99:  %s → %s (%+.1f%%)\n",
		fmtDur(first.WriteP99), fmtDur(last.WriteP99), result.WriteP99DriftPct)
	fmt.Fprintf(w, "  Heap:       %s → %s (growth: %s, %s/iter)\n",
		fmtBytes(first.HeapBytes), fmtBytes(last.HeapBytes),
		fmtBytes(result.HeapGrowthBytes), fmtBytes(uint64(result.HeapLeakRate)))
}

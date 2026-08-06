package main

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/larsartmann/go-output"
	"github.com/larsartmann/go-output/delimited"
	"github.com/larsartmann/go-output/markdown"
	gotable "github.com/larsartmann/go-output/table"

	"github.com/larsartmann/go-cqrs-lite/benchkit/v4"
)

// ── format resolution ──

const (
	formatAuto      = "auto"
	formatTable     = "table"
	formatText      = "text"
	formatJSON      = "json"
	formatCSV       = "csv"
	formatTSV       = "tsv"
	formatMarkdown  = "markdown"
	formatBenchstat = "benchstat"
	formatManifest  = "manifest"
)

// resolveFormat picks "table" (styled terminal) when stdout is a terminal
// and "text" (plain) otherwise. Explicit format choices pass through.
func resolveFormat(format string) string {
	if format == "" || format == formatAuto {
		if output.ColorModeAuto.ShouldColor() {
			return formatTable
		}

		return formatText
	}

	return format
}

// ── comparison rendering ──

func renderComparison(w io.Writer, format string, results map[string]*benchkit.Result) {
	switch format {
	case formatJSON:
		if err := benchkit.WriteComparisonJSON(w, results); err != nil {
			fatalf("write JSON: %v", err)
		}
	case formatTable:
		data := buildComparisonTable(results)
		if err := gotable.Write(w, data, gotable.WithColorMode(output.ColorModeAuto)); err != nil {
			fatalf("render table: %v", err)
		}

		if summary := comparisonWinnerSummary(results); summary != "" {
			fmt.Fprintf(w, "\n%s\n", summary)
		}
	case formatMarkdown:
		data := buildComparisonTable(results)

		rendered, err := markdown.Render(data, markdown.WithColorMode(output.ColorModeNever))
		if err != nil {
			fatalf("render markdown: %v", err)
		}

		rendered = strings.TrimPrefix(rendered, "|")
		fmt.Fprint(w, rendered)
	case formatCSV:
		data := buildComparisonTable(results)

		if err := delimited.WriteCSV(w, data); err != nil {
			fatalf("render CSV: %v", err)
		}
	case formatTSV:
		data := buildComparisonTable(results)

		if err := delimited.WriteTSV(w, data); err != nil {
			fatalf("render TSV: %v", err)
		}
	default:
		benchkit.PrintComparison(w, results)
	}
}

func buildComparisonTable(results map[string]*benchkit.Result) *output.Table {
	headers := []string{
		"Backend", "Write P50", "Write P99", "Load P50", "Load P99",
		"Cold P50", "GC Max Pause", "Tail Ratio", "Allocs/Op",
		"Write Amp", "CoV %", "RAM", "Heap", "Disk",
	}

	t := output.NewTable(headers)

	empty := make([]string, len(headers)-1)
	for i := range empty {
		empty[i] = "-"
	}

	for _, name := range sortedResultKeys(results) {
		r := results[name]

		if r == nil {
			t.AddRow(append([]string{name, "FAILED: no result"}, empty[1:]...))

			continue
		}

		if r.Error != "" {
			t.AddRow(append([]string{name, "FAILED: " + truncateMsg(r.Error, 30)}, empty[1:]...))

			continue
		}

		t.AddRow([]string{
			name,
			fmtDur(r.WriteLatency.P50),
			fmtDur(r.WriteLatency.P99),
			fmtDur(r.LoadLatency.P50),
			fmtDur(r.LoadLatency.P99),
			fmtDur(r.ColdReadLatency.P50),
			fmtGCDash(r.GCMaxPause),
			fmtRatioDash(r.TailRatio),
			fmtAllocDash(r.AllocsPerOp),
			fmtRatioDash(r.Disk.WriteAmplification),
			fmtCoVDash(r.RepeatCoV),
			fmtBytes(r.Memory.Resident),
			fmtBytes(r.Memory.After),
			fmtBytes(uint64(r.Disk.DatabaseBytes)),
		})
	}

	return t
}

// comparisonWinnerSummary finds the best backend per key metric and returns
// a one-line summary. Returns empty when fewer than 2 valid results.
func comparisonWinnerSummary(results map[string]*benchkit.Result) string {
	names := sortedResultKeys(results)

	type metricBest struct {
		label string
		name  string
		val   string
	}

	var bests []metricBest

	findMinDur := func(label string, get func(r *benchkit.Result) (time.Duration, bool)) {
		var (
			winner string
			minVal time.Duration
		)

		found := false

		for _, name := range names {
			r := results[name]
			if r == nil || r.Error != "" {
				continue
			}

			v, ok := get(r)
			if !ok || v <= 0 {
				continue
			}

			if !found || v < minVal {
				minVal = v
				winner = name
				found = true
			}
		}

		if found {
			bests = append(bests, metricBest{label, winner, fmtDur(minVal)})
		}
	}

	findMinNum := func(label string, get func(r *benchkit.Result) (float64, bool), format func(float64) string) {
		var (
			winner string
			minVal float64
		)

		found := false

		for _, name := range names {
			r := results[name]
			if r == nil || r.Error != "" {
				continue
			}

			v, ok := get(r)
			if !ok || v <= 0 {
				continue
			}

			if !found || v < minVal {
				minVal = v
				winner = name
				found = true
			}
		}

		if found {
			bests = append(bests, metricBest{label, winner, format(minVal)})
		}
	}

	findMinDur("writes", func(r *benchkit.Result) (time.Duration, bool) {
		return r.WriteLatency.P50, r.WriteLatency.P50 > 0
	})

	findMinDur("reads", func(r *benchkit.Result) (time.Duration, bool) {
		return r.LoadLatency.P50, r.LoadLatency.P50 > 0
	})

	findMinNum("allocs", func(r *benchkit.Result) (float64, bool) {
		return r.AllocsPerOp, r.AllocsPerOp > 0
	}, func(v float64) string { return fmt.Sprintf("%.0f", v) })

	findMinNum("heap", func(r *benchkit.Result) (float64, bool) {
		return float64(r.Memory.After), r.Memory.After > 0
	}, func(v float64) string { return fmtBytes(uint64(v)) })

	findMinDur("GC pause", func(r *benchkit.Result) (time.Duration, bool) {
		return r.GCMaxPause, r.GCMaxPause > 0
	})

	if len(bests) == 0 {
		return ""
	}

	parts := make([]string, len(bests))
	for i, b := range bests {
		parts[i] = fmt.Sprintf("%s: %s (%s)", b.label, b.name, b.val)
	}

	return "Best — " + strings.Join(parts, " | ")
}

// ── sweep rendering ──

func renderSweep(w io.Writer, format string, results []benchkit.SweepResult) {
	switch format {
	case formatJSON:
		if err := benchkit.WriteSweepJSON(w, results); err != nil {
			fatalf("write JSON: %v", err)
		}
	case formatTable:
		data := buildSweepTable(results)
		if err := gotable.Write(w, data, gotable.WithColorMode(output.ColorModeAuto)); err != nil {
			fatalf("render table: %v", err)
		}
	case formatMarkdown:
		data := buildSweepTable(results)

		rendered, err := markdown.Render(data, markdown.WithColorMode(output.ColorModeNever))
		if err != nil {
			fatalf("render markdown: %v", err)
		}

		rendered = strings.TrimPrefix(rendered, "|")
		fmt.Fprint(w, rendered)
	case formatCSV:
		data := buildSweepTable(results)

		if err := delimited.WriteCSV(w, data); err != nil {
			fatalf("render CSV: %v", err)
		}
	case formatTSV:
		data := buildSweepTable(results)

		if err := delimited.WriteTSV(w, data); err != nil {
			fatalf("render TSV: %v", err)
		}
	default:
		benchkit.PrintSweep(w, results)
	}
}

func buildSweepTable(results []benchkit.SweepResult) *output.Table {
	if len(results) == 0 {
		return output.NewTable(nil)
	}

	param := titleCase(results[0].Parameter)
	headers := []string{
		param, "Write P50", "Write P99", "Load P50",
		"GC Max Pause", "Allocs/Op", "Write Amp", "Heap",
	}

	t := output.NewTable(headers)

	empty := make([]string, len(headers)-1)
	for i := range empty {
		empty[i] = "-"
	}

	for _, sr := range results {
		r := sr.Result
		if r == nil {
			t.AddRow(
				append([]string{strconv.Itoa(sr.Value), "FAILED: no result"}, empty[1:]...),
			)

			continue
		}

		if r.Error != "" {
			t.AddRow(
				append(
					[]string{strconv.Itoa(sr.Value), "FAILED: " + truncateMsg(r.Error, 30)},
					empty[1:]...),
			)

			continue
		}

		t.AddRow([]string{
			strconv.Itoa(sr.Value),
			fmtDur(r.WriteLatency.P50),
			fmtDur(r.WriteLatency.P99),
			fmtDur(r.LoadLatency.P50),
			fmtGCDash(r.GCMaxPause),
			fmtAllocDash(r.AllocsPerOp),
			fmtRatioDash(r.Disk.WriteAmplification),
			fmtBytes(r.Memory.After),
		})
	}

	return t
}

// ── run result rendering ──

func renderRunResult(w io.Writer, format string, config benchkit.Config, result *benchkit.Result) {
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
		benchkit.WriteBenchstat(w, result)
	case formatManifest:
		if err := benchkit.WriteManifest(w, config, result); err != nil {
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
		t.AddRow([]string{"FAILED", truncateMsg(r.Error, 60)})

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

	if r.RepeatCount > 1 {
		t.AddRow(
			[]string{
				"Repeat",
				fmt.Sprintf("median of %d (CoV %.1f%%)", r.RepeatCount, r.RepeatCoV*100),
			},
		)
	}

	if r.IntegrityErrors > 0 {
		t.AddRow([]string{"Integrity Errors", strconv.Itoa(r.IntegrityErrors)})
	}

	return t
}

// ── soak rendering ──

func renderSoakResult(w io.Writer, format string, result *benchkit.SoakResult) {
	switch format {
	case formatJSON:
		if err := benchkit.WriteSoakJSON(w, result); err != nil {
			fatalf("write JSON: %v", err)
		}
	case formatTable:
		data := buildSoakTable(result)
		if err := gotable.Write(w, data, gotable.WithColorMode(output.ColorModeAuto)); err != nil {
			fatalf("render table: %v", err)
		}

		printSoakSummary(w, result)
	case formatCSV:
		data := buildSoakTable(result)

		if err := delimited.WriteCSV(w, data); err != nil {
			fatalf("render CSV: %v", err)
		}
	case formatTSV:
		data := buildSoakTable(result)

		if err := delimited.WriteTSV(w, data); err != nil {
			fatalf("render TSV: %v", err)
		}
	default:
		benchkit.PrintSoakReport(w, result)
	}
}

func buildSoakTable(r *benchkit.SoakResult) *output.Table {
	headers := []string{"Iter", "Throughput", "Heap", "Write P50", "Write P99"}
	t := output.NewTable(headers)

	for _, s := range r.Samples {
		t.AddRow([]string{
			strconv.Itoa(s.Iteration + 1),
			fmtFloat(s.Throughput) + "/s",
			fmtBytes(s.HeapBytes),
			fmtDur(s.WriteP50),
			fmtDur(s.WriteP99),
		})
	}

	return t
}

func printSoakSummary(w io.Writer, r *benchkit.SoakResult) {
	if len(r.Samples) == 0 {
		return
	}

	first := r.Samples[0]
	last := r.Samples[len(r.Samples)-1]

	fmt.Fprintf(w, "\nDrift over %s (%d iterations):\n", fmtDur(r.Duration), r.Iterations)
	fmt.Fprintf(w, "  Throughput: %s/s → %s/s (%+.1f%%)\n",
		fmtFloat(first.Throughput), fmtFloat(last.Throughput), r.ThroughputDriftPct)
	fmt.Fprintf(w, "  Write P99:  %s → %s (%+.1f%%)\n",
		fmtDur(first.WriteP99), fmtDur(last.WriteP99), r.WriteP99DriftPct)
	fmt.Fprintf(w, "  Heap:       %s → %s (growth: %s, %s/iter)\n",
		fmtBytes(first.HeapBytes), fmtBytes(last.HeapBytes),
		fmtBytes(r.HeapGrowthBytes), fmtBytes(uint64(r.HeapLeakRate)))
}

// ── formatting helpers ──

func sortedResultKeys(m map[string]*benchkit.Result) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	return keys
}

// fmtDur wraps benchkit.FormatDuration so render.go callers stay short.
func fmtDur(d time.Duration) string { return benchkit.FormatDuration(d) }

// fmtBytes wraps benchkit.FormatBytes.
func fmtBytes(b uint64) string { return benchkit.FormatBytes(b) }

// fmtFloat wraps benchkit.FormatFloat.
func fmtFloat(f float64) string { return benchkit.FormatFloat(f) }

// fmtInt wraps benchkit.FormatInt.
func fmtInt(n int) string { return benchkit.FormatInt(n) }

func fmtGCDash(d time.Duration) string {
	if d == 0 {
		return "-"
	}

	return fmtDur(d)
}

func fmtRatioDash(r float64) string {
	if r <= 0 {
		return "-"
	}

	return fmt.Sprintf("%.1fx", r)
}

func fmtAllocDash(a float64) string {
	if a <= 0 {
		return "-"
	}

	return fmt.Sprintf("%.0f", a)
}

func fmtCoVDash(c float64) string {
	if c <= 0 {
		return "-"
	}

	return fmt.Sprintf("%.1f%%", c*100)
}

func truncateMsg(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	return s[:maxLen-3] + "..."
}

func titleCase(s string) string {
	if s == "" {
		return s
	}

	return strings.ToUpper(s[:1]) + s[1:]
}

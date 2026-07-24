package benchkit

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
)

// durationMarshalers serializes time.Duration as nanoseconds (int64)
// because JSON v2 has no default representation for time.Duration.
var durationMarshalers = json.MarshalFunc(
	func(d time.Duration) ([]byte, error) {
		return []byte(strconv.FormatInt(d.Nanoseconds(), 10)), nil
	},
)

// jsonOpts are the default JSON encoding options: indented output
// with time.Duration serialized as nanoseconds.
var jsonOpts = json.JoinOptions(
	jsontext.WithIndent("  "),
	json.WithMarshalers(durationMarshalers),
)

// PrintReport writes a human-readable text report for a single result.
func PrintReport(w io.Writer, r *Result) {
	if r.Error != "" {
		fmt.Fprintf(w, "Benchmark FAILED: %s\n", r.Error)

		return
	}

	fmt.Fprintf(w, "Benchmark: %s | profile=%s | codec=%s\n",
		r.Backend, r.Profile, r.Codec)
	fmt.Fprintln(w, strings.Repeat("=", 60))
	fmt.Fprintf(w, "Workload: %s streams x %d events = %s events\n",
		formatInt(r.Streams), r.EventsPerStream, formatInt(r.TotalEvents))

	if len(r.PayloadSizes) > 1 {
		fmt.Fprintf(
			w,
			"Payload:  %d bytes/event (mean; mixed %v)\n",
			r.PayloadBytes,
			r.PayloadSizes,
		)
	} else {
		fmt.Fprintf(w, "Payload:  %d bytes/event\n", r.PayloadBytes)
	}

	fmt.Fprintf(w, "Duration: %s\n\n", roundDuration(r.Duration))

	if r.Environment.GoVersion != "" {
		fmt.Fprintf(w, "Env: %s | %s/%s | CPU=%d GOMAXPROCS=%d workers=%d\n\n",
			r.Environment.GoVersion, r.Environment.GOOS, r.Environment.GOARCH,
			r.Environment.NumCPU, r.Environment.GOMAXPROCS, r.Workers)
	}

	if r.RepeatCount > 1 {
		fmt.Fprintf(
			w,
			"Repeat:  median of %d runs (min: %s/s, max: %s/s)\n\n",
			r.RepeatCount,
			formatFloat(r.RepeatMin),
			formatFloat(r.RepeatMax),
		)
	}

	printLatencySection(
		w,
		"Raw Sink (prebuilt events, Save only):",
		r.RawSinkLatency,
		r.RawSinkThroughput,
	)
	printLatencySection(
		w,
		"Write Performance (generated + Save):",
		r.WriteLatency,
		r.WriteThroughput,
	)
	printLatencySection(w, "Read Performance:", r.LoadLatency, 0)

	if r.ReadAllTime > 0 {
		fmt.Fprintf(w, "  ReadAll:  %s\n", roundDuration(r.ReadAllTime))
	}

	if r.ReadFromTime > 0 {
		fmt.Fprintf(w, "  ReadFrom: %s\n", roundDuration(r.ReadFromTime))
	}

	fmt.Fprintln(w)

	if r.ReadModelSet.Count > 0 {
		fmt.Fprintln(w, "Read Model:")
		printLatencyLine(w, "  Set:", r.ReadModelSet)
		printLatencyLine(w, "  Get:", r.ReadModelGet)
		fmt.Fprintln(w)
	}

	if r.ProjectionEvents > 0 {
		fmt.Fprintf(w, "Projection: %s events, lag=%s\n\n",
			formatInt(int(r.ProjectionEvents)), roundDuration(r.ProjectionLag))
	}

	if r.RecoveryTime > 0 {
		fmt.Fprintf(w, "Recovery: %s (%s events recovered)\n\n",
			roundDuration(r.RecoveryTime), formatInt(r.RecoveredEvents))
	}

	fmt.Fprintln(w, "Resources:")
	fmt.Fprintf(w, "  Heap:  %s peak\n", formatBytes(r.Memory.After))
	fmt.Fprintf(w, "  Delta: %s\n", formatBytes(r.Memory.Delta))
	fmt.Fprintf(w, "  CPU:   %s\n", formatCPUDuration(r.CPU.Delta))

	if r.Disk.DatabaseBytes > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Storage:")
		fmt.Fprintf(w, "  Database: %s\n", formatBytes(uint64(r.Disk.DatabaseBytes)))
		fmt.Fprintf(w, "  Events:   %s\n", formatBytes(uint64(r.Disk.EventBytes)))
		fmt.Fprintf(w, "  Overhead: %.1f%%\n", r.Disk.OverheadPct)
	}
}

func printLatencySection(w io.Writer, header string, stats LatencyStats, throughput float64) {
	fmt.Fprintln(w, header)
	printLatencyLine(w, "  Latency:", stats)

	if throughput > 0 {
		fmt.Fprintf(w, "  Throughput: %s events/sec\n", formatFloat(throughput))
	}

	fmt.Fprintln(w)
}

func printLatencyLine(w io.Writer, label string, stats LatencyStats) {
	if stats.Count == 0 {
		return
	}

	fmt.Fprintf(
		w, "%s P50=%s P95=%s P99=%s Max=%s\n",
		label,
		roundDuration(stats.P50),
		roundDuration(stats.P95),
		roundDuration(stats.P99),
		roundDuration(stats.P100),
	)
}

// PrintComparison writes a side-by-side comparison table of multiple results.
func PrintComparison(w io.Writer, results map[string]*Result) {
	names := sortedKeys(results)

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Backend Comparison")
	fmt.Fprintln(w, strings.Repeat("=", 80))

	header := fmt.Sprintf(
		"%-10s %12s %12s %12s %12s %10s %10s",
		"Backend", "Write P50", "Write P99", "Load P50", "Load P99", "Heap MB", "Disk MB",
	)
	fmt.Fprintln(w, header)
	fmt.Fprintln(w, strings.Repeat("-", len(header)))

	for _, name := range names {
		r := results[name]
		printComparisonRow(w, name, r)
	}

	fmt.Fprintln(w)
}

func printComparisonRow(w io.Writer, name string, r *Result) {
	if r.Error != "" {
		fmt.Fprintf(w, "%-10s %s\n", name, "skipped: "+truncate(r.Error, 40))

		return
	}

	fmt.Fprintf(
		w, "%-10s %12s %12s %12s %12s %10s %10s\n",
		name,
		roundDuration(r.WriteLatency.P50),
		roundDuration(r.WriteLatency.P99),
		roundDuration(r.LoadLatency.P50),
		roundDuration(r.LoadLatency.P99),
		formatBytes(r.Memory.After),
		formatBytes(uint64(r.Disk.DatabaseBytes)),
	)
}

// WriteJSON serializes a result as indented JSON.
func WriteJSON(w io.Writer, r *Result) error {
	return json.MarshalWrite(w, r, jsonOpts)
}

// WriteComparisonJSON serializes all results as a JSON object.
func WriteComparisonJSON(w io.Writer, results map[string]*Result) error {
	return json.MarshalWrite(w, results, jsonOpts)
}

// PrintMarkdown writes a markdown comparison table.
func PrintMarkdown(w io.Writer, results map[string]*Result) {
	names := sortedKeys(results)

	fmt.Fprintln(
		w,
		"| Backend | Write P50 | Write P99 | Load P50 | Load P99 | Throughput | Heap | Disk |",
	)
	fmt.Fprintln(
		w,
		"|---------|----------:|----------:|---------:|---------:|-----------:|-----:|-----:|",
	)

	for _, name := range names {
		r := results[name]
		if r.Error != "" {
			fmt.Fprintf(w, "| %s | skipped | | | | | | |", name)

			continue
		}

		fmt.Fprintf(
			w, "| %s | %s | %s | %s | %s | %s/s | %s | %s |\n",
			name,
			roundDuration(r.WriteLatency.P50),
			roundDuration(r.WriteLatency.P99),
			roundDuration(r.LoadLatency.P50),
			roundDuration(r.LoadLatency.P99),
			formatFloat(r.WriteThroughput),
			formatBytes(r.Memory.After),
			formatBytes(uint64(r.Disk.DatabaseBytes)),
		)
	}
}

// ── formatting helpers ──

func roundDuration(d time.Duration) time.Duration {
	switch {
	case d < time.Microsecond:
		return d.Round(time.Nanosecond)
	case d < time.Millisecond:
		return d.Round(100 * time.Nanosecond)
	case d < time.Second:
		return d.Round(time.Microsecond)
	default:
		return d.Round(time.Millisecond)
	}
}

func formatInt(n int) string {
	return humanize.Comma(int64(n))
}

func formatFloat(f float64) string {
	switch {
	case f >= 1_000_000:
		return fmt.Sprintf("%.1fM", f/1_000_000)
	case f >= 1_000:
		return fmt.Sprintf("%.1fK", f/1_000)
	default:
		return fmt.Sprintf("%.0f", f)
	}
}

func formatBytes(b uint64) string {
	const (
		kib = 1024
		mib = 1024 * kib
		gib = 1024 * mib
	)

	switch {
	case b >= gib:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(gib))
	case b >= mib:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(mib))
	case b >= kib:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(kib))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func formatCPUDuration(ns uint64) string {
	if ns == 0 {
		return "n/a"
	}

	return roundDuration(time.Duration(ns)).String()
}

func sortedKeys(m map[string]*Result) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	return keys
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	return s[:maxLen-3] + "..."
}

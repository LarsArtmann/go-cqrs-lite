package benchkit

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// durationMarshalers serializes time.Duration as nanoseconds (int64)
// because JSON v2 has no default representation for time.Duration.
var durationMarshalers = json.MarshalFunc(
	func(d time.Duration) ([]byte, error) {
		return []byte(strconv.FormatInt(d.Nanoseconds(), 10)), nil
	},
)

// durationUnmarshalers deserializes time.Duration from nanoseconds (int64),
// enabling JSON round-trip (WriteJSON → json.Unmarshal with jsonOpts).
var durationUnmarshalers = json.UnmarshalFunc(
	func(b []byte, t *time.Duration) error {
		n, err := strconv.ParseInt(string(b), 10, 64)
		if err != nil {
			return fmt.Errorf("parse duration nanoseconds: %w", err)
		}

		*t = time.Duration(n)

		return nil
	},
)

// jsonOpts are the default JSON encoding options: indented output
// with time.Duration serialized/deserialized as nanoseconds.
var jsonOpts = json.JoinOptions(
	jsontext.WithIndent("  "),
	json.WithMarshalers(durationMarshalers),
	json.WithUnmarshalers(durationUnmarshalers),
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

	if r.JourneySamples > 0 {
		fmt.Fprintln(w, "Journey (publish→projection→query):")
		printLatencyLine(w, "  Round trip:", r.JourneyLatency)
		printLatencyLine(w, "  Projection:", r.JourneyProjectionLatency)
		printLatencyLine(w, "  Query leg:", r.JourneyQueryLatency)
		fmt.Fprintln(w)
	}

	if r.QueryHitLatency.Count > 0 {
		fmt.Fprintln(w, "Query Dispatch:")
		printLatencyLine(w, "  Hit:", r.QueryHitLatency)
		printLatencyLine(w, "  Miss:", r.QueryMissLatency)
		printLatencyLine(w, "  Paginated:", r.QueryPaginatedLatency)
		fmt.Fprintln(w)
	}

	if r.SnapshotColdLatency.Count > 0 {
		fmt.Fprintln(w, "Snapshot / Cache:")
		printLatencyLine(w, "  Cold replay:", r.SnapshotColdLatency)
		printLatencyLine(w, "  Snapshot load:", r.SnapshotLoadLatency)
		printLatencyLine(w, "  Cache miss:", r.CacheMissLatency)
		printLatencyLine(w, "  Cache hit:", r.CacheHitLatency)
		fmt.Fprintln(w)
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
		"%-10s %10s %10s %10s %10s %10s %10s %10s %10s",
		"Backend",
		"Raw P50",
		"Raw P99",
		"Write P50",
		"Write P99",
		"Load P50",
		"Load P99",
		"Heap MB",
		"Disk MB",
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
		w, "%-10s %10s %10s %10s %10s %10s %10s %10s %10s\n",
		name,
		roundDuration(r.RawSinkLatency.P50),
		roundDuration(r.RawSinkLatency.P99),
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

// writeJSONAny serializes any value as indented JSON using the standard options.
func writeJSONAny(w io.Writer, v any) error {
	return json.MarshalWrite(w, v, jsonOpts)
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

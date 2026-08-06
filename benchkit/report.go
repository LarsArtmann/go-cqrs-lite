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
		cpuInfo := fmt.Sprintf("CPU=%d", r.Environment.NumCPU)
		if r.Environment.CPUModel != "" {
			cpuInfo = r.Environment.CPUModel
		}

		fmt.Fprintf(w, "Env: %s | %s/%s | %s GOMAXPROCS=%d workers=%d\n\n",
			r.Environment.GoVersion, r.Environment.GOOS, r.Environment.GOARCH,
			cpuInfo, r.Environment.GOMAXPROCS, r.Workers)
	}

	if r.RepeatCount > 1 {
		reliability := "RELIABLE"
		if !r.RepeatIsReliable {
			reliability = "NOISY — increase Repeat for trustworthy comparison"
		}

		fmt.Fprintf(
			w,
			"Repeat:  median of %d runs | CoV=%.1f%% | %s\n",
			r.RepeatCount, r.RepeatCoV*100, reliability,
		)
		fmt.Fprintf(w, "         min: %s/s, max: %s/s, stddev: %s/s\n\n",
			formatFloat(r.RepeatMin), formatFloat(r.RepeatMax),
			formatFloat(r.RepeatStdDev))
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

	if r.ColdReadLatency.Count > 0 {
		printLatencyLine(w, "  Cold (1st pass):", r.ColdReadLatency)
	}

	if r.TailRatio > 0 {
		fmt.Fprintf(w, "  Tail ratio: %.1fx (P99/P50)\n", r.TailRatio)
	}

	if r.WriteTailRatio > 0 {
		fmt.Fprintf(w, "  Write tail: %.1fx (P99/P50)\n", r.WriteTailRatio)
	}

	if r.ReadAllTime > 0 {
		fmt.Fprintf(w, "  ReadAll:  %s\n", roundDuration(r.ReadAllTime))
	}

	if r.ReadFromTime > 0 {
		fmt.Fprintf(w, "  ReadFrom: %s\n", roundDuration(r.ReadFromTime))
	}

	fmt.Fprintln(w)

	if r.LoadFromVersionLatency.Count > 0 {
		fmt.Fprintln(w, "Versioned Reads:")
		printLatencyLine(w, "  LoadFromVersion:", r.LoadFromVersionLatency)
		printLatencyLine(w, "  LoadToVersion:", r.LoadToVersionLatency)
		printLatencyLine(w, "  LoadToTimestamp:", r.LoadToTimestampLatency)
		fmt.Fprintln(w)
	}

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

	if r.MixedWorkload.WriteOps > 0 || r.MixedWorkload.ReadOps > 0 {
		fmt.Fprintln(w, "Mixed Workload (concurrent reads + writes):")
		printLatencyLine(w, "  Write (under read load):", r.MixedWorkload.WriteLatency)
		printLatencyLine(w, "  Read (under write load):", r.MixedWorkload.ReadLatency)
		fmt.Fprintf(w, "  Writers=%d Readers=%d | writes=%s reads=%s",
			r.MixedWorkload.Writers, r.MixedWorkload.Readers,
			formatInt(int(r.MixedWorkload.WriteOps)), formatInt(int(r.MixedWorkload.ReadOps)))

		if r.MixedWorkload.WriteErrors > 0 || r.MixedWorkload.ReadErrors > 0 {
			fmt.Fprintf(w, " | errors: write=%d read=%d",
				r.MixedWorkload.WriteErrors, r.MixedWorkload.ReadErrors)
		}

		fmt.Fprintln(w)
		fmt.Fprintln(w)
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

	if r.MetaEngineApplyLatency.Count > 0 {
		printMetaEngineSection(w, r)
	}

	if r.RecoveryTime > 0 {
		fmt.Fprintf(w, "Recovery: %s (%s events recovered)\n\n",
			roundDuration(r.RecoveryTime), formatInt(r.RecoveredEvents))
	}

	printResourcesSection(w, r)

	if r.IntegrityErrors > 0 {
		fmt.Fprintf(w, "\n⚠ CORRUPTION: %d integrity errors detected!\n", r.IntegrityErrors)
	}

	printSkippedAndWarnings(w, r)
}

func printSkippedAndWarnings(w io.Writer, r *Result) {
	if len(r.SkippedPhases) > 0 {
		fmt.Fprintln(w, "\nSkipped Phases:")

		for _, phase := range r.SkippedPhases {
			fmt.Fprintf(w, "  ⚠ %s\n", phase)
		}
	}

	if len(r.Warnings) > 0 {
		fmt.Fprintln(w, "\nWarnings:")

		for _, msg := range r.Warnings {
			fmt.Fprintf(w, "  ⚠ %s\n", msg)
		}
	}
}

func printMetaEngineSection(w io.Writer, r *Result) {
	fmt.Fprintln(w, "Metaengine:")
	printLatencyLine(w, "  Apply:", r.MetaEngineApplyLatency)
	fmt.Fprintf(w, "  Apply throughput: %s/s (single), %s/s (concurrent)\n",
		formatFloat(r.MetaEngineApplyThroughput), formatFloat(r.MetaEngineApplyConcurrent))
	printLatencyLine(w, "  Query (ExecuteTyped):", r.MetaEngineQueryLatency)
	printLatencyLine(w, "  Scan (filtered):", r.MetaEngineScanLatency)
	printLatencyLine(w, "  Point read:", r.MetaEnginePointReadLatency)

	if r.MetaEngineScanResults > 0 {
		fmt.Fprintf(w, "  Scan results: %d items (status=active)\n", r.MetaEngineScanResults)
	}

	if r.MetaEngineSQLiteScanLatency.Count > 0 {
		fmt.Fprintln(w, "  --- SQLite engine ---")
		printLatencyLine(w, "  Scan (filtered):", r.MetaEngineSQLiteScanLatency)
		printLatencyLine(w, "  Point read:", r.MetaEngineSQLitePointReadLatency)
		fmt.Fprintf(w, "  Apply throughput: %s/s\n",
			formatFloat(r.MetaEngineSQLiteApplyThroughput))
	}

	fmt.Fprintln(w)
}

func printResourcesSection(w io.Writer, r *Result) {
	fmt.Fprintln(w, "Resources:")
	fmt.Fprintf(w, "  Heap:  %s peak\n", formatBytes(r.Memory.After))
	fmt.Fprintf(w, "  Delta: %s\n", formatBytes(r.Memory.Delta))
	fmt.Fprintf(w, "  CPU:   %s\n", formatCPUDuration(r.CPU.Delta))

	if r.GCCount > 0 {
		fmt.Fprintf(w, "  GC:    %d cycles, max pause %s, total %s (%.1f%% of duration)\n",
			r.GCCount, roundDuration(r.GCMaxPause), roundDuration(r.GCTotalPause), r.GCPercent)
	}

	if r.AllocCount > 0 {
		fmt.Fprintf(w, "  Allocs: %s (%s) | %.1f allocs/op, %s/op\n",
			formatInt(int(r.AllocCount)), formatBytes(r.AllocBytes),
			r.AllocsPerOp, formatBytes(uint64(r.BytesPerOp)))
	}

	if r.Disk.DatabaseBytes > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Storage:")
		fmt.Fprintf(w, "  Database: %s\n", formatBytes(uint64(r.Disk.DatabaseBytes)))
		fmt.Fprintf(w, "  Events:   %s\n", formatBytes(uint64(r.Disk.EventBytes)))
		fmt.Fprintf(w, "  Overhead: %.1f%%\n", r.Disk.OverheadPct)

		if r.Disk.WriteAmplification > 0 {
			fmt.Fprintf(w, "  Write amp: %.2fx\n", r.Disk.WriteAmplification)
		}
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

// WriteJSON serializes a result as indented JSON.
func WriteJSON(w io.Writer, r *Result) error {
	return json.MarshalWrite(w, r, jsonOpts)
}

// writeJSONAny serializes any value as indented JSON using the standard options.
func writeJSONAny(w io.Writer, v any) error {
	return json.MarshalWrite(w, v, jsonOpts)
}

package benchkit

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// SuiteManifest records the complete context for a benchmark result:
// the config that produced it, the environment it ran in, and the result.
// This enables reproducibility: a manifest can be archived alongside a
// release to document exactly what was measured and under what conditions.
type SuiteManifest struct {
	SchemaVersion string      `json:"schemaVersion"`
	Config        Config      `json:"config"`
	Environment   Environment `json:"environment"`
	Result        *Result     `json:"result"`
}

// NewManifest creates a SuiteManifest from a Config and Result.
func NewManifest(config Config, result *Result) SuiteManifest {
	return SuiteManifest{
		SchemaVersion: SchemaVersion,
		Config:        config,
		Environment:   result.Environment,
		Result:        result,
	}
}

// WriteManifest serializes a SuiteManifest as indented JSON.
func WriteManifest(w io.Writer, config Config, result *Result) error {
	return writeJSONAny(w, NewManifest(config, result))
}

// metric is one benchstat-compatible measurement extracted from a [Result]:
// the name suffix appended to the benchmark name, the value, and the benchstat
// unit token.
type metric struct {
	suffix string
	unit   string
	value  float64
}

// resultMetrics extracts the benchstat-compatible metric set from a Result in a
// stable phase-grouped order. Zero-valued metrics — phases that were skipped or
// that the backend does not support — are omitted, so a report never claims a
// measurement the run did not make.
//
// Units are benchstat tokens: "ns/op" and "ops/s" describe per-operation costs
// and rates, while plain "ns", "B" and "allocs" describe run totals whose
// per-operation siblings carry the "/op" suffix. Labelling a run total as
// "<unit>/op" made benchstat-comparable reports look like per-op costs and
// invited bogus before/after comparisons.
func resultMetrics(r *Result) []metric {
	all := []metric{
		{"write_throughput", "ops/s", r.WriteThroughput},
		{"rawsink_throughput", "ops/s", r.RawSinkThroughput},
		{"write_p50_ns", "ns/op", float64(r.WriteLatency.P50.Nanoseconds())},
		{"write_p99_ns", "ns/op", float64(r.WriteLatency.P99.Nanoseconds())},
		{"write_max_ns", "ns", float64(r.WriteLatency.P100.Nanoseconds())},
		{"load_p50_ns", "ns/op", float64(r.LoadLatency.P50.Nanoseconds())},
		{"load_p99_ns", "ns/op", float64(r.LoadLatency.P99.Nanoseconds())},
		{"load_max_ns", "ns", float64(r.LoadLatency.P100.Nanoseconds())},
		{"rawsink_p50_ns", "ns/op", float64(r.RawSinkLatency.P50.Nanoseconds())},
		{"rawsink_p99_ns", "ns/op", float64(r.RawSinkLatency.P99.Nanoseconds())},
		{"journey_p50_ns", "ns/op", float64(r.JourneyLatency.P50.Nanoseconds())},
		{"journey_p99_ns", "ns/op", float64(r.JourneyLatency.P99.Nanoseconds())},
		{
			"journey_projection_p99_ns",
			"ns/op",
			float64(r.JourneyProjectionLatency.P99.Nanoseconds()),
		},
		{"journey_query_p99_ns", "ns/op", float64(r.JourneyQueryLatency.P99.Nanoseconds())},
		{"query_hit_p50_ns", "ns/op", float64(r.QueryHitLatency.P50.Nanoseconds())},
		{"query_hit_p99_ns", "ns/op", float64(r.QueryHitLatency.P99.Nanoseconds())},
		{"query_miss_p99_ns", "ns/op", float64(r.QueryMissLatency.P99.Nanoseconds())},
		{"query_paginated_p99_ns", "ns/op", float64(r.QueryPaginatedLatency.P99.Nanoseconds())},
		{"snapshot_cold_p50_ns", "ns/op", float64(r.SnapshotColdLatency.P50.Nanoseconds())},
		{"snapshot_cold_p99_ns", "ns/op", float64(r.SnapshotColdLatency.P99.Nanoseconds())},
		{"snapshot_load_p99_ns", "ns/op", float64(r.SnapshotLoadLatency.P99.Nanoseconds())},
		{"cache_miss_p99_ns", "ns/op", float64(r.CacheMissLatency.P99.Nanoseconds())},
		{"cache_hit_p99_ns", "ns/op", float64(r.CacheHitLatency.P99.Nanoseconds())},
		{"cold_read_p50_ns", "ns/op", float64(r.ColdReadLatency.P50.Nanoseconds())},
		{"cold_read_p99_ns", "ns/op", float64(r.ColdReadLatency.P99.Nanoseconds())},
		{"gc_max_pause_ns", "ns", float64(r.GCMaxPause.Nanoseconds())},
		{"gc_total_pause_ns", "ns", float64(r.GCTotalPause.Nanoseconds())},
		{"alloc_count", "allocs", float64(r.AllocCount)},
		{"allocs_per_op", "allocs/op", r.AllocsPerOp},
		{"bytes_per_op", "B/op", r.BytesPerOp},
		{"gc_percent", "percent", r.GCPercent},
		{"tail_ratio", "ratio", r.TailRatio},
		{"metaengine_scan_p99_ns", "ns/op", float64(r.MetaEngineScanLatency.P99.Nanoseconds())},
		{
			"metaengine_point_read_p99_ns",
			"ns/op",
			float64(r.MetaEnginePointReadLatency.P99.Nanoseconds()),
		},
		{"metaengine_apply_concurrent", "ops/s", r.MetaEngineApplyConcurrent},
		{
			"metaengine_sqlite_scan_p99_ns",
			"ns/op",
			float64(r.MetaEngineSQLiteScanLatency.P99.Nanoseconds()),
		},
		{
			"metaengine_sqlite_point_read_p99_ns",
			"ns/op",
			float64(r.MetaEngineSQLitePointReadLatency.P99.Nanoseconds()),
		},
		{"metaengine_sqlite_apply_throughput", "ops/s", r.MetaEngineSQLiteApplyThroughput},
		{"write_tail_ratio", "ratio", r.WriteTailRatio},
		{"write_amplification", "ratio", r.Disk.WriteAmplification},
		{"heap_bytes", "B", float64(r.Memory.After)},
	}

	out := make([]metric, 0, len(all))

	for _, m := range all {
		if m.value == 0 {
			continue
		}

		out = append(out, m)
	}

	return out
}

// metricValues indexes a Result's metrics by suffix, so a multi-run writer can
// line up the same metric across runs.
func metricValues(r *Result) map[string]float64 {
	values := make(map[string]float64, 40)

	for _, m := range resultMetrics(r) {
		values[m.suffix] = m.value
	}

	return values
}

// benchstatName builds the benchmark identifier benchstat groups by:
// Benchmark<Backend>_<Profile>-<GOMAXPROCS>.
func benchstatName(r *Result) string {
	gomaxprocs := r.Environment.GOMAXPROCS
	if gomaxprocs == 0 {
		gomaxprocs = 1
	}

	return fmt.Sprintf(
		"Benchmark%s_%s-%d",
		sanitizeBenchName(r.Backend),
		sanitizeBenchName(r.Profile),
		gomaxprocs,
	)
}

// WriteBenchstat emits results in a benchstat-compatible text format.
// Each metric becomes a separate line with the standard Go benchmark
// naming convention: BenchmarkName-N <value> <unit>.
//
// A single run yields exactly one sample per metric, which is enough for
// benchstat to compare point estimates but not enough for it to report a
// confidence interval. For statistically meaningful comparisons, run with
// repeats and use [WriteBenchstatRepeated] (the cqrs-bench `--repeat N`
// path), or compare a baseline captured from an earlier revision:
//
//	cqrs-bench run --backend memory --format benchstat > old.txt
//	# ... make changes ...
//	cqrs-bench run --backend memory --format benchstat > new.txt
//	benchstat old.txt new.txt
func WriteBenchstat(w io.Writer, r *Result) {
	if r == nil {
		return
	}

	writeBenchstatLines(w, benchstatName(r), r, []*Result{r})
}

// WriteBenchstatRepeated emits one benchstat sample per repeat run for every
// metric the median run recorded. benchstat needs n>1 samples per benchmark to
// compute a mean, a confidence interval, and a p-value, so a `--repeat N` run
// through this writer is what turns `benchstat old.txt new.txt` from a point
// comparison into a statistical one.
//
// Runs that did not record a metric contribute no sample to it (rather than a
// misleading zero), so a phase that only fired in some runs shrinks that
// metric's sample count instead of inflating its spread.
func WriteBenchstatRepeated(w io.Writer, repeated *RepeatedResult) {
	if repeated == nil || repeated.Median == nil {
		return
	}

	if len(repeated.Runs) <= 1 {
		WriteBenchstat(w, repeated.Median)

		return
	}

	writeBenchstatLines(w, benchstatName(repeated.Median), repeated.Median, repeated.Runs)
}

// writeBenchstatLines writes reference's metrics for every run, one line per
// run, so a metric recorded in all N runs contributes N samples to benchstat.
func writeBenchstatLines(w io.Writer, name string, reference *Result, runs []*Result) {
	values := make([]map[string]float64, len(runs))
	for i, run := range runs {
		values[i] = metricValues(run)
	}

	for _, ref := range resultMetrics(reference) {
		for _, runValues := range values {
			value, ok := runValues[ref.suffix]
			if !ok {
				continue
			}

			fmt.Fprintf(w, "%s_%s\t1\t%.0f %s\n", name, ref.suffix, value, ref.unit)
		}
	}
}

// ExpectedJSONFields returns the canonical set of top-level JSON field names
// in a Result. This is used by schema-stability tests to verify that the
// JSON shape hasn't changed across versions.
func ExpectedJSONFields() []string {
	return []string{
		"backend",
		"profile",
		"timestamp",
		"duration",
		"schemaVersion",
		"environment",
		"workers",
		"streams",
		"eventsPerStream",
		"totalEvents",
		"payloadBytesPerEvent",
		"rawSinkLatency",
		"writeLatency",
		"writeThroughput",
		"loadLatency",
		"coldReadLatency",
		"readAllTime",
		"readFromTime",
		"readModelGet",
		"readModelSet",
		"projectionLag",
		"projectionEvents",
		"journeyLatency",
		"journeyProjectionLatency",
		"journeyQueryLatency",
		"queryHitLatency",
		"queryMissLatency",
		"queryPaginatedLatency",
		"snapshotColdLatency",
		"snapshotLoadLatency",
		"cacheMissLatency",
		"cacheHitLatency",
		"gcCount",
		"gcTotalPause",
		"gcMaxPause",
		"gcMeanPause",
		"allocCount",
		"allocBytes",
		"allocsPerOp",
		"bytesPerOp",
		"gcPercent",
		"tailRatio",
		"metaEngineScanLatency",
		"metaEnginePointReadLatency",
		"metaEngineApplyConcurrent",
		"metaEngineScanResults",
		"metaEngineSQLiteApplyThroughput",
		"metaEngineSQLiteScanLatency",
		"metaEngineSQLitePointReadLatency",
		"writeTailRatio",
		"memory",
		"cpu",
		"disk",
		"codec",
	}
}

// VerifyJSONFields checks that the marshaled JSON contains all expected
// top-level field names. Returns the list of missing fields (empty if all
// present). This is the programmatic equivalent of a golden-file test:
// if the Result struct changes shape, this will catch it.
func VerifyJSONFields(jsonKeys []string) []string {
	expected := ExpectedJSONFields()

	lookup := make(map[string]bool, len(jsonKeys))
	for _, k := range jsonKeys {
		lookup[k] = true
	}

	var missing []string

	for _, e := range expected {
		if !lookup[e] {
			missing = append(missing, e)
		}
	}

	sort.Strings(missing)

	return missing
}

func sanitizeBenchName(s string) string {
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, ":", "_")

	return s
}

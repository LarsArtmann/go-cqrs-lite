package benchkit

import (
	"bytes"
	"encoding/json/v2"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestLatencyCollector_TracksExactMin(t *testing.T) {
	t.Parallel()

	lc := NewLatencyCollector(0)

	for _, d := range []time.Duration{5 * time.Microsecond, time.Microsecond, 3 * time.Microsecond} {
		lc.Record(d)
	}

	stats := lc.Stats()

	if stats.Min != time.Microsecond {
		t.Errorf("Min = %v, want the exact fastest sample 1µs", stats.Min)
	}

	if stats.P100 != 5*time.Microsecond {
		t.Errorf("P100 = %v, want the exact slowest sample 5µs", stats.P100)
	}
}

func TestLatencyCollector_MinZeroWithoutSamples(t *testing.T) {
	t.Parallel()

	if stats := NewLatencyCollector(0).Stats(); stats != (LatencyStats{}) {
		t.Errorf("empty collector Stats = %+v, want zero value", stats)
	}
}

func TestLatencyCollector_InterpolatedPercentilesSmallN(t *testing.T) {
	t.Parallel()

	samples := []time.Duration{
		1 * time.Millisecond,
		2 * time.Millisecond,
		3 * time.Millisecond,
		4 * time.Millisecond,
		5 * time.Millisecond,
	}

	lc := NewLatencyCollectorWithOptions(0, WithInterpolatedPercentiles())
	for _, d := range samples {
		lc.Record(d)
	}

	stats := lc.Stats()

	// P50 rank = 0.5*4 = 2.0 — lands exactly on a sample either way.
	if stats.P50 != 3*time.Millisecond {
		t.Errorf("interpolated P50 = %v, want 3ms", stats.P50)
	}

	// Nearest-rank P90 would collapse onto the maximum (ceil(0.9*5)=5th
	// sample = 5ms); interpolation reads 4ms + 0.6*1ms.
	if want := 4600 * time.Microsecond; stats.P90 != want {
		t.Errorf(
			"interpolated P90 = %v, want %v (between the 4th and 5th samples)",
			stats.P90,
			want,
		)
	}

	// P99 must stay below the true max instead of equalling it — the whole
	// point for small-n runs where nearest-rank P99 IS the max.
	if stats.P99 >= stats.P100 {
		t.Errorf(
			"interpolated P99 = %v, want strictly below the exact max %v",
			stats.P99,
			stats.P100,
		)
	}

	// Default collector keeps nearest-rank: P99 equals the max for n=5.
	def := NewLatencyCollector(0)
	for _, d := range samples {
		def.Record(d)
	}

	if def.Stats().P99 != 5*time.Millisecond {
		t.Errorf("default nearest-rank P99 = %v, want the 5th sample 5ms", def.Stats().P99)
	}
}

func TestPercentileInterpolated_EdgeCases(t *testing.T) {
	t.Parallel()

	if got := percentileInterpolated(nil, 50); got != 0 {
		t.Errorf("percentileInterpolated(nil) = %v, want 0", got)
	}

	single := []time.Duration{7 * time.Millisecond}
	if got := percentileInterpolated(single, 99); got != 7*time.Millisecond {
		t.Errorf("percentileInterpolated(single) = %v, want the only sample", got)
	}
}

func TestResultNoisyMetricCount(t *testing.T) {
	t.Parallel()

	r := &Result{MetricVariation: []MetricVariation{
		{Name: "a", Reliable: true},
		{Name: "b", Reliable: false},
		{Name: "c", Reliable: false},
	}}

	if got := r.NoisyMetricCount(); got != 2 {
		t.Errorf("NoisyMetricCount = %d, want 2", got)
	}

	if got := (&Result{}).NoisyMetricCount(); got != 0 {
		t.Errorf("single-run NoisyMetricCount = %d, want 0", got)
	}
}

func TestMetricNames_StableUniverse(t *testing.T) {
	t.Parallel()

	names := MetricNames()

	if len(names) < 40 {
		t.Fatalf("MetricNames len = %d, want the full metric universe (>= 40)", len(names))
	}

	if names[0] != "write_throughput" {
		t.Errorf("first metric = %q, want write_throughput (report order)", names[0])
	}

	seen := make(map[string]bool, len(names))
	for _, n := range names {
		if seen[n] {
			t.Errorf("duplicate metric name %q", n)
		}

		seen[n] = true
	}

	for _, want := range []string{"write_p50_ns", "load_p99_ns", "write_max_ns", "heap_bytes"} {
		if !slices.Contains(names, want) {
			t.Errorf("MetricNames missing %q", want)
		}
	}

	// A fresh copy per call: mutating one result must not corrupt the next.
	names[0] = "mutated"
	if MetricNames()[0] != "write_throughput" {
		t.Error("MetricNames must return a fresh slice each call")
	}
}

func TestWriteRepeatedJSON_RoundTrip(t *testing.T) {
	t.Parallel()

	rr := &RepeatedResult{
		Median: &Result{
			Backend:         "memory",
			WriteThroughput: 100,
			RepeatCount:     2,
			MetricVariation: []MetricVariation{
				{Name: "write_throughput", Unit: "ops/s", Mean: 105, CoV: 0.05, Reliable: true},
			},
		},
		Runs: []*Result{
			{Backend: "memory", WriteThroughput: 100},
			{Backend: "memory", WriteThroughput: 110},
		},
	}

	var buf bytes.Buffer

	if err := rr.WriteRepeatedJSON(&buf); err != nil {
		t.Fatalf("WriteRepeatedJSON: %v", err)
	}

	var decoded RepeatedResult

	if err := json.Unmarshal(buf.Bytes(), &decoded, jsonOpts); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(decoded.Runs) != 2 {
		t.Errorf("decoded Runs len = %d, want 2", len(decoded.Runs))
	}

	if decoded.Median == nil || decoded.Median.RepeatCount != 2 {
		t.Errorf("decoded Median missing or RepeatCount = %d, want 2", decoded.Median.RepeatCount)
	}

	if len(decoded.Median.MetricVariation) != 1 ||
		decoded.Median.MetricVariation[0].Name != "write_throughput" {
		t.Error("decoded MetricVariation lost")
	}

	var nilRR *RepeatedResult

	if err := nilRR.WriteRepeatedJSON(&buf); err != nil {
		t.Errorf("nil receiver must write nothing without error, got %v", err)
	}
}

func TestManifestRepeated_CarriesRuns(t *testing.T) {
	t.Parallel()

	runs := []*Result{
		{Backend: "sqlite", WriteThroughput: 100},
		{Backend: "sqlite", WriteThroughput: 110},
	}

	rr := &RepeatedResult{Median: runs[1], Runs: runs}

	manifest := NewManifestRepeated(Config{Backend: "sqlite"}, rr)
	if manifest.Result != runs[1] {
		t.Error("manifest Result must be the median run")
	}

	if len(manifest.Runs) != 2 {
		t.Fatalf("manifest Runs len = %d, want 2", len(manifest.Runs))
	}

	var buf bytes.Buffer

	if err := WriteManifestRepeated(&buf, Config{Backend: "sqlite"}, rr); err != nil {
		t.Fatalf("write: %v", err)
	}

	if !strings.Contains(buf.String(), `"runs"`) {
		t.Error("repeated manifest JSON must contain the runs[] field")
	}

	// Single-run manifest: runs[] omitted (opt-in only).
	buf.Reset()

	if err := WriteManifest(&buf, Config{}, &Result{Backend: "sqlite"}); err != nil {
		t.Fatalf("WriteManifest: %v", err)
	}

	if strings.Contains(buf.String(), `"runs"`) {
		t.Error("single-run manifest must not carry runs[]")
	}
}

func TestAuditZeroValues_WarnsOnDisagreement(t *testing.T) {
	t.Parallel()

	r := &runner{result: Result{
		WriteLatency:    LatencyStats{},
		WriteThroughput: 500,
		RawSinkLatency:  LatencyStats{Count: 10},
	}}
	r.auditZeroValues()

	if len(r.result.Warnings) != 2 {
		t.Fatalf("warnings = %v, want one per disagreeing pair", r.result.Warnings)
	}

	if !strings.Contains(r.result.Warnings[0], "write phase") ||
		!strings.Contains(r.result.Warnings[0], "zero latency samples") {
		t.Errorf("throughput-without-samples warning missing: %q", r.result.Warnings[0])
	}

	if !strings.Contains(r.result.Warnings[1], "raw sink phase") ||
		!strings.Contains(r.result.Warnings[1], "zero throughput") {
		t.Errorf("samples-without-throughput warning missing: %q", r.result.Warnings[1])
	}
}

func TestAuditZeroValues_ConsistentPairSilent(t *testing.T) {
	t.Parallel()

	r := &runner{result: Result{
		WriteLatency:    LatencyStats{Count: 100},
		WriteThroughput: 500,
	}}
	r.auditZeroValues()

	if len(r.result.Warnings) != 0 {
		t.Errorf("consistent throughput pair must not warn, got %v", r.result.Warnings)
	}
}

func TestComputeSoakTrends_CrossIterationCoV(t *testing.T) {
	t.Parallel()

	r := &SoakResult{Samples: []SoakSample{
		{Throughput: 100, WriteP99: 10 * time.Millisecond},
		{Throughput: 200, WriteP99: 30 * time.Millisecond},
	}}
	computeSoakTrends(r)

	if !nearlyEqual(r.ThroughputCoV, 1.0/3.0) {
		t.Errorf("ThroughputCoV = %v, want 1/3 (stddev 50 / mean 150)", r.ThroughputCoV)
	}

	if !nearlyEqual(r.WriteP99CoV, 0.5) {
		t.Errorf("WriteP99CoV = %v, want 0.5 (stddev 10ms / mean 20ms)", r.WriteP99CoV)
	}
}

func TestComputeSoakTrends_CoVZeroForSingleSample(t *testing.T) {
	t.Parallel()

	r := &SoakResult{Samples: []SoakSample{{Throughput: 100}}}
	computeSoakTrends(r)

	if r.ThroughputCoV != 0 {
		t.Errorf("single-sample ThroughputCoV = %v, want 0 (not computed)", r.ThroughputCoV)
	}
}

func TestOversubscribed_ThresholdConfigurable(t *testing.T) {
	t.Parallel()

	// Default: the oversubscription line is load == NumCPU.
	r := &runner{result: Result{Environment: Environment{NumCPU: 4}}}
	if !r.oversubscribed(4.5) {
		t.Error("load above NumCPU must be oversubscribed by default")
	}

	if r.oversubscribed(3.9) {
		t.Error("load below NumCPU must not be oversubscribed by default")
	}

	// Config knob: a box that always runs hot can raise the line to 2x CPUs.
	r.config = Config{LoadWarnThreshold: 2.0}
	if r.oversubscribed(7.0) {
		t.Error("load below 2x NumCPU must not warn with LoadWarnThreshold 2.0")
	}

	if !r.oversubscribed(9.0) {
		t.Error("load above 2x NumCPU must warn with LoadWarnThreshold 2.0")
	}
}

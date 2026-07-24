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

// WriteBenchstat emits results in a benchstat-compatible text format.
// Each metric becomes a separate line with the standard Go benchmark
// naming convention: BenchmarkName-N <value> <unit>.
//
// The output can be piped to `benchstat` for statistical comparison:
//
//	cqrs-bench run --backend memory --format benchstat > old.txt
//	# ... make changes ...
//	cqrs-bench run --backend memory --format benchstat > new.txt
//	benchstat old.txt new.txt
func WriteBenchstat(w io.Writer, r *Result) {
	if r == nil {
		return
	}

	gomaxprocs := r.Environment.GOMAXPROCS
	if gomaxprocs == 0 {
		gomaxprocs = 1
	}

	name := fmt.Sprintf(
		"Benchmark%s_%s-%d",
		sanitizeBenchName(r.Backend),
		sanitizeBenchName(r.Profile),
		gomaxprocs,
	)

	lines := []struct {
		suffix string
		value  float64
		unit   string
	}{
		{"write_throughput", r.WriteThroughput, "ops/s"},
		{"rawsink_throughput", r.RawSinkThroughput, "ops/s"},
		{"write_p50_ns", float64(r.WriteLatency.P50.Nanoseconds()), "ns/op"},
		{"write_p99_ns", float64(r.WriteLatency.P99.Nanoseconds()), "ns/op"},
		{"load_p50_ns", float64(r.LoadLatency.P50.Nanoseconds()), "ns/op"},
		{"load_p99_ns", float64(r.LoadLatency.P99.Nanoseconds()), "ns/op"},
		{"rawsink_p50_ns", float64(r.RawSinkLatency.P50.Nanoseconds()), "ns/op"},
		{"rawsink_p99_ns", float64(r.RawSinkLatency.P99.Nanoseconds()), "ns/op"},
		{"heap_bytes", float64(r.Memory.After), "B/op"},
	}

	for _, l := range lines {
		if l.value == 0 {
			continue
		}

		fmt.Fprintf(w, "%s_%s\t1\t%.0f %s\n", name, l.suffix, l.value, l.unit)
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
		"aggregates",
		"eventsPerAggregate",
		"totalEvents",
		"payloadBytesPerEvent",
		"writeLatency",
		"writeThroughput",
		"loadLatency",
		"readAllTime",
		"readFromTime",
		"memory",
		"cpu",
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

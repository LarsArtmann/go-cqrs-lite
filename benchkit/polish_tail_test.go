package benchkit

import (
	"bytes"
	"context"
	"os"
	"math"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/stack/memory/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
)

// TestPrintLatencyLine_ShowsExactMinAndMax pins the report line format: the
// exact Min (fastest observed) renders next to Max so the scheduler-overhead
// story (Mean/Min) is visible without JSON tooling.
func TestPrintLatencyLine_ShowsExactMinAndMax(t *testing.T) {
	t.Parallel()

	stats := LatencyStats{
		Count: 10,
		P50:   100 * time.Microsecond,
		P95:   200 * time.Microsecond,
		P99:   400 * time.Microsecond,
		P100:  900 * time.Microsecond,
		Min:   40 * time.Microsecond,
	}

	var buf bytes.Buffer

	printLatencyLine(&buf, "  Latency:", stats)

	out := buf.String()
	for _, want := range []string{"P50=", "P95=", "P99=", "Max=", "Min="} {
		if !strings.Contains(out, want) {
			t.Errorf("latency line missing %q: %q", want, out)
		}
	}
}

// TestPrintReport_Load1InEnvLine: the env line carries the start load average
// whenever one was recorded, and stays silent for zero (unknown) loads.
func TestPrintReport_Load1InEnvLine(t *testing.T) {
	t.Parallel()

	withLoad := &Result{
		Backend: "memory",
		Profile: "dev",
		Environment: Environment{
			GoVersion:  "go1.27",
			NumCPU:     4,
			GOMAXPROCS: 4,
			GOOS:       "linux",
			GOARCH:     "amd64",
			LoadAvg1:   3.5,
		},
	}

	var buf bytes.Buffer

	PrintReport(&buf, withLoad)
	if !strings.Contains(buf.String(), "Load1=3.5") {
		t.Errorf("env line missing Load1=3.5: %q", buf.String())
	}

	withoutLoad := *withLoad
	withoutLoad.Environment.LoadAvg1 = 0

	buf.Reset()
	PrintReport(&buf, &withoutLoad)
	if strings.Contains(buf.String(), "Load1=") {
		t.Errorf("env line shows Load1 for an unrecorded load: %q", buf.String())
	}
}

// TestWriteTailRatio_UsesTrueMax pins the write-tail semantics: the ratio is
// the EXACT maximum (write_max_ns) over P50, not the reservoir P99 — a tame
// P99 must not hide a max spike.
func TestWriteTailRatio_UsesTrueMax(t *testing.T) {
	t.Parallel()

	result := mustRun(t, Config{
		Profile:     ProfileDev,
		PayloadSize: 64,
		Backend:     "memory",
	}, func() (*stack.Bundle, error) { return memory.New() })

	if result.WriteLatency.P50 <= 0 {
		t.Fatal("write P50 not recorded; cannot verify tail ratio")
	}

	want := float64(result.WriteLatency.P100) / float64(result.WriteLatency.P50)
	if math.Abs(result.WriteTailRatio-want) > 0.01 {
		t.Errorf("WriteTailRatio = %.3f, want Max/P50 = %.3f (P100=%s P50=%s)",
			result.WriteTailRatio, want, result.WriteLatency.P100, result.WriteLatency.P50)
	}
}

// TestHeadlineMetricNames_MatchesGateDefault pins the shared headline set:
// cqrs-bench --strict and scripts/benchmark-regression.sh key on exactly
// these names (write_p99_ns deliberately absent — demoted 2026-09-20).
func TestHeadlineMetricNames_MatchesGateDefault(t *testing.T) {
	t.Parallel()

	got := HeadlineMetricNames()

	want := []string{MetricWriteThroughput, MetricWriteP50NS, MetricLoadP50NS}
	if !slices.Equal(got, want) {
		t.Errorf("HeadlineMetricNames() = %v, want %v", got, want)
	}

	got[0] = "mutated"
	if HeadlineMetricNames()[0] == "mutated" {
		t.Error("HeadlineMetricNames must return a fresh copy per call")
	}

	universe := MetricNames()
	for _, name := range want {
		if !slices.Contains(universe, name) {
			t.Errorf("headline metric %q missing from MetricNames() universe", name)
		}
	}
}

// TestRunRepeated_PerRepeatProgress: with a progress writer configured, each
// repetition announces itself so a long --repeat N run cannot look stuck.
func TestRunRepeated_PerRepeatProgress(t *testing.T) {
	t.Parallel()

	var progress bytes.Buffer

	_, err := RunRepeated(context.Background(), Config{
		Profile:        ProfileDev,
		PayloadSize:    64,
		Repeat:         2,
		ProgressWriter: &progress,
	}, func() (*stack.Bundle, error) { return memory.New() })
	if err != nil {
		t.Fatalf("RunRepeated: %v", err)
	}

	out := progress.String()
	for _, want := range []string{"repeat 1/2", "repeat 2/2"} {
		if !strings.Contains(out, want) {
			t.Errorf("progress output missing %q: %q", want, out)
		}
	}
}

// TestNewCollector_ReservoirSizeConfig: Config.ReservoirSize feeds every
// phase collector when the call site does not pin its own size.
func TestNewCollector_ReservoirSizeConfig(t *testing.T) {
	t.Parallel()

	fromConfig := (&runner{config: Config{ReservoirSize: 42}}).newCollector(0)
	if fromConfig.maxLen != 42 {
		t.Errorf("newCollector(0) with ReservoirSize=42: maxLen = %d, want 42", fromConfig.maxLen)
	}

	defaultLen := (&runner{config: Config{}}).newCollector(0)
	if defaultLen.maxLen != defaultReservoirSize {
		t.Errorf("newCollector(0) with no config: maxLen = %d, want default %d",
			defaultLen.maxLen, defaultReservoirSize)
	}

	explicit := (&runner{config: Config{ReservoirSize: 42}}).newCollector(7)
	if explicit.maxLen != 7 {
		t.Errorf("explicit maxLen lost: got %d, want 7", explicit.maxLen)
	}
}

// TestConfigValidate_RejectsNegativeReservoirSize guards the config knob.
func TestConfigValidate_RejectsNegativeReservoirSize(t *testing.T) {
	t.Parallel()

	cfg := Config{Profile: ProfileDev, ReservoirSize: -1}
	if err := cfg.validate(); err == nil {
		t.Error("validate() accepted a negative ReservoirSize; want ErrInvalidConfig")
	}
}

// TestPrintSweep_CoVColumnOnlyWhenRepeated: the CoV column appears exactly
// when a data point carries repeat dispersion; single-run sweeps keep the
// legacy column set.
func TestPrintSweep_CoVColumnOnlyWhenRepeated(t *testing.T) {
	t.Parallel()

	single := []SweepResult{
		{Parameter: "workers", Value: 1, Result: &Result{WriteLatency: LatencyStats{
			Count: 5, P50: time.Millisecond, P99: 2 * time.Millisecond,
		}}},
	}

	var buf bytes.Buffer

	PrintSweep(&buf, single)
	if strings.Contains(buf.String(), "CoV") {
		t.Errorf("single-run sweep shows CoV column: %q", buf.String())
	}

	repeated := []SweepResult{
		{Parameter: "workers", Value: 1, Result: &Result{
			RepeatCount: 3, RepeatCoV: 0.152,
			WriteLatency: LatencyStats{Count: 5, P50: time.Millisecond},
		}},
	}

	buf.Reset()
	PrintSweep(&buf, repeated)

	out := buf.String()
	if !strings.Contains(out, "CoV") {
		t.Errorf("repeated sweep missing CoV column: %q", out)
	}

	if !strings.Contains(out, "15.2%") {
		t.Errorf("CoV column does not render the point's 15.2%% dispersion: %q", out)
	}
}

// TestHeadlineMetricNames_MatchesGateScriptLiteral pins the OTHER half of
// the split-brain (2026-09-25): the library list is pinned against itself
// above, but scripts/benchmark-regression.sh's default NOISE_HEADLINE is a
// shell literal — this test fails when the script's list and the library's
// list drift apart (the equality was comment-enforced until now).
func TestHeadlineMetricNames_MatchesGateScriptLiteral(t *testing.T) {
	t.Parallel()

	script, err := os.ReadFile("../scripts/benchmark-regression.sh")
	if err != nil {
		t.Skipf("gate script not readable from test context: %v", err)
	}

	for _, line := range strings.Split(string(script), "\n") {
		if !strings.HasPrefix(line, "NOISE_HEADLINE=") || strings.Contains(line, "${") {
			continue
		}

		want := strings.Trim(strings.TrimPrefix(line, "NOISE_HEADLINE="), `"`)
		got := strings.Join(HeadlineMetricNames(), " ")

		if want != got {
			t.Fatalf("NOISE_HEADLINE drift: gate gates %q, benchkit headlines %q", want, got)
		}

		return
	}

	t.Fatal("NOISE_HEADLINE default assignment not found in benchmark-regression.sh")
}

// TestRunSuiteRepeated_ReportsCoVMetrics pins the suite helper that had zero
// direct coverage (polish-tail (a), 2026-09-25): driven through
// testing.Benchmark so the real *testing.B path executes — Repeat >= 2 must
// surface <metric>_cov% custom metrics, Repeat < 2 must delegate to the
// single-run suite without them.
func TestRunSuiteRepeated_ReportsCoVMetrics(t *testing.T) {
	t.Parallel()

	factory := func() (*stack.Bundle, error) {
		return memory.New()
	}

	br := testing.Benchmark(func(b *testing.B) {
		RunSuiteRepeated(b, Config{
			Profile:     ProfileDev,
			PayloadSize: 64,
			Warmup:      1,
			Repeat:      2,
		}, factory)
	})

	if !strings.Contains(br.String(), "write_throughput_cov%") {
		t.Fatalf("RunSuiteRepeated must report <metric>_cov%% custom metrics, got: %q", br.String())
	}
}

// TestRunSuiteRepeated_DelegatesBelowTwoRepeats pins the delegation edge.
func TestRunSuiteRepeated_DelegatesBelowTwoRepeats(t *testing.T) {
	t.Parallel()

	factory := func() (*stack.Bundle, error) {
		return memory.New()
	}

	br := testing.Benchmark(func(b *testing.B) {
		RunSuiteRepeated(b, Config{
			Profile:     ProfileDev,
			PayloadSize: 64,
			Warmup:      1,
			Repeat:      1,
		}, factory)
	})

	if strings.Contains(br.String(), "_cov%") {
		t.Fatalf("single-run delegation must not report CoV metrics, got: %q", br.String())
	}
}

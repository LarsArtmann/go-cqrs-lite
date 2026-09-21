package main

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/benchkit/v4"
)

// TestStrictNoiseGate pins --strict's noise semantics: only HEADLINE metrics
// fail the run (the gate set shared with benchmark-regression.sh), stable
// runs and single runs pass, and the message names the offender.
func TestStrictNoiseGate(t *testing.T) {
	t.Parallel()

	repeated := func(variations ...benchkit.MetricVariation) *benchkit.RepeatedResult {
		return &benchkit.RepeatedResult{
			Median: &benchkit.Result{MetricVariation: variations},
			Runs:   make([]*benchkit.Result, 3),
		}
	}

	cases := []struct {
		name    string
		strict  bool
		repeated *benchkit.RepeatedResult
		wantMsg bool
	}{
		{
			name:    "noisy headline metric fails",
			strict:  true,
			repeated: repeated(benchkit.MetricVariation{
				Name: benchkit.MetricWriteThroughput, CoV: 0.14, Reliable: false,
			}),
			wantMsg: true,
		},
		{
			name:    "noisy non-headline metric passes",
			strict:  true,
			repeated: repeated(benchkit.MetricVariation{
				Name: "gc_total_pause_ns", CoV: 0.40, Reliable: false,
			}),
			wantMsg: false,
		},
		{
			name:    "stable headline passes",
			strict:  true,
			repeated: repeated(benchkit.MetricVariation{
				Name: benchkit.MetricWriteP50NS, CoV: 0.03, Reliable: true,
			}),
			wantMsg: false,
		},
		{name: "strict off passes", strict: false, repeated: repeated(), wantMsg: false},
		{name: "single run passes", strict: true, repeated: nil, wantMsg: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			msg := strictNoiseGate(tc.strict, tc.repeated)
			if (msg != "") != tc.wantMsg {
				t.Errorf("strictNoiseGate message = %q, wantMsg = %v", msg, tc.wantMsg)
			}

			if tc.wantMsg && !strings.Contains(msg, benchkit.MetricWriteThroughput) &&
				!strings.Contains(msg, benchkit.MetricLoadP50NS) &&
				!strings.Contains(msg, benchkit.MetricWriteP50NS) {
				t.Errorf("failure message does not name a headline metric: %q", msg)
			}
		})
	}
}

// TestBuildRunSummaryTable_Load1AndWriteMin: the summary table shows the
// start load average when recorded and the exact write Min next to P50/P99.
func TestBuildRunSummaryTable_Load1AndWriteMin(t *testing.T) {
	t.Parallel()

	r := &benchkit.Result{
		Backend: "memory",
		Profile: "dev",
		WriteLatency: benchkit.LatencyStats{
			Count: 5, P50: time.Millisecond, P99: 2 * time.Millisecond, Min: 100 * time.Microsecond,
		},
		Environment: benchkit.Environment{LoadAvg1: 4.2},
	}

	var labels []string

	for _, row := range buildRunSummaryTable(r).Rows {
		labels = append(labels, row[0])
	}

	if !slices.Contains(labels, "Load1 (start)") {
		t.Errorf("summary table missing 'Load1 (start)' row; got %v", labels)
	}

	if !slices.Contains(labels, "Write Min") {
		t.Errorf("summary table missing 'Write Min' row; got %v", labels)
	}

	noLoad := *r
	noLoad.Environment.LoadAvg1 = 0

	labels = labels[:0]
	for _, row := range buildRunSummaryTable(&noLoad).Rows {
		labels = append(labels, row[0])
	}

	if slices.Contains(labels, "Load1 (start)") {
		t.Errorf("summary table shows Load1 for an unrecorded load; got %v", labels)
	}
}

// TestAddVariationRows_HeadlineCoV: repeated runs get one CoV row per
// headline metric — the rows csv/tsv exports carry as variation columns.
func TestAddVariationRows_HeadlineCoV(t *testing.T) {
	t.Parallel()

	r := &benchkit.Result{
		RepeatCount: 3,
		RepeatCoV:   0.04,
		MetricVariation: []benchkit.MetricVariation{
			{Name: benchkit.MetricWriteThroughput, CoV: 0.04, Reliable: true},
			{Name: benchkit.MetricWriteP50NS, CoV: 0.21, Reliable: false},
			{Name: benchkit.MetricLoadP50NS, CoV: 0.02, Reliable: true},
			{Name: "gc_total_pause_ns", CoV: 0.9, Reliable: false},
		},
	}

	table := buildRunSummaryTable(r)

	var got []string

	for _, row := range table.Rows {
		if strings.HasSuffix(row[0], " CoV") {
			got = append(got, row[0]+"="+row[1])
		}
	}

	want := []string{
		benchkit.MetricWriteThroughput + " CoV=4.0% (stable)",
		benchkit.MetricWriteP50NS + " CoV=21.0% (NOISY)",
		benchkit.MetricLoadP50NS + " CoV=2.0% (stable)",
	}

	if !slices.Equal(got, want) {
		t.Errorf("headline CoV rows = %v, want %v", got, want)
	}
}

// TestBuildSweepTable_CoVColumnWhenRepeated: the sweep table grows a CoV %
// column exactly when a data point carries repeat dispersion.
func TestBuildSweepTable_CoVColumnWhenRepeated(t *testing.T) {
	t.Parallel()

	single := []benchkit.SweepResult{
		{Parameter: "workers", Value: 4, Result: &benchkit.Result{
			WriteLatency: benchkit.LatencyStats{Count: 5, P50: time.Millisecond},
		}},
	}

	if got := len(buildSweepTable(single).Headers); got != 8 {
		t.Errorf("single-run sweep headers = %d, want 8 (no CoV column)", got)
	}

	repeated := []benchkit.SweepResult{
		{Parameter: "workers", Value: 4, Result: &benchkit.Result{
			RepeatCount: 3, RepeatCoV: 0.123,
			WriteLatency: benchkit.LatencyStats{Count: 5, P50: time.Millisecond},
		}},
	}

	table := buildSweepTable(repeated)

	if got := len(table.Headers); got != 9 {
		t.Fatalf("repeated sweep headers = %d, want 9 (with CoV column): %v", got, table.Headers)
	}

	if table.Headers[8] != "CoV %" {
		t.Errorf("last header = %q, want 'CoV %%'", table.Headers[8])
	}

	if !strings.Contains(table.Rows[0][8], "12.3%") {
		t.Errorf("CoV cell = %q, want 12.3%%", table.Rows[0][8])
	}
}

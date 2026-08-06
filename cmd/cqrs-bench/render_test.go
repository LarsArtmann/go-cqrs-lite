package main

import (
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/benchkit/v4"
)

func TestParsePayloadSizes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    []int
		wantErr bool
		errMsg  string
	}{
		{name: "empty", input: "", want: nil},
		{name: "whitespace only", input: "  ", want: nil},
		{name: "valid pair", input: "64,256", want: []int{64, 256}},
		{name: "valid triple with spaces", input: " 64, 256 , 4096 ", want: []int{64, 256, 4096}},
		{name: "single size errors", input: "128", wantErr: true, errMsg: "at least 2"},
		{name: "zero size errors", input: "0,64", wantErr: true, errMsg: "must be > 0"},
		{name: "negative size errors", input: "-1,64", wantErr: true, errMsg: "must be > 0"},
		{name: "non-integer errors", input: "abc,64", wantErr: true, errMsg: "invalid size"},
		{
			name:    "flag-like value (--profile)",
			input:   "--profile",
			wantErr: true,
			errMsg:  "looks like a flag name",
		},
		{
			name:    "flag-like value (--backends)",
			input:   "--backends",
			wantErr: true,
			errMsg:  "looks like a flag name",
		},
		{
			name:    "short flag-like value (-p)",
			input:   "-p",
			wantErr: true,
			errMsg:  "looks like a flag name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parsePayloadSizes(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errMsg)
				}

				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errMsg)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(got) != len(tt.want) {
				t.Fatalf("got %d sizes, want %d", len(got), len(tt.want))
			}

			for i, v := range got {
				if v != tt.want[i] {
					t.Errorf("size[%d] = %d, want %d", i, v, tt.want[i])
				}
			}
		})
	}
}

func TestLooksLikeFlag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  bool
	}{
		{"", false},
		{"64", false},
		{"64,256", false},
		{"--profile", true},
		{"--backends", true},
		{"-p", true},
		{"-1", false},   // negative number
		{"-1.5", false}, // negative float
		{"-", false},    // just a dash
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			if got := looksLikeFlag(tt.input); got != tt.want {
				t.Errorf("looksLikeFlag(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func sampleResult(backend string) *benchkit.Result {
	return &benchkit.Result{
		Backend: backend,
		Profile: "dev",
		WriteLatency: benchkit.LatencyStats{
			Count: 100,
			P50:   500 * time.Nanosecond,
			P99:   5 * time.Microsecond,
		},
		LoadLatency: benchkit.LatencyStats{
			Count: 50,
			P50:   600 * time.Nanosecond,
			P99:   6 * time.Microsecond,
		},
		ColdReadLatency: benchkit.LatencyStats{
			Count: 10,
			P50:   800 * time.Nanosecond,
		},
		GCMaxPause:      2 * time.Millisecond,
		TailRatio:       10.0,
		AllocsPerOp:     200,
		WriteThroughput: 50000,
		Memory: benchkit.ResourceStats{
			After: 2 * 1024 * 1024,
		},
		Disk: benchkit.DiskStats{
			DatabaseBytes:      1024 * 1024,
			WriteAmplification: 2.0,
		},
	}
}

func TestBuildComparisonTable(t *testing.T) {
	t.Parallel()

	results := map[string]*benchkit.Result{
		"memory": sampleResult("memory"),
		"sqlite": sampleResult("sqlite"),
		"failed": {Backend: "failed", Error: "disk full"},
		"nilled": nil,
	}

	table := buildComparisonTable(results)
	if table == nil {
		t.Fatal("expected non-nil table")
	}

	if len(table.Headers) != 13 {
		t.Errorf("expected 13 headers, got %d", len(table.Headers))
	}

	if table.Headers[0] != "Backend" {
		t.Errorf("first header should be 'Backend', got %q", table.Headers[0])
	}

	if len(table.Rows) != 4 {
		t.Errorf("expected 4 rows, got %d", len(table.Rows))
	}

	for _, row := range table.Rows {
		if len(row) != len(table.Headers) {
			t.Errorf("row has %d cells, expected %d", len(row), len(table.Headers))
		}

		name := row[0]

		switch name {
		case "failed":
			if !strings.HasPrefix(row[1], "FAILED:") {
				t.Errorf("failed row should show FAILED: prefix, got %q", row[1])
			}

		case "nilled":
			if !strings.HasPrefix(row[1], "FAILED:") {
				t.Errorf("nil-result row should show FAILED: prefix, got %q", row[1])
			}
		}
	}
}

func TestBuildSweepTable(t *testing.T) {
	t.Parallel()

	results := []benchkit.SweepResult{
		{Parameter: "workers", Value: 1, Result: sampleResult("memory")},
		{Parameter: "workers", Value: 2, Result: sampleResult("memory")},
		{Parameter: "workers", Value: 4, Result: nil},
	}

	table := buildSweepTable(results)
	if table == nil {
		t.Fatal("expected non-nil table")
	}

	if len(table.Headers) != 8 {
		t.Errorf("expected 8 headers, got %d", len(table.Headers))
	}

	if table.Headers[0] != "Workers" {
		t.Errorf("first header should be 'Workers', got %q", table.Headers[0])
	}

	if len(table.Rows) != 3 {
		t.Errorf("expected 3 rows, got %d", len(table.Rows))
	}
}

func TestBuildSweepTable_Empty(t *testing.T) {
	t.Parallel()

	table := buildSweepTable(nil)
	if table == nil {
		t.Fatal("expected non-nil table for empty input")
	}
}

func TestBuildRunSummaryTable(t *testing.T) {
	t.Parallel()

	r := sampleResult("memory")
	r.Streams = 100
	r.EventsPerStream = 5
	r.TotalEvents = 500
	r.Codec = "json"
	r.Duration = 100 * time.Millisecond
	r.Workers = 4

	table := buildRunSummaryTable(r)
	if table == nil {
		t.Fatal("expected non-nil table")
	}

	if len(table.Headers) != 2 {
		t.Errorf("expected 2 headers (Metric/Value), got %d", len(table.Headers))
	}

	if table.Headers[0] != "Metric" || table.Headers[1] != "Value" {
		t.Errorf("headers should be Metric/Value, got %v", table.Headers)
	}

	foundBackend := false

	for _, row := range table.Rows {
		if row[0] == "Backend" && row[1] == "memory" {
			foundBackend = true
		}
	}

	if !foundBackend {
		t.Error("table should contain Backend=memory row")
	}
}

func TestBuildRunSummaryTable_Error(t *testing.T) {
	t.Parallel()

	r := &benchkit.Result{Backend: "sqlite", Error: "connection refused"}

	table := buildRunSummaryTable(r)
	if table == nil {
		t.Fatal("expected non-nil table")
	}

	if len(table.Headers) != 2 || table.Headers[0] != "Status" {
		t.Errorf("error table should have Status/Message headers, got %v", table.Headers)
	}

	if len(table.Rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(table.Rows))
	}

	if table.Rows[0][0] != "FAILED" {
		t.Errorf("first cell should be FAILED, got %q", table.Rows[0][0])
	}
}

func TestComparisonWinnerSummary(t *testing.T) {
	t.Parallel()

	fast := sampleResult("memory")
	fast.WriteLatency.P50 = 100 * time.Nanosecond
	fast.LoadLatency.P50 = 200 * time.Nanosecond
	fast.GCMaxPause = 1 * time.Millisecond
	fast.AllocsPerOp = 50
	fast.Memory.After = 1024 * 1024

	slow := sampleResult("sqlite")
	slow.WriteLatency.P50 = 500 * time.Nanosecond
	slow.LoadLatency.P50 = 800 * time.Nanosecond
	slow.GCMaxPause = 5 * time.Millisecond
	slow.AllocsPerOp = 300
	slow.Memory.After = 10 * 1024 * 1024

	results := map[string]*benchkit.Result{"memory": fast, "sqlite": slow}

	summary := comparisonWinnerSummary(results)
	if summary == "" {
		t.Fatal("expected non-empty summary")
	}

	if !strings.HasPrefix(summary, "Best — ") {
		t.Errorf("summary should start with 'Best — ', got %q", summary)
	}

	if !strings.Contains(summary, "writes: memory") {
		t.Errorf("memory should win writes (lower P50), got %q", summary)
	}

	if !strings.Contains(summary, "allocs: memory") {
		t.Errorf("memory should win allocs (lower), got %q", summary)
	}

	if !strings.Contains(summary, "heap: memory") {
		t.Errorf("memory should win heap (lower), got %q", summary)
	}
}

func TestComparisonWinnerSummary_SingleResult(t *testing.T) {
	t.Parallel()

	results := map[string]*benchkit.Result{
		"memory": sampleResult("memory"),
	}

	summary := comparisonWinnerSummary(results)
	if summary == "" {
		t.Fatal("single valid result should still produce a summary")
	}

	if !strings.Contains(summary, "memory") {
		t.Errorf("summary should mention memory, got %q", summary)
	}
}

func TestComparisonWinnerSummary_AllFailed(t *testing.T) {
	t.Parallel()

	results := map[string]*benchkit.Result{
		"memory": {Backend: "memory", Error: "crashed"},
		"sqlite": {Backend: "sqlite", Error: "crashed"},
	}

	summary := comparisonWinnerSummary(results)
	if summary != "" {
		t.Errorf("all-failed results should produce empty summary, got %q", summary)
	}
}

func TestComparisonWinnerSummary_Empty(t *testing.T) {
	t.Parallel()

	summary := comparisonWinnerSummary(nil)
	if summary != "" {
		t.Errorf("empty results should produce empty summary, got %q", summary)
	}
}

func TestFmtDur(t *testing.T) {
	t.Parallel()

	tests := []struct {
		d    time.Duration
		want string
	}{
		{500 * time.Nanosecond, "500ns"},
		{5 * time.Microsecond, "5µs"},
		{5 * time.Millisecond, "5ms"},
		{2 * time.Second, "2s"},
	}

	for _, tt := range tests {
		got := fmtDur(tt.d)
		if got != tt.want {
			t.Errorf("fmtDur(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestFmtGCDash(t *testing.T) {
	t.Parallel()

	if got := fmtGCDash(0); got != "-" {
		t.Errorf("fmtGCDash(0) = %q, want '-'", got)
	}

	if got := fmtGCDash(5 * time.Millisecond); got != "5ms" {
		t.Errorf("fmtGCDash(5ms) = %q, want '5ms'", got)
	}
}

func TestFmtRatioDash(t *testing.T) {
	t.Parallel()

	if got := fmtRatioDash(0); got != "-" {
		t.Errorf("fmtRatioDash(0) = %q, want '-'", got)
	}

	if got := fmtRatioDash(2.5); got != "2.5x" {
		t.Errorf("fmtRatioDash(2.5) = %q, want '2.5x'", got)
	}
}

func TestFmtAllocDash(t *testing.T) {
	t.Parallel()

	if got := fmtAllocDash(0); got != "-" {
		t.Errorf("fmtAllocDash(0) = %q, want '-'", got)
	}

	if got := fmtAllocDash(200); got != "200" {
		t.Errorf("fmtAllocDash(200) = %q, want '200'", got)
	}
}

func TestFmtCoVDash(t *testing.T) {
	t.Parallel()

	if got := fmtCoVDash(0); got != "-" {
		t.Errorf("fmtCoVDash(0) = %q, want '-'", got)
	}

	if got := fmtCoVDash(0.05); got != "5.0%" {
		t.Errorf("fmtCoVDash(0.05) = %q, want '5.0%%'", got)
	}
}

func TestResolveFormat(t *testing.T) {
	t.Parallel()

	if got := resolveFormat("json"); got != "json" {
		t.Errorf("resolveFormat(json) = %q, want json", got)
	}

	if got := resolveFormat("csv"); got != "csv" {
		t.Errorf("resolveFormat(csv) = %q, want csv", got)
	}
}

func TestTruncateMsg(t *testing.T) {
	t.Parallel()

	if got := truncateMsg("short", 10); got != "short" {
		t.Errorf("truncateMsg(short, 10) = %q, want 'short'", got)
	}

	if got := truncateMsg("this is a long message", 10); got != "this is..." {
		t.Errorf("truncateMsg(long, 10) = %q, want 'this is...'", got)
	}
}

func TestTitleCase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input, want string
	}{
		{"", ""},
		{"workers", "Workers"},
		{"batchSize", "BatchSize"},
	}

	for _, tt := range tests {
		if got := titleCase(tt.input); got != tt.want {
			t.Errorf("titleCase(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

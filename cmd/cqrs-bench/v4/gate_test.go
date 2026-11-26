package main

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/benchkit/v4"
)

// strictRun is a finished run of backend that skipped exactly the phases
// inherent to it.
func strictRun(backend string) *benchkit.Result {
	return &benchkit.Result{SkippedPhases: inherentSkippedPhases(backend)}
}

// TestStrictGate pins what --strict holds a run to: every phase the backend
// can run, no integrity errors, and (with repeats) no NOISY headline metric.
func TestStrictGate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		backend string
		mutate  func(*benchkit.Result)
		wantNil bool
		teaches []string
	}{
		{name: "a clean preset run passes", backend: "sqlite", wantNil: true},
		{name: "a clean system run passes", backend: "system-memory", wantNil: true},
		{
			name: "an inherent skip is not a failure", backend: "system-memory", wantNil: true,
			mutate: func(result *benchkit.Result) { result.SkippedPhases = inherentSkippedPhases("system-memory") },
		},
		{
			name:    "a phase the backend can run was skipped",
			backend: "sqlite",
			mutate:  func(result *benchkit.Result) { result.SkippedPhases = append(result.SkippedPhases, "query phase") },
			teaches: []string{"query phase"},
		},
		{
			name: "the capability baseline is stale", backend: "sqlite",
			mutate:  func(result *benchkit.Result) { result.SkippedPhases = nil },
			teaches: []string{"baseline", "metaengine phase"},
		},
		{
			name: "integrity errors fail", backend: "sqlite",
			mutate:  func(result *benchkit.Result) { result.IntegrityErrors = 2 },
			teaches: []string{"2 integrity error"},
		},
		{
			name: "a NOISY headline metric fails with repeats", backend: "sqlite",
			mutate: func(result *benchkit.Result) {
				result.RepeatCount = 3
				result.MetricVariation = []benchkit.MetricVariation{{
					Name: benchkit.MetricWriteThroughput, CoV: 0.14, Reliable: false,
				}}
			},
			teaches: []string{"NOISY " + benchkit.MetricWriteThroughput, "14.0%", "3 runs"},
		},
		{
			name: "a NOISY tail metric is not a headline", backend: "sqlite", wantNil: true,
			mutate: func(result *benchkit.Result) {
				result.RepeatCount = 3
				result.MetricVariation = []benchkit.MetricVariation{{
					Name: benchkit.MetricWriteP99NS, CoV: 0.40, Reliable: false,
				}}
			},
		},
		{
			name: "a stable headline metric passes", backend: "sqlite", wantNil: true,
			mutate: func(result *benchkit.Result) {
				result.RepeatCount = 3
				result.MetricVariation = []benchkit.MetricVariation{{
					Name: benchkit.MetricWriteP50NS, CoV: 0.03, Reliable: true,
				}}
			},
		},
		{
			name: "a single run has no noise verdict", backend: "sqlite", wantNil: true,
			mutate: func(result *benchkit.Result) {
				result.MetricVariation = []benchkit.MetricVariation{{
					Name: benchkit.MetricWriteThroughput, CoV: 0.90, Reliable: false,
				}}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := strictRun(tc.backend)
			if tc.mutate != nil {
				tc.mutate(result)
			}

			err := strictGate(tc.backend, result)
			if tc.wantNil {
				if err != nil {
					t.Fatalf("strictGate = %v, want nil", err)
				}

				return
			}

			if !errors.Is(err, errGate) {
				t.Fatalf("strictGate = %v, want errGate", err)
			}

			if !strings.Contains(err.Error(), tc.backend) {
				t.Errorf("verdict should name the backend: %v", err)
			}

			for _, teaches := range tc.teaches {
				if !strings.Contains(err.Error(), teaches) {
					t.Errorf("verdict %q should mention %q", err, teaches)
				}
			}
		})
	}
}

func TestCheckStrict(t *testing.T) {
	t.Parallel()

	if err := checkStrict(false, []string{"--skip-reads"}, []string{"--replay"}); err != nil {
		t.Errorf("without --strict nothing is rejected, got %v", err)
	}

	errorMentions(t, checkStrict(true, []string{"--skip-snapshot"}, nil), "--strict", "--skip-snapshot")
	errorMentions(t, checkStrict(true, nil, []string{"--soak"}), "--strict", "--soak")

	if err := checkStrict(true, nil, nil); err != nil {
		t.Errorf("--strict on its own is valid, got %v", err)
	}
}

// TestInherentSkippedPhases_MatchBenchkit pins the capability baseline
// against a real run of every backend this build can open. --strict compares
// each run with this table, so a backend whose composition changed shows up
// here before it shows up as a confusing gate failure.
func TestInherentSkippedPhases_MatchBenchkit(t *testing.T) {
	t.Parallel()

	for _, name := range backendNames() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			bound, err := openBackend(name, openEnv{cgo: cgoEnabled})
			if errors.Is(err, errDSNRequired) || errors.Is(err, errBackendUnavailable) {
				t.Skipf("%s needs a server or a CGo build: %v", name, err)
			}

			if err != nil {
				t.Fatalf("openBackend: %v", err)
			}

			t.Cleanup(func() { _ = bound.close() })

			result, err := benchkit.Run(t.Context(), benchkit.Config{Profile: benchkit.ProfileDev}, bound.factory)
			if err != nil {
				t.Fatalf("benchkit.Run: %v", err)
			}

			skipped := slices.Clone(result.SkippedPhases)
			slices.Sort(skipped)

			inherent := slices.Clone(inherentSkippedPhases(name))
			slices.Sort(inherent)

			if !slices.Equal(skipped, inherent) {
				t.Errorf("a plain %s run skips %v, the baseline lists %v", name, skipped, inherent)
			}

			if err := strictGate(name, result); err != nil {
				t.Errorf("a plain %s run must pass --strict: %v", name, err)
			}
		})
	}
}

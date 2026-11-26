package main

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/benchkit/v4"
)

// memoryResult runs the dev profile against the in-memory backend with
// config applied.
func memoryResult(t *testing.T, apply func(*benchkit.Config)) *benchkit.Result {
	t.Helper()

	bound, err := openBackend("memory", openEnv{cgo: cgoEnabled})
	if err != nil {
		t.Fatalf("openBackend(memory): %v", err)
	}

	t.Cleanup(func() { _ = bound.close() })

	config := benchkit.Config{Profile: benchkit.ProfileDev}
	apply(&config)

	result, err := benchkit.Run(t.Context(), config, bound.factory)
	if err != nil {
		t.Fatalf("benchkit.Run: %v", err)
	}

	return result
}

// TestPhaseDocs_FollowBenchkitExecutionOrder pins the listing's order
// against benchkit's own phase names; the raw-sink phase is the one the
// runner executes but benchkit.PhaseNames omits.
func TestPhaseDocs_FollowBenchkitExecutionOrder(t *testing.T) {
	t.Parallel()

	names := make([]string, 0, len(phaseDocs()))
	for _, phase := range phaseDocs() {
		names = append(names, phase.name)
	}

	if want := append(slices.Clone(benchkit.PhaseNames()), "raw-sink"); !slices.Equal(names, want) {
		t.Errorf("listed phases = %q, want benchkit's order %q", names, want)
	}
}

// TestPhaseDocs_MetricNamesExist: every benchstat name the listing promises
// is one benchkit can actually emit.
func TestPhaseDocs_MetricNamesExist(t *testing.T) {
	t.Parallel()

	known := benchkit.MetricNames()

	for _, phase := range phaseDocs() {
		if strings.HasPrefix(phase.metrics, "report:") {
			continue
		}

		for _, name := range strings.Fields(phase.metrics) {
			if !slices.Contains(known, name) {
				t.Errorf("phase %q lists metric %q, which benchkit never emits", phase.name, name)
			}
		}
	}
}

// TestPhaseDocs_EverySkipFlagSkipsItsPhase pins each documented flag against
// a real run, so `list-phases` cannot claim a flag that does nothing.
func TestPhaseDocs_EverySkipFlagSkipsItsPhase(t *testing.T) {
	t.Parallel()

	for _, phase := range phaseDocs() {
		if phase.skip == nil {
			continue
		}

		t.Run(phase.name, func(t *testing.T) {
			t.Parallel()

			result := memoryResult(t, phase.skip)

			if !slices.Contains(result.SkippedPhases, phase.label) {
				t.Errorf("%s is documented to skip %q, the run skipped %v",
					phase.flags[len(phase.flags)-1], phase.label, result.SkippedPhases)
			}
		})
	}
}

// TestPhaseDocs_FlagsAgreeWithTheSkipFlags: a phase documented as skipped by
// a --skip-* flag has that flag on benchFlags, and vice versa.
func TestPhaseDocs_FlagsAgreeWithTheSkipFlags(t *testing.T) {
	t.Parallel()

	documented := []string{}

	for _, phase := range phaseDocs() {
		for _, flag := range phase.flags {
			if strings.HasPrefix(flag, "--skip-") {
				documented = append(documented, flag)
			}
		}
	}

	all := benchFlags{
		SkipReads: true, SkipReadModels: true, SkipProjection: true, SkipMetaEngine: true,
		SkipBatchWrite: true, SkipRawSink: true, SkipJourney: true, SkipQuery: true,
		SkipSnapshot: true, SkipMixed: true,
	}

	slices.Sort(documented)

	if !slices.Equal(documented, all.skipFlags()) {
		t.Errorf("flags in the listing = %q, skip flags on the command = %q", documented, all.skipFlags())
	}
}

func TestListPhases(t *testing.T) {
	t.Parallel()

	var out strings.Builder

	if err := listPhasesHandler(
		t.Context(),
		&appConfig{},
		noFlagsForTest(),
	); !errors.Is(err, errImpossible) &&
		err != nil {
		t.Fatalf("listPhasesHandler: %v", err)
	}

	if !strings.Contains(out.String(), "") {
		t.Fatal("unreachable")
	}
}

package main

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	cmdguard "github.com/larsartmann/cmdguard/v5/pkg/cmdguard/v5"

	"github.com/larsartmann/go-cqrs-lite/benchkit/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
)

// errorMentions fails when err is nil or its text lacks any of teaches.
func errorMentions(t *testing.T, err error, teaches ...string) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected an error mentioning %q", teaches)

		return
	}

	for _, want := range teaches {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q should mention %q", err, want)
		}
	}
}

func TestBackendsFromList(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		list    string
		want    []string
		teaches []string
	}{
		{name: "canonical names keep their order", list: "pebble,memory", want: []string{"pebble", "memory"}},
		{name: "aliases resolve and spaces are trimmed", list: " mem , pg ", want: []string{"memory", "postgres"}},
		{name: "unknown backend", list: "memory,bogus", teaches: []string{"unknown backend", "bogus"}},
		{name: "backend named twice", list: "memory,mem", teaches: []string{"twice"}},
		{name: "empty list", list: " , ", teaches: []string{"--backends"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := backendsFromList(tc.list)

			if len(tc.teaches) > 0 {
				errorMentions(t, err, tc.teaches...)

				return
			}

			if err != nil {
				t.Fatalf("backendsFromList(%q): %v", tc.list, err)
			}

			if !slices.Equal(got, tc.want) {
				t.Errorf("backendsFromList(%q) = %q, want %q", tc.list, got, tc.want)
			}
		})
	}
}

// TestBenchFlagsPlan_ReportsEveryProblem: one round-trip shows every invalid
// flag, not only the first.
func TestBenchFlagsPlan_ReportsEveryProblem(t *testing.T) {
	t.Parallel()

	flags := benchFlags{
		Profile:      "nope",
		Codec:        "xml",
		PayloadSizes: "0",
		Repeat:       -1,
		Durability:   "ultra",
		Output:       filepath.Join(t.TempDir(), "missing", "report.json"),
	}

	_, err := flags.plan()

	errorMentions(t, err, "nope", "xml", "--payload-sizes", "--repeat", "ultra", "--output")

	for _, want := range []error{errProfile, errCodec, errFlagValue} {
		if !errors.Is(err, want) {
			t.Errorf("error should wrap %v: %v", want, err)
		}
	}
}

func TestBenchFlagsPlan_ResolvesSharedFlags(t *testing.T) {
	t.Parallel()

	flags := benchFlags{
		Profile:      "small",
		Codec:        "cbor",
		PayloadSizes: " 64, 256 ",
		Durability:   " STRICT ",
		Repeat:       3,
		Timeout:      cmdguard.FromDuration(time.Minute),
		Quiet:        true,
		SkipMixed:    true,
	}

	plan, err := flags.plan()
	if err != nil {
		t.Fatalf("plan: %v", err)
	}

	if plan.profile.Name != "small" {
		t.Errorf("profile = %q, want small", plan.profile.Name)
	}

	if !slices.Equal(plan.config.PayloadSizes, []int{64, 256}) {
		t.Errorf("PayloadSizes = %v, want [64 256]", plan.config.PayloadSizes)
	}

	if plan.env.durability != stack.DurabilityStrict || !plan.env.durabilitySet {
		t.Errorf("durability = %q (set %v), want strict", plan.env.durability, plan.env.durabilitySet)
	}

	if plan.env.generations != true || plan.config.Repeat != 3 {
		t.Errorf("repeats were not resolved: generations %v, Repeat %d", plan.env.generations, plan.config.Repeat)
	}

	if plan.budget != time.Minute {
		t.Errorf("budget = %s, want 1m0s", plan.budget)
	}

	if plan.config.Strict {
		t.Error("Config.Strict must stay false: the CLI runs the strict gate itself")
	}

	if plan.config.ProgressWriter != nil {
		t.Error("--quiet must switch progress off")
	}
}

// TestBenchFlagsConfig_ReachesBenchkit: every skip flag and tuning flag is
// copied into the one benchkit.Config all three commands share.
func TestBenchFlagsConfig_ReachesBenchkit(t *testing.T) {
	t.Parallel()

	flags := benchFlags{
		SkipReads: true, SkipReadModels: true, SkipProjection: true, SkipMetaEngine: true,
		SkipBatchWrite: true, SkipRawSink: true, SkipJourney: true, SkipQuery: true,
		SkipSnapshot: true, SkipMixed: true,
		Interpolated: true, ReservoirSize: 4096, PayloadSize: 128,
	}

	config := flags.config(benchkit.ProfileDev, nil, nil)

	for name, skipped := range map[string]bool{
		"reads": config.SkipReads, "read models": config.SkipReadModels,
		"projections": config.SkipProjections, "metaengine": config.SkipMetaEngine,
		"batch write": config.SkipBatchWrite, "raw sink": config.SkipRawSink,
		"journey": config.SkipJourney, "query": config.SkipQuery,
		"snapshot": config.SkipSnapshot, "mixed": config.SkipMixed,
	} {
		if !skipped {
			t.Errorf("the %s phase was not skipped", name)
		}
	}

	if !config.InterpolatedPercentiles || config.ReservoirSize != 4096 || config.PayloadSize != 128 {
		t.Errorf("tuning flags were dropped: interpolated %v, reservoir %d, payload %d",
			config.InterpolatedPercentiles, config.ReservoirSize, config.PayloadSize)
	}
}

// TestBenchFlagsPlan_RejectsASinglePayloadSize: one size is not a
// distribution.
func TestBenchFlagsPlan_RejectsASinglePayloadSize(t *testing.T) {
	t.Parallel()

	_, err := (benchFlags{PayloadSizes: "128"}).plan()

	errorMentions(t, err, "--payload-sizes", "at least two")
}

func TestSkipFlagsAreNamedAndSorted(t *testing.T) {
	t.Parallel()

	flags := benchFlags{SkipSnapshot: true, SkipMixed: true}

	if got := flags.skipFlags(); !slices.Equal(got, []string{"--skip-mixed", "--skip-snapshot"}) {
		t.Errorf("skipFlags = %q", got)
	}

	if got := (benchFlags{}).skipFlags(); len(got) != 0 {
		t.Errorf("no skip flag is set, got %q", got)
	}
}

func TestBudgetContext(t *testing.T) {
	t.Parallel()

	t.Run("no budget only adds cancellation", func(t *testing.T) {
		t.Parallel()

		plan := benchPlan{budget: 0}

		ctx, cancel := plan.budgetContext(t.Context(), 3, 0)
		t.Cleanup(cancel)

		if _, bounded := ctx.Deadline(); bounded {
			t.Error("--timeout 0 must not set a deadline")
		}
	})

	cases := []struct {
		name    string
		budget  time.Duration
		runs    int
		atLeast time.Duration
		want    time.Duration
	}{
		{name: "one budget per run", budget: time.Minute, runs: 3, want: 3 * time.Minute},
		{
			name: "at least the soak duration", budget: time.Minute, runs: 1, atLeast: 10 * time.Minute,
			want: 10 * time.Minute,
		},
		{name: "zero runs still get one budget", budget: time.Minute, runs: 0, want: time.Minute},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			plan := benchPlan{budget: tc.budget}

			before := time.Now()

			ctx, cancel := plan.budgetContext(t.Context(), tc.runs, tc.atLeast)
			t.Cleanup(cancel)

			deadline, bounded := ctx.Deadline()
			if !bounded {
				t.Fatal("a budget must set a deadline")
			}

			if got := deadline.Sub(before); got < tc.want-time.Second || got > tc.want {
				t.Errorf("deadline in %s, want about %s", got, tc.want)
			}
		})
	}
}

func TestBenchmarkErrorExplainsTheCause(t *testing.T) {
	t.Parallel()

	plan := benchPlan{budget: time.Minute}
	cause := errors.New("write phase: context canceled")

	t.Run("interrupted command", func(t *testing.T) {
		t.Parallel()

		parent, cancel := context.WithCancel(t.Context())
		cancel()

		err := plan.benchmarkError(parent, parent, 1, cause)
		if !errors.Is(err, errInterrupted) {
			t.Errorf("error = %v, want errInterrupted", err)
		}

		errorMentions(t, err, "interrupted", "temporary store was removed")
	})

	t.Run("exhausted budget", func(t *testing.T) {
		t.Parallel()

		parent := t.Context()

		runCtx, cancel := context.WithTimeout(parent, time.Nanosecond)
		t.Cleanup(cancel)

		<-runCtx.Done()

		err := plan.benchmarkError(parent, runCtx, 7, cause)
		if !errors.Is(err, errBudget) {
			t.Errorf("error = %v, want errBudget", err)
		}

		errorMentions(t, err, "1m0s per run", "7 run", "--timeout")
	})

	t.Run("plain failure", func(t *testing.T) {
		t.Parallel()

		err := plan.benchmarkError(t.Context(), t.Context(), 1, cause)
		if !errors.Is(err, errBenchmark) || !errors.Is(err, cause) {
			t.Errorf("error = %v, want errBenchmark wrapping the cause", err)
		}
	})
}

// TestProfileNamesMatchBenchkitAndTheFlag: the profile list is written by
// hand (benchkit exposes none), so it is pinned from both sides.
func TestProfileNamesMatchBenchkitAndTheFlag(t *testing.T) {
	t.Parallel()

	for _, name := range profileNames() {
		if _, ok := benchkit.ProfileByName(name); !ok {
			t.Errorf("profile %q is listed but benchkit does not know it", name)
		}
	}

	field, ok := reflect.TypeFor[benchFlags]().FieldByName("Profile")
	if !ok {
		t.Fatal("benchFlags has no Profile field")
	}

	if got := strings.Split(field.Tag.Get("values"), ","); !slices.Equal(got, profileNames()) {
		t.Errorf("--profile values = %q, want %q", got, profileNames())
	}
}

func TestCheckWritable(t *testing.T) {
	t.Parallel()

	for _, stdout := range []string{"", "-"} {
		if err := checkWritable(stdout); err != nil {
			t.Errorf("checkWritable(%q) = %v, want nil (stdout)", stdout, err)
		}
	}

	if err := checkWritable(filepath.Join(t.TempDir(), "report.json")); err != nil {
		t.Errorf("checkWritable in a writable directory = %v", err)
	}

	errorMentions(t, checkWritable(filepath.Join(t.TempDir(), "missing", "report.json")), "--output")
}

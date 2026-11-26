package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/larsartmann/go-codec"

	"github.com/larsartmann/go-cqrs-lite/benchkit/v4"
)

// benchPlan is a validated benchmark request: every shared flag resolved and
// checked before a store is created. Commands add their own validation on
// top and then open their backends.
type benchPlan struct {
	profile benchkit.Profile
	sizes   []int
	config  benchkit.Config
	env     openEnv
	// budget is the wall-clock budget per run; zero disables it.
	budget time.Duration
	target reportTarget
}

// plan validates the shared flags and reports every problem it finds, not
// only the first, so one round-trip fixes them all.
func (b benchFlags) plan() (benchPlan, error) {
	tier, tierSet, durabilityErr := parseDurability(b.Durability)

	profile, profileErr := profileByName(b.Profile)

	enc, codecErr := codecByName(b.Codec)

	sizes, sizesErr := parseSizeList("payload-sizes", b.PayloadSizes)

	err := errors.Join(
		durabilityErr, profileErr, codecErr, sizesErr,
		b.checkNumbers(sizes), checkWritable(b.Output),
	)
	if err != nil {
		return benchPlan{}, err
	}

	return benchPlan{
		profile: profile,
		sizes:   sizes,
		config:  b.config(profile, enc, sizes),
		env: openEnv{
			dsn:           b.DSN,
			dir:           b.Dir,
			durability:    tier,
			durabilitySet: tierSet,
			generations:   b.Repeat > 1,
			cgo:           cgoEnabled,
		},
		budget: b.Timeout.Duration(),
		target: fileTarget(b.Output),
	}, nil
}

// checkNumbers validates the numeric flags. sizes is the parsed
// --payload-sizes (nil when the flag is unset).
func (b benchFlags) checkNumbers(sizes []int) error {
	var errs [5]error

	if b.Repeat < 0 {
		errs[0] = fmt.Errorf("%w: --repeat must be 0 or greater, got %d", errFlagValue, b.Repeat)
	}

	if b.PayloadSize < 0 {
		errs[1] = fmt.Errorf("%w: --payload-size must be 0 or greater, got %d", errFlagValue, b.PayloadSize)
	}

	if b.ReservoirSize < 0 {
		errs[2] = fmt.Errorf("%w: --reservoir-size must be 0 or greater, got %d", errFlagValue, b.ReservoirSize)
	}

	if b.Timeout.Duration() < 0 {
		errs[3] = fmt.Errorf("%w: --timeout must be 0 or greater, got %s", errFlagValue, b.Timeout.Duration())
	}

	if len(sizes) == 1 {
		errs[4] = fmt.Errorf(
			"%w: --payload-sizes needs at least two sizes for a mixed workload, got one", errFlagValue,
		)
	}

	return errors.Join(errs[:]...)
}

// config is the benchkit configuration the three benchmark commands share.
//
// Config.Strict stays false: benchkit's strict mode rejects any skipped
// phase, including the ones a backend has no component for, which made
// --strict fail on every bundled backend. The CLI runs the same gate itself
// against the backend's inherent skips (see strictGate).
func (b benchFlags) config(profile benchkit.Profile, enc codec.Codec, sizes []int) benchkit.Config {
	config := benchkit.Config{
		Profile:                 profile,
		PayloadSize:             b.PayloadSize,
		PayloadSizes:           sizes,
		Codec:                   enc,
		Repeat:                  b.Repeat,
		SkipReads:               b.SkipReads,
		SkipReadModels:          b.SkipReadModels,
		SkipProjections:         b.SkipProjection,
		SkipMetaEngine:          b.SkipMetaEngine,
		SkipBatchWrite:          b.SkipBatchWrite,
		SkipRawSink:             b.SkipRawSink,
		SkipJourney:             b.SkipJourney,
		SkipQuery:               b.SkipQuery,
		SkipSnapshot:            b.SkipSnapshot,
		SkipMixed:               b.SkipMixed,
		InterpolatedPercentiles: b.Interpolated,
		ReservoirSize:           b.ReservoirSize,
	}

	if !b.Quiet && b.Progress.Duration() > 0 {
		config.ProgressWriter = os.Stderr
		config.ProgressInterval = b.Progress.Duration()
	}

	return config
}

// skipFlags names the --skip-* flags that are set, for error messages.
func (b benchFlags) skipFlags() []string {
	var set []string

	for flag, enabled := range map[string]bool{
		"--skip-reads":       b.SkipReads,
		"--skip-read-models": b.SkipReadModels,
		"--skip-projections": b.SkipProjection,
		"--skip-metaengine":  b.SkipMetaEngine,
		"--skip-batch-write": b.SkipBatchWrite,
		"--skip-raw-sink":    b.SkipRawSink,
		"--skip-journey":     b.SkipJourney,
		"--skip-query":       b.SkipQuery,
		"--skip-snapshot":    b.SkipSnapshot,
		"--skip-mixed":       b.SkipMixed,
	} {
		if enabled {
			set = append(set, flag)
		}
	}

	slices.Sort(set)

	return set
}

// open prepares the named backends for the plan. A failure closes every
// backend that was already opened, so nothing outlives the command.
func (p benchPlan) open(calls storeCalls, names ...string) ([]binding, error) {
	env := p.env
	env.calls = calls

	bindings := make([]binding, 0, len(names))

	for _, name := range names {
		bound, err := openBackend(name, env)
		if err != nil {
			reportClose(bindings)

			return nil, err
		}

		bindings = append(bindings, bound)
	}

	return bindings, nil
}

// budgetContext bounds ctx by the per-run budget times runs, never by less
// than atLeast (so a soak gets the time it was asked for). A zero budget
// only adds cancellation.
func (p benchPlan) budgetContext(
	ctx context.Context, runs int, atLeast time.Duration,
) (context.Context, context.CancelFunc) {
	if p.budget <= 0 {
		return context.WithCancel(ctx)
	}

	return context.WithTimeout(ctx, max(p.budget*time.Duration(max(runs, 1)), atLeast))
}

// benchmarkError explains a failed run: an interrupted command, a run that
// outlived its budget, or a plain benchmark failure.
func (p benchPlan) benchmarkError(ctx, runCtx context.Context, runs int, err error) error {
	switch {
	case ctx.Err() != nil:
		return fmt.Errorf("%w: %w (partial results were discarded; the temporary store was removed)",
			errInterrupted, err)
	case runCtx.Err() != nil && p.budget > 0:
		return fmt.Errorf(
			"%w: the command ran past the budget of %s per run across %d run(s); "+
				"raise --timeout (0 disables it) or pick a smaller --profile: %w",
			errBudget, p.budget, runs, err,
		)
	default:
		return fmt.Errorf("%w: %w", errBenchmark, err)
	}
}

// backendsFromList resolves a comma-separated --backends list into canonical
// names in the given order. An alias resolves to the canonical name, an
// unknown name and a backend named twice are errors.
func backendsFromList(list string) ([]string, error) {
	if strings.TrimSpace(list) == "" {
		return nil, fmt.Errorf("%w: --backends names no backend (use: %s)",
			errFlagValue, strings.Join(backendNames(), ", "))
	}

	var (
		names []string
		errs  []error
	)

	for _, entry := range strings.Split(list, ",") {
		entry = strings.TrimSpace(entry)

		spec, ok := lookupBackend(entry)
		if !ok {
			errs = append(errs, fmt.Errorf("%w %q (use: %s)",
				errBackend, entry, strings.Join(backendNames(), ", ")))

			continue
		}

		if slices.Contains(names, spec.name) {
			errs = append(errs, fmt.Errorf("%w: --backends names %q twice", errFlagValue, spec.name))

			continue
		}

		names = append(names, spec.name)
	}

	return names, errors.Join(errs...)
}

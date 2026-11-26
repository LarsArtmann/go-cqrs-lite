package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/larsartmann/go-cqrs-lite/benchkit/v4"
)

const (
	// soakReportInterval is how often a soak prints an interim line.
	soakReportInterval = 10 * time.Second
	// soakBudgetFactor is how many soak durations a soak may take when
	// --timeout would allow less.
	soakBudgetFactor = 2
	// minRepeats is the smallest --repeat that gives cross-run dispersion.
	minRepeats = 2
)

// runHandler is the run command: one backend, one profile, one report.
func runHandler(ctx context.Context, _ *appConfig, flags runFlags) error {
	plan, err := flags.plan()
	if err != nil {
		return err
	}

	stopProfiling, err := startProfiling(flags.CPUProfile, flags.MemProfile)
	if err != nil {
		return err
	}

	defer stopProfiling()

	calls := storeCalls{warmup: flags.Warmup > 0, recovery: flags.Recovery, replay: flags.Replay}

	bindings, err := plan.open(calls, flags.Backend)
	if err != nil {
		return err
	}

	defer reportClose(bindings)

	warnBindings(bindings)

	config := plan.config
	config.Backend = bindings[0].name
	config.DiskPath = bindings[0].diskPath
	config.Warmup = flags.Warmup
	config.Recovery = flags.Recovery
	config.ReplayOnly = flags.Replay

	if flags.Soak.Duration() > 0 {
		return runSoak(ctx, plan, flags, config, bindings[0])
	}

	return runOnce(ctx, plan, flags, config, bindings[0])
}

// plan validates the run flags on top of the shared ones.
func (f runFlags) plan() (benchPlan, error) {
	plan, baseErr := f.benchFlags.plan()

	soaking := f.Soak.Duration() > 0

	command := "run"
	if soaking {
		command = "run --soak"
	}

	formats := runFormats()
	if soaking {
		formats = reportFormats()
	}

	var conflicting []string

	if f.Replay {
		conflicting = append(conflicting, "--replay")
	}

	if soaking {
		conflicting = append(conflicting, "--soak")
	}

	err := errors.Join(
		baseErr,
		validateFormat(command, f.Format, formats),
		checkStrict(f.Strict, f.skipFlags(), conflicting),
		f.checkWarmup(),
		f.checkReplay(),
		f.checkIncludeRuns(soaking),
	)
	if err != nil {
		return benchPlan{}, err
	}

	return plan, nil
}

func (f runFlags) checkWarmup() error {
	if f.Warmup < 0 {
		return fmt.Errorf("%w: --warmup must be 0 or greater, got %d", errFlagValue, f.Warmup)
	}

	return nil
}

// checkReplay: a replay needs a store to read. An in-memory DSN is empty on
// every open, so only --dir or a file --dsn can hold one.
func (f runFlags) checkReplay() error {
	if f.Replay && f.Dir == "" && (f.DSN == "" || isMemoryDSN(f.DSN)) {
		return fmt.Errorf("%w: --replay needs --dir (or a file --dsn) naming the store to read", errFlagCombo)
	}

	return nil
}

func (f runFlags) checkIncludeRuns(soaking bool) error {
	if f.IncludeRuns && (soaking || f.Format != formatManifest) {
		return fmt.Errorf("%w: --include-runs only applies to --format manifest", errFlagCombo)
	}

	return nil
}

// runOnce benchmarks once, or --repeat times and reports the median.
func runOnce(
	ctx context.Context,
	plan benchPlan,
	flags runFlags,
	config benchkit.Config,
	bound binding,
) error {
	runs := max(flags.Repeat, 1)

	runCtx, cancel := plan.budgetContext(ctx, runs, 0)
	defer cancel()

	var (
		result   *benchkit.Result
		repeated *benchkit.RepeatedResult
		err      error
	)

	if flags.Repeat > 1 {
		repeated, err = benchkit.RunRepeated(runCtx, config, bound.factory)
		if repeated != nil {
			result = repeated.Median
		}
	} else {
		result, err = benchkit.Run(runCtx, config, bound.factory)
	}

	// A run that "succeeded" under an expired context was cut short: benchkit
	// skips the remaining phases instead of failing once some work was done.
	if err == nil {
		err = runCtx.Err()
	}

	if err != nil {
		return plan.benchmarkError(ctx, runCtx, runs, err)
	}

	format := plan.target.resolveFormat(flags.Format)

	if err := plan.target.render(func(out reportWriter) error {
		return renderRunResult(out, format, config, result, repeated, flags.IncludeRuns)
	}); err != nil {
		return err
	}

	strictNote(flags.Strict, result)

	// The gate runs after the report so the artifact still lands for triage.
	if flags.Strict {
		return strictGate(bound.name, result)
	}

	return nil
}

// runSoak repeats the workload for the requested duration and reports leak
// and degradation trends.
func runSoak(
	ctx context.Context,
	plan benchPlan,
	flags runFlags,
	config benchkit.Config,
	bound binding,
) error {
	soak := flags.Soak.Duration()

	runCtx, cancel := plan.budgetContext(ctx, 1, soak*soakBudgetFactor)
	defer cancel()

	var progress io.Writer = os.Stderr
	if flags.Quiet {
		progress = nil
	}

	result, err := benchkit.RunSoak(runCtx, benchkit.SoakConfig{
		Duration:       soak,
		ReportInterval: soakReportInterval,
		ProgressWriter: progress,
		Config:         config,
	}, bound.factory)
	if err == nil {
		err = ctx.Err()
	}

	if err != nil {
		return plan.benchmarkError(ctx, runCtx, 1, err)
	}

	format := plan.target.resolveFormat(flags.Format)

	return plan.target.render(func(out reportWriter) error {
		return renderSoakResult(out, format, result)
	})
}

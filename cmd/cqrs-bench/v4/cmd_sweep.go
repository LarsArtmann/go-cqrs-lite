package main

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/larsartmann/go-cqrs-lite/benchkit/v4"
)

// sweepParam is one parameter the sweep command can vary.
type sweepParam struct {
	// spellings are the accepted --param values.
	spellings []string
	// sweep runs the benchmark once per value.
	sweep func(ctx context.Context, config benchkit.Config, factory benchkit.Factory, values []int) []benchkit.SweepResult
}

// sweepParams lists the sweepable parameters. benchkit labels each result
// with its own parameter name.
func sweepParams() []sweepParam {
	return []sweepParam{
		{spellings: []string{"workers"}, sweep: benchkit.WorkerSweep},
		{spellings: []string{"batchSize", "batch-size", "batch"}, sweep: benchkit.BatchSizeSweep},
		{spellings: []string{"streamLength", "stream-length", "stream"}, sweep: benchkit.StreamLengthSweep},
		{spellings: []string{"gomaxprocs", "gomax"}, sweep: benchkit.GOMAXPROCSSweep},
	}
}

// lookupSweepParam resolves a --param value.
func lookupSweepParam(name string) (sweepParam, bool) {
	for _, param := range sweepParams() {
		if slices.Contains(param.spellings, name) {
			return param, true
		}
	}

	return sweepParam{}, false
}

// sweepHandler is the sweep command: one parameter, several values.
func sweepHandler(ctx context.Context, _ *appConfig, flags sweepFlags) error {
	plan, param, values, err := flags.plan()
	if err != nil {
		return err
	}

	bindings, err := plan.open(storeCalls{}, flags.Backend)
	if err != nil {
		return err
	}

	defer reportClose(bindings)

	warnBindings(bindings)

	bound := bindings[0]

	config := plan.config
	config.Backend = bound.name
	config.DiskPath = bound.diskPath

	// Every value and every repeat counts as a run against the budget.
	runs := len(values) * max(flags.Repeat, 1)

	runCtx, cancel := plan.budgetContext(ctx, runs, 0)
	defer cancel()

	results := param.sweep(runCtx, config, bound.factory, values)

	format := plan.target.resolveFormat(flags.Format)

	if err := plan.target.render(func(out reportWriter) error {
		return renderSweep(out, format, results)
	}); err != nil {
		return err
	}

	if allSweepFailed(results) {
		return fmt.Errorf("%w: every data point failed (see the FAILED rows in the report)", errBenchmark)
	}

	return sweepStrictVerdict(flags, bound, results)
}

// plan validates the sweep flags on top of the shared ones and resolves the
// parameter and its values.
func (f sweepFlags) plan() (benchPlan, sweepParam, []int, error) {
	plan, baseErr := f.BenchFlags.plan()

	var errs []error

	errs = append(errs, baseErr)

	param, paramOK := lookupSweepParam(f.Param)
	if !paramOK {
		errs = append(errs, fmt.Errorf("%w %q (use: workers, batchSize, streamLength, gomaxprocs)",
			errSweepParam, f.Param))
	}

	// --values has no --payload-sizes to fall back on, so it is required and
	// needs at least two points to be a sweep.
	values, valuesErr := parseSizeList("values", f.Values)

	switch {
	case valuesErr != nil:
		errs = append(errs, valuesErr)
	case len(values) < minRepeats:
		errs = append(errs, fmt.Errorf("%w: --values needs at least two numbers to sweep, got %d",
			errFlagValue, len(values)))
	}

	errs = append(errs,
		validateFormat("sweep", f.Format, reportFormats()),
		checkStrict(f.Strict, f.skipFlags(), nil),
	)

	err := errors.Join(errs...)
	if err != nil {
		return benchPlan{}, sweepParam{}, nil, err
	}

	return plan, param, values, nil
}

// allSweepFailed reports whether not one data point produced a result.
func allSweepFailed(results []benchkit.SweepResult) bool {
	for _, sweep := range results {
		if sweep.Result != nil && sweep.Result.Error == "" {
			return false
		}
	}

	return true
}

// sweepStrictVerdict is --strict's verdict over every data point.
func sweepStrictVerdict(flags sweepFlags, bound binding, results []benchkit.SweepResult) error {
	if !flags.Strict {
		return nil
	}

	var errs []error

	for _, sweep := range results {
		if result := sweep.Result; result != nil && result.Error == "" {
			strictNote(true, result)
			errs = append(errs, strictGate(bound.name, result))
		}
	}

	return errors.Join(errs...)
}

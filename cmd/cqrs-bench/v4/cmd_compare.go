package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/benchkit/v4"
)

// compareHandler is the compare command: the same benchmark against several
// backends, reported side by side.
func compareHandler(ctx context.Context, _ *appConfig, flags compareFlags) error {
	plan, names, err := flags.plan()
	if err != nil {
		return err
	}

	bindings, err := plan.open(storeCalls{}, names...)
	if err != nil {
		return err
	}

	defer reportClose(bindings)

	warnBindings(bindings)

	// Every backend and every repeat counts as a run against the budget.
	runs := max(flags.Repeat, 1) * len(bindings)

	runCtx, cancel := plan.budgetContext(ctx, runs, 0)
	defer cancel()

	results := runComparison(runCtx, plan.config, bindings)

	format := plan.target.resolveFormat(flags.Format)

	if err := plan.target.render(func(out reportWriter) error {
		return renderComparison(out, format, results)
	}); err != nil {
		return err
	}

	if allFailed(results) {
		return fmt.Errorf("%w: every backend failed (see the FAILED rows in the report)", errBenchmark)
	}

	return strictVerdict(flags, bindings, results)
}

// plan validates the compare flags on top of the shared ones and resolves
// --backends into canonical names in report order.
func (f compareFlags) plan() (benchPlan, []string, error) {
	plan, baseErr := f.benchFlags.plan()

	names, namesErr := backendsFromList(f.Backends)

	err := errors.Join(
		baseErr,
		namesErr,
		validateFormat("compare", f.Format, reportFormats()),
		checkStrict(f.Strict, f.skipFlags(), nil),
	)
	if err != nil {
		return benchPlan{}, nil, err
	}

	return plan, names, nil
}

// allFailed reports whether not one backend produced a result.
func allFailed(results map[string]*benchkit.Result) bool {
	for _, result := range results {
		if result != nil && result.Error == "" {
			return false
		}
	}

	return true
}

// strictVerdict is --strict's verdict over every compared backend.
func strictVerdict(flags compareFlags, bindings []binding, results map[string]*benchkit.Result) error {
	if !flags.Strict {
		return nil
	}

	var errs []error

	for _, bound := range bindings {
		if result := results[bound.name]; result != nil && result.Error == "" {
			strictNote(true, result)
			errs = append(errs, strictGate(bound.name, result))
		}
	}

	return errors.Join(errs...)
}

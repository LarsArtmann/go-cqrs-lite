package main

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/benchkit/v4"
)

// inherentSkippedPhases lists the phases benchkit always skips for a backend
// because its bundle has no component for them. No stack preset exposes a
// metaengine; the system adapter also lacks kv read models, the projection
// host's journal tail, checkpoints, the journey, and typed queries.
// TestInherentSkippedPhases pins the table against a real run of every
// bundled backend.
//
// --strict holds a run to every other phase. Before this table existed the
// flag was handed to benchkit, whose strict mode rejects any skipped phase,
// so --strict failed on every bundled backend before its noise gate ran.
func inherentSkippedPhases(backend string) []string {
	if backend == "system-memory" {
		return []string{
			"read model phase", "projection phase", "checkpoint phase",
			"journey phase", "query phase", "metaengine phase",
		}
	}

	return []string{"metaengine phase"}
}

// strictGate is the --strict verdict for one finished run: every phase the
// backend can run must have run, the run must be free of integrity errors,
// and with repeats no headline metric may be NOISY. It returns nil when the
// run is decision-grade.
func strictGate(backend string, result *benchkit.Result) error {
	problems := coverageProblems(result.SkippedPhases, inherentSkippedPhases(backend))

	if result.IntegrityErrors > 0 {
		problems = append(problems, fmt.Sprintf("%d integrity error(s) detected", result.IntegrityErrors))
	}

	problems = append(problems, noisyHeadlines(result)...)
	if len(problems) == 0 {
		return nil
	}

	return fmt.Errorf("%w: backend %q: %s", errGate, backend, strings.Join(problems, "; "))
}

// coverageProblems compares the phases a run skipped with the ones inherent
// to its backend, in both directions: a skipped phase the backend can run is
// a coverage hole, and an inherent phase that ran means the baseline is
// stale.
func coverageProblems(skipped, inherent []string) []string {
	var problems []string

	inherentTo := func(phase string) bool { return slices.Contains(inherent, phase) }

	if unexpected := slices.DeleteFunc(slices.Clone(skipped), inherentTo); len(unexpected) > 0 {
		problems = append(problems, fmt.Sprintf("phase(s) skipped that the backend can run: %v", unexpected))
	}

	skippedToo := func(phase string) bool { return slices.Contains(skipped, phase) }

	if stale := slices.DeleteFunc(slices.Clone(inherent), skippedToo); len(stale) > 0 {
		problems = append(problems, fmt.Sprintf(
			"phase(s) run that the capability baseline lists as never available: %v; update inherentSkippedPhases",
			stale,
		))
	}

	return problems
}

// noisyHeadlines names the headline metrics whose cross-run CoV reached the
// threshold. A single run has no dispersion to judge and yields nothing.
func noisyHeadlines(result *benchkit.Result) []string {
	if result.RepeatCount < 2 {
		return nil
	}

	headline := benchkit.HeadlineMetricNames()
	var noisy []string

	for _, variation := range result.MetricVariation {
		if slices.Contains(headline, variation.Name) && !variation.Reliable {
			noisy = append(noisy, fmt.Sprintf("NOISY %s (CoV %.1f%%)", variation.Name, variation.CoV*100))
		}
	}

	if len(noisy) > 0 {
		noisy = append(noisy, fmt.Sprintf(
			"medians from %d runs are not decision-grade; re-run on a quieter machine or raise --repeat",
			result.RepeatCount,
		))
	}

	return noisy
}

// strictNote warns that --strict could not judge noise: without repeats
// there is no cross-run dispersion.
func strictNote(strict bool, result *benchkit.Result) {
	if strict && result.RepeatCount < 2 {
		warnf("--strict: without --repeat 2 or more the noise gate is not evaluated " +
			"(phase coverage and integrity still are)")
	}
}

// checkStrict rejects --strict combinations that can never pass or cannot be
// judged. skipped names the --skip-* flags that are set; conflicting names
// other flags that must not be combined with --strict.
func checkStrict(strict bool, skipped, conflicting []string) error {
	if !strict {
		return nil
	}

	var errs []error

	if len(skipped) > 0 {
		errs = append(errs, fmt.Errorf("%w: --strict requires every phase to run, but %s skip phases",
			errFlagCombo, strings.Join(skipped, ", ")))
	}

	for _, flag := range conflicting {
		errs = append(errs, fmt.Errorf("%w: --strict cannot be combined with %s", errFlagCombo, flag))
	}

	return errors.Join(errs...)
}

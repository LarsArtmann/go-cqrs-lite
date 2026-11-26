package main

import (
	"context"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/benchkit/v4"
)

// runComparison benchmarks every binding with the same config, in the order
// the backends were named, and points each run at its own store directory so
// disk metrics are populated. It mirrors benchkit.Compare but injects that
// per-backend DiskPath, which benchkit.Compare cannot.
//
// A backend whose run fails is reported with the error in its Result instead
// of aborting the comparison.
func runComparison(ctx context.Context, config benchkit.Config, bindings []binding) map[string]*benchkit.Result {
	results := make(map[string]*benchkit.Result, len(bindings))

	for i, bound := range bindings {
		config.Backend = bound.name
		config.DiskPath = bound.diskPath

		if config.ProgressWriter != nil {
			fmt.Fprintf(config.ProgressWriter, "\n[%d/%d] backend: %s\n", i+1, len(bindings), bound.name)
		}

		result, err := benchkit.Run(ctx, config, bound.factory)
		if err != nil {
			results[bound.name] = failedResult(bound.name, config.Profile.Name, err)

			continue
		}

		results[bound.name] = result
	}

	return results
}

// failedResult is the Result of a backend whose run returned an error.
func failedResult(backend, profile string, err error) *benchkit.Result {
	return &benchkit.Result{
		Backend:   backend,
		Profile:   profile,
		Timestamp: time.Now(),
		Error:     err.Error(),
	}
}

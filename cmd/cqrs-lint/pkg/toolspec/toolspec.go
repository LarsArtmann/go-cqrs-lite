// Package toolspec exposes cqrs-lint as a go-finding toolsdk Spec so
// BuildFlow (or any toolsdk host) can discover and run it. The Spec lives in
// an importable package — the cqrs-lint main package cannot be imported — and
// registers itself into the default toolsdk registry on import, following the
// database/sql driver pattern.
package toolspec

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-finding"
	"github.com/larsartmann/go-finding/pipeline"
	"github.com/larsartmann/go-finding/toolsdk"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/fix"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules"
)

// Spec returns the cqrs-lint tool specification: full CQRS rule detection
// plus an all-or-nothing repair pass restricted to the fix provider's safe,
// structural rewrites (C-series correctness fixes). BuildFlow re-runs Detect
// after Repair and measures the finding delta itself. The working directory
// comes from the context (toolsdk convention), defaulting to ".".
func Spec() toolsdk.Spec {
	return toolsdk.Spec{
		Name: "cqrs-lint",
		Description: fmt.Sprintf(
			"CQRS + event-sourcing linter for Go (%d rules across 10 categories)",
			len(rules.AllRules()),
		),
		Trigger: toolsdk.Trigger{
			Files:    []string{"**/*.go"},
			Language: "go",
			Requires: []string{"go.mod", "go.work"},
		},
		Inputs: []string{"**/*.go"},
		Detect: finding.DetectorFunc(detect),
		Repair: toolsdk.RepairerFunc(func(ctx context.Context) (toolsdk.RepairResult, error) {
			fixed, err := repair(ctx, workingDir(ctx))
			if err != nil {
				return toolsdk.RepairResult{}, err
			}

			return toolsdk.RepairResult{
				Description: fmt.Sprintf(
					"applied %d safe fix(es) via the cqrs-lint fix pipeline",
					fixed,
				),
			}, nil
		}),
	}
}

// workingDir resolves the toolsdk working-directory convention, defaulting
// to "." when the context carries none.
func workingDir(ctx context.Context) string {
	if wd := finding.WorkingDirFromContext(ctx); wd != "" {
		return wd
	}

	return "."
}

// detect runs every registered rule detector against the working directory.
func detect(ctx context.Context) ([]finding.Finding, error) {
	actx, err := analyzer.BuildContext(workingDir(ctx))
	if err != nil {
		return nil, fmt.Errorf("cqrs-lint: load packages: %w", err)
	}

	var all []finding.Finding

	for _, det := range rules.RegisterAll(actx) {
		findings, err := det.Detect(ctx)
		if err != nil {
			return nil, err
		}

		all = append(all, findings...)
	}

	return all, nil
}

// repair applies the safe fix set (the same CQRSFixProvider the CLI's --fix
// uses) over one pipeline pass and returns how many findings were fixed.
func repair(ctx context.Context, wd string) (int, error) {
	actx, err := analyzer.BuildContext(wd)
	if err != nil {
		return 0, fmt.Errorf("cqrs-lint repair: load packages: %w", err)
	}

	pipe, err := pipeline.New(pipeline.Config{
		MaxIterations: 1,
		DryRun:        false,
		FixProviders:  []pipeline.FixProvider{fix.NewCQRSFixProvider()},
	}, wd, rules.RegisterAll(actx)...)
	if err != nil {
		return 0, fmt.Errorf("cqrs-lint repair: create pipeline: %w", err)
	}

	result, err := pipe.Run(ctx)
	if err != nil {
		return 0, fmt.Errorf("cqrs-lint repair: pipeline run: %w", err)
	}

	fixed := 0
	for _, iter := range result.Iterations {
		fixed += int(iter.Applied)
	}

	return fixed, nil
}

func init() {
	toolsdk.Register(Spec())
}

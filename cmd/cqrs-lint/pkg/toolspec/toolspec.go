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
			Requires: []string{"**/go.mod", "**/go.work"},
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
// The project's .cqrs-lint.json (preset + rules block) is honored exactly as
// the CLI honors it: rule-specific config flows to detectors via
// AnalysisContext.RulesConfig, and disabled rules are filtered post-detection.
func detect(ctx context.Context) ([]finding.Finding, error) {
	wd := workingDir(ctx)

	actx, err := analyzer.BuildContext(wd)
	if err != nil {
		return nil, fmt.Errorf("cqrs-lint: load packages: %w", err)
	}

	effective, err := loadEffectiveRules(wd)
	if err != nil {
		return nil, err
	}

	actx.RulesConfig = effective

	var all []finding.Finding

	for _, det := range rules.RegisterAll(actx) {
		findings, err := det.Detect(ctx)
		if err != nil {
			return nil, err
		}

		all = append(all, findings...)
	}

	all = analyzer.ApplySeverityOverrides(all, effective.SeverityOverrides)

	return analyzer.FilterDisabledFindings(all, effective.DisabledSet()), nil
}

// loadEffectiveRules resolves the project's .cqrs-lint.json (preset + rules)
// against wd. A missing config means "no overrides" — identical behavior to a
// project without a config file. A malformed config or unknown preset is an
// error: silently linting unconfigured would make the config file a no-op lie.
func loadEffectiveRules(wd string) (analyzer.RulesConfig, error) {
	cfg, found, err := analyzer.LoadProjectConfig(wd)
	if err != nil {
		return analyzer.RulesConfig{}, fmt.Errorf("cqrs-lint: %w", err)
	}

	if !found {
		return analyzer.RulesConfig{}, nil
	}

	return cfg.EffectiveRules(), nil
}

// repair applies the safe fix set (the same CQRSFixProvider the CLI's --fix
// uses) over one pipeline pass and returns how many findings were fixed. The
// project's .cqrs-lint.json rules config applies to the fix pipeline too, so
// config-driven rule behavior (e.g. c008 ignore lists) matches detection.
func repair(ctx context.Context, wd string) (int, error) {
	actx, err := analyzer.BuildContext(wd)
	if err != nil {
		return 0, fmt.Errorf("cqrs-lint repair: load packages: %w", err)
	}

	effective, err := loadEffectiveRules(wd)
	if err != nil {
		return 0, fmt.Errorf("cqrs-lint repair: %w", err)
	}

	actx.RulesConfig = effective

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
		fixed += iter.Applied
	}

	return fixed, nil
}

func init() {
	toolsdk.Register(Spec())
}

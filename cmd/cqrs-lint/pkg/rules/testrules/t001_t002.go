package testrules

import (
	"context"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// T001: No scenario tests for deciders.
// Detects projects that use the decider module but have no scenario.Given
// calls in test files. BDD-style scenario tests verify decide/fold behavior
// in one fluent chain and are the recommended way to test deciders.
//
//nolint:ireturn // factory returns public interface
func NewT001Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"T001-no-scenario-tests-for-deciders",
		func(_ context.Context) ([]finding.Finding, error) {
			if !anyProdFileImports(ctx, "/decider") {
				return nil, nil
			}

			if !hasTestFiles(ctx) {
				return nil, nil
			}

			if testFileCallsPkgFunc(ctx, "scenario", "Given") {
				return nil, nil
			}

			return []finding.Finding{
				projectFinding(
					"T001",
					"Project defines deciders but has no scenario.Given tests — deciders lack BDD-style behavioral tests",
					"Use scenario.Given[T,S](t, fold, initial, events...).When(cmd, decide).Then(expectedEvents...) for decider testing",
					finding.SeverityInfo,
					finding.ConfidenceMedium,
					ctx,
				),
			}, nil
		},
	)
}

// T002: No scenario tests for projections.
// Detects projects that define projections but have no scenario.GivenProjection
// calls in test files. Projections should be tested for correct event handling.
//
//nolint:ireturn // factory returns public interface
func NewT002Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"T002-no-scenario-tests-for-projections",
		func(_ context.Context) ([]finding.Finding, error) {
			hasProjections := len(ctx.Registry.Projections) > 0 ||
				anyProdFileImports(ctx, "/projection")

			if !hasProjections {
				return nil, nil
			}

			if !hasTestFiles(ctx) {
				return nil, nil
			}

			if testFileCallsPkgFunc(ctx, "scenario", "GivenProjection") {
				return nil, nil
			}

			return []finding.Finding{
				projectFinding(
					"T002",
					"Project defines projections but has no scenario.GivenProjection tests — projections lack behavioral tests",
					"Use scenario.GivenProjection(t, proj, evt1, evt2...).ThenNoError() to verify projections handle events correctly",
					finding.SeverityInfo,
					finding.ConfidenceMedium,
					ctx,
				),
			}, nil
		},
	)
}

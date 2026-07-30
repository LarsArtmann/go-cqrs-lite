package testrules

import (
	"context"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// T005: Projection without error-handling test.
// Detects projects that define projections but have no tests exercising
// error paths (ThenError). Projections must gracefully handle malformed
// payloads and unknown event types.
//
//nolint:ireturn // factory returns public interface
func NewT005Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"T005-projection-without-error-test",
		func(_ context.Context) ([]finding.Finding, error) {
			hasProjections := len(ctx.Registry.Projections) > 0 ||
				anyProdFileImports(ctx, "/projection")

			if !hasProjections {
				return nil, nil
			}

			if !hasTestFiles(ctx) {
				return nil, nil
			}

			if testFileCallsMethod(ctx, "ThenError") {
				return nil, nil
			}

			return []finding.Finding{
				projectFinding(
					"T005",
					"Projections have no error-path tests — malformed payloads and unknown event types are untested",
					"Add scenario.GivenProjection tests with ThenError, or test your projection with malformed payloads and unknown event types",
					finding.SeverityInfo,
					finding.ConfidenceLow,
					ctx,
				),
			}, nil
		},
	)
}

// T006: Decider test without conflict-path test.
// Detects projects that use scenario.Given for decider testing but only
// assert happy paths (Then) without testing conflict/error paths (ThenError).
// Testing both success and conflict paths catches edge-case regressions.
//
//nolint:ireturn // factory returns public interface
func NewT006Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"T006-decider-without-conflict-test",
		func(_ context.Context) ([]finding.Finding, error) {
			if !testFileCallsPkgFunc(ctx, "scenario", "Given") {
				return nil, nil
			}

			if testFileCallsMethod(ctx, "ThenError") {
				return nil, nil
			}

			return []finding.Finding{
				projectFinding(
					"T006",
					"Decider scenario tests only cover happy paths — no ThenError assertions for conflict/error scenarios",
					"Add scenario.Given(...).When(conflictCmd, decide).ThenError(targetError) to test rejection and conflict paths",
					finding.SeverityInfo,
					finding.ConfidenceMedium,
					ctx,
				),
			}, nil
		},
	)
}

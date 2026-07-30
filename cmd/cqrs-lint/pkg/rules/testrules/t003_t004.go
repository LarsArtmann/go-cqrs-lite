package testrules

import (
	"context"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// T003: No eventtest imports.
// Detects projects that use the event module (event stores, events) but do
// not import event/v4/eventtest for fake stores/buses in tests. Manual test
// setup with hand-rolled mocks is error-prone.
//
//nolint:ireturn // factory returns public interface
func NewT003Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"T003-no-eventtest-imports",
		func(_ context.Context) ([]finding.Finding, error) {
			if !anyFileImports(ctx, "/event/v4") {
				return nil, nil
			}

			if !hasTestFiles(ctx) {
				return nil, nil
			}

			if anyFileImports(ctx, "eventtest") {
				return nil, nil
			}

			return []finding.Finding{
				projectFinding(
					"T003",
					"Project uses events but does not import eventtest — tests lack fake stores/buses from the library",
					"Import event/v4/eventtest for FakeStore, FakeBus, event factories, and test assertions instead of hand-rolling mocks",
					finding.SeverityInfo,
					finding.ConfidenceMedium,
					ctx,
				),
			}, nil
		},
	)
}

// T004: No golden/snapshot tests.
// Detects projects that use catalog (documentation/schema generation) but do
// not import go-snaps for snapshot testing. Generated output should be
// snapshot-tested to catch undocumented changes.
//
//nolint:ireturn // factory returns public interface
func NewT004Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"T004-no-golden-snapshot-tests",
		func(_ context.Context) ([]finding.Finding, error) {
			if !anyFileImports(ctx, "catalog") {
				return nil, nil
			}

			if !hasTestFiles(ctx) {
				return nil, nil
			}

			if anyFileImports(ctx, "snaps") || anyFileImports(ctx, "go-snaps") {
				return nil, nil
			}

			return []finding.Finding{
				projectFinding(
					"T004",
					"Project uses catalog for documentation generation but has no snapshot tests — generated output changes go undetected",
					"Import github.com/gkampitakis/go-snaps and use snaps.MatchSnapshot to pin generated catalog/schema output",
					finding.SeverityInfo,
					finding.ConfidenceLow,
					ctx,
				),
			}, nil
		},
	)
}

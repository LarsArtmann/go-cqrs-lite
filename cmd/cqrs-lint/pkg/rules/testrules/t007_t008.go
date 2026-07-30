package testrules

import (
	"context"
	"fmt"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

// T007: No integration test for event round-trip.
// Detects projects that use events/event stores but have no test exercising
// the save→load round-trip. Round-trip tests catch serialization, ordering,
// and version-increment bugs that unit tests miss.
//
//nolint:ireturn // factory returns public interface
func NewT007Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"T007-no-event-roundtrip-test",
		func(_ context.Context) ([]finding.Finding, error) {
			if !anyFileImports(ctx, "/event") {
				return nil, nil
			}

			if !hasTestFiles(ctx) {
				return nil, nil
			}

			if testFilesCallBoth(ctx, "Save", "Load") {
				return nil, nil
			}

			return []finding.Finding{
				projectFinding(
					"T007",
					"Project uses event stores but has no save→load round-trip test — serialization and ordering bugs go undetected",
					"Add a test that calls store.Save then store.Load on the same stream and asserts event count, types, and version ordering",
					finding.SeverityInfo,
					finding.ConfidenceLow,
					ctx,
				),
			}, nil
		},
	)
}

// productionStoreSubstrings lists import-path substrings that identify
// production (non-test) event store backends. Importing these in _test.go
// files couples tests to a real database instead of using eventtest fakes.
var productionStoreSubstrings = []string{
	"go-cqrs-lite/storage/v4",
	"go-cqrs-lite/storage/turso",
	"go-cqrs-lite/storage/pebble",
	"go-cqrs-lite/stack/sqlite",
	"go-cqrs-lite/stack/pebble",
	"go-cqrs-lite/stack/postgres",
	"go-cqrs-lite/stack/turso",
	"go-cqrs-lite/stack/duckdb",
}

// T008: Test files import production event store.
// Detects test files that import production storage backends (SQL, Pebble,
// Turso, stack presets) instead of test utilities (eventtest.FakeStore,
// storage/memory.MemoryStore). Production stores slow tests and add
// external dependencies.
//
//nolint:ireturn // factory returns public interface
func NewT008Detector(ctx *analyzer.AnalysisContext) finding.Detector {
	return finding.NamedDetectorFunc(
		"T008-test-imports-production-store",
		func(_ context.Context) ([]finding.Finding, error) {
			var findings []finding.Finding

			for _, gf := range ctx.GoFiles {
				if !gf.IsTest {
					continue
				}

				for _, imp := range gf.AST.Imports {
					if imp == nil || imp.Path == nil {
						continue
					}

					path := strings.Trim(imp.Path.Value, `"`)
					matched := ""

					for _, pattern := range productionStoreSubstrings {
						if strings.Contains(path, pattern) {
							matched = pattern

							break
						}
					}

					if matched == "" {
						continue
					}

					pos := ctx.Fset.Position(imp.Pos())

					f, err := finding.NewBuilder(
						"T008", toolName,
						fmt.Sprintf(
							"Test file imports production store %q — use eventtest.FakeStore or storage/memory.MemoryStore instead",
							path,
						),
						finding.SeverityWarning,
						finding.Pos(finding.FilePath(pos.Filename), pos.Line, pos.Column),
					).
						WithCategory(finding.CategoryTesting).
						WithConfidence(finding.ConfidenceHigh).
						WithSuggestion(
							"Import event/v4/eventtest for FakeStore/FakeBus, or storage/memory for MemoryStore — production backends slow tests and add external dependencies",
						).
						WithSnippet(ctx.SourceLine(pos.Filename, pos.Line)).
						Build()
					if err == nil {
						findings = append(findings, f)
					}
				}
			}

			return findings, nil
		},
	)
}

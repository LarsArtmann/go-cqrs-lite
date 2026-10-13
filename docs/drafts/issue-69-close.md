Shipped the suggested shape on master: analyzer.CollectTestImportPaths (cmd/cqrs-lint/pkg/analyzer/test_imports.go:27) walks the module tree — vendor/testdata/.git skipped — parsing _test.go files with go/parser ImportsOnly, and DetectUsedModules grew a third pass matching those paths through the same matchModule/pathBoundaryMatch. The heavy typed load stays Tests: false.

Verified against this issue's repro: cqrs-lint scorecard over go-credentials-store @ master now reports System Scenario Testing: used, evidence internal/boot/scenario_test.go's systemscenario import. Scenario Testing stays missing there — that consumer does not import scenario, correctly.

Pinned by TestDetectUsedModules_TestOnlyImports + TestScorecard_TestOnlyImport_Integration (the exact Collect → Detect → Compute pipeline over a disk fixture whose only cqrs import lives in a _test.go file).

One transition note: if you recorded a waiver against a row this fix turns USED, the existing waive-used hard error fires — delete the stale waiver, the error message says exactly that.

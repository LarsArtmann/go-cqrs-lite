package correctness_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/correctness"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
)

// TestC036_PerModuleProfileEvaluatesByModule verifies that C036 compares
// secondary stores against the FILE'S module backend, not the primary profile.
// A Pebble checkpoint store in a SQLite-backed module should fire even when
// the primary profile says Pebble (from a different module).
func TestC036_PerModuleProfileEvaluatesByModule(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"/repo/lib/setup.go": `package main

func setup() {
	cpStore, _ := pebble.NewCheckpointStore(db)
	_ = cpStore
}
`,
	})

	// Multi-module: lib uses SQLite, app uses Pebble.
	ctx.FeatureProfiles = map[string]analyzer.FeatureProfile{
		"/repo/lib":          {Store: analyzer.StoreSQLite},
		"/repo/examples/app": {Store: analyzer.StorePebble},
	}
	// Primary profile reflects the app module (Pebble).
	ctx.FeatureProfile = analyzer.FeatureProfile{Store: analyzer.StorePebble}

	findings := ruletest.RunDetector(t, correctness.NewC036Detector(ctx))

	// The file is in /repo/lib (SQLite). A Pebble checkpoint store mismatches
	// SQLite — should fire. With the old primary-profile behavior, it would
	// NOT fire because the primary profile says Pebble.
	ruletest.AssertRule(t, findings, "C036", 1)
}

// TestC036_PerModuleProfileSameBackendNoMismatch verifies that a secondary
// store matching the file's own module backend does NOT fire, even when the
// primary profile says a different backend.
func TestC036_PerModuleProfileSameBackendNoMismatch(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"/repo/app/setup.go": `package main

func setup() {
	cpStore, _ := pebble.NewCheckpointStore(db)
	_ = cpStore
}
`,
	})

	ctx.FeatureProfiles = map[string]analyzer.FeatureProfile{
		"/repo/lib": {Store: analyzer.StoreSQLite},
		"/repo/app": {Store: analyzer.StorePebble},
	}
	// Primary profile = SQLite (from the lib module).
	ctx.FeatureProfile = analyzer.FeatureProfile{Store: analyzer.StoreSQLite}

	findings := ruletest.RunDetector(t, correctness.NewC036Detector(ctx))

	// File is in /repo/app (Pebble). Pebble checkpoint matches Pebble module.
	// Should NOT fire, even though primary says SQLite.
	ruletest.AssertRule(t, findings, "C036", 0)
}

// TestC036_PerModuleProfileSkipsNonPersistentModule verifies that a file in
// a module with a memory store is skipped entirely (no mismatch check).
func TestC036_PerModuleProfileSkipsNonPersistentModule(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"/repo/lib/setup.go": `package main

func setup() {
	cpStore, _ := pebble.NewCheckpointStore(db)
	_ = cpStore
}
`,
	})

	ctx.FeatureProfiles = map[string]analyzer.FeatureProfile{
		"/repo/lib":          {Store: analyzer.StoreMemory},
		"/repo/examples/app": {Store: analyzer.StoreSQLite},
	}
	ctx.FeatureProfile = analyzer.FeatureProfile{Store: analyzer.StoreSQLite}

	findings := ruletest.RunDetector(t, correctness.NewC036Detector(ctx))

	// File is in /repo/lib (Memory). Memory store → skip (no mismatch possible).
	ruletest.AssertRule(t, findings, "C036", 0)
}

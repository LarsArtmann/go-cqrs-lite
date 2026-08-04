package correctness_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/correctness"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
)

func TestC036_BackendMismatch(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup() {
	cpStore, _ := pebble.NewCheckpointStore(db)
	_ = cpStore
}
`,
	})
	ctx.FeatureProfile.Store = analyzer.StoreSQLite

	findings := ruletest.RunDetector(t, correctness.NewC036Detector(ctx))
	ruletest.AssertRule(t, findings, "C036", 1)
}

func TestC036_SameBackendNoMismatch(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup() {
	cpStore, _ := pebble.NewCheckpointStore(db)
	_ = cpStore
}
`,
	})
	ctx.FeatureProfile.Store = analyzer.StorePebble

	findings := ruletest.RunDetector(t, correctness.NewC036Detector(ctx))
	ruletest.AssertRule(t, findings, "C036", 0)
}

func TestC036_MemoryEventStoreNoCheck(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

func setup() {
	cpStore, _ := pebble.NewCheckpointStore(db)
	_ = cpStore
}
`,
	})
	ctx.FeatureProfile.Store = analyzer.StoreMemory

	findings := ruletest.RunDetector(t, correctness.NewC036Detector(ctx))
	ruletest.AssertRule(t, findings, "C036", 0)
}

func TestC036_LibraryUtilityNotFlagged(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

import storage "github.com/larsartmann/go-cqrs-lite/storage/v4"

func setup() {
	backend, _ := storage.NewSQLiteBackend(db)
	_ = backend
}
`,
	})
	ctx.FeatureProfile.Store = analyzer.StorePostgres

	findings := ruletest.RunDetector(t, correctness.NewC036Detector(ctx))
	ruletest.AssertRule(t, findings, "C036", 0)
}

func TestC036_RealSecondaryStoreMismatchFlagged(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"setup.go": `package main

import storage "github.com/larsartmann/go-cqrs-lite/storage/v4"

func setup() {
	cpStore, _ := storage.NewSQLiteCheckpointStore(db)
	_ = cpStore
}
`,
	})
	ctx.FeatureProfile.Store = analyzer.StorePostgres

	findings := ruletest.RunDetector(t, correctness.NewC036Detector(ctx))
	ruletest.AssertRule(t, findings, "C036", 1)
}

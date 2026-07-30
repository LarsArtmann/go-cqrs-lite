package version_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/version"
)

// writeGoMod creates a temp project root with a go.mod file and returns the
// path. The caller does not need to clean up (t.TempDir handles it).
func writeGoMod(t *testing.T, content string) string {
	t.Helper()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(content), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	return dir
}

func ctxWithGoMod(t *testing.T, goModContent string) *analyzer.AnalysisContext {
	t.Helper()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import "github.com/larsartmann/go-cqrs-lite/event/v4"

var _ = event.New
`,
	})
	ctx.ProjectRoot = writeGoMod(t, goModContent)
	return ctx
}

// --- V002: unpinned-version ---

func TestV002_DetectsPseudoVersion(t *testing.T) {
	t.Parallel()

	ctx := ctxWithGoMod(t, `module example.com/app

go 1.26.4

require (
	github.com/larsartmann/go-cqrs-lite/event/v4 v0.0.0-00010101000000-000000000000
)
`)
	findings := runDetector(t, version.NewV002Detector(ctx))
	assertRule(t, findings, "V002", 1)
}

func TestV002_NoFindingForTaggedRelease(t *testing.T) {
	t.Parallel()

	ctx := ctxWithGoMod(t, `module example.com/app

go 1.26.4

require (
	github.com/larsartmann/go-cqrs-lite/event/v4 v4.2.0
)
`)
	findings := runDetector(t, version.NewV002Detector(ctx))
	assertRule(t, findings, "V002", 0)
}

func TestV002_NoProjectRoot(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"a.go": `package main

import "github.com/larsartmann/go-cqrs-lite/event/v4"
`,
	})
	findings := runDetector(t, version.NewV002Detector(ctx))
	assertRule(t, findings, "V002", 0)
}

// --- V003: version-lag ---

func TestV003_DetectsLag(t *testing.T) {
	t.Parallel()

	ctx := ctxWithGoMod(t, `module example.com/app

go 1.26.4

require (
	github.com/larsartmann/go-cqrs-lite/event/v4 v4.0.0
)
`)
	findings := runDetector(t, version.NewV003Detector(ctx))
	assertRule(t, findings, "V003", 1)
}

func TestV003_NoFindingForRecentVersion(t *testing.T) {
	t.Parallel()

	ctx := ctxWithGoMod(t, `module example.com/app

go 1.26.4

require (
	github.com/larsartmann/go-cqrs-lite/event/v4 v4.3.0
)
`)
	findings := runDetector(t, version.NewV003Detector(ctx))
	assertRule(t, findings, "V003", 0)
}

func TestV003_NoFindingForIndirect(t *testing.T) {
	t.Parallel()

	ctx := ctxWithGoMod(t, `module example.com/app

go 1.26.4

require (
	github.com/larsartmann/go-cqrs-lite/event/v4 v4.0.0 // indirect
)
`)
	findings := runDetector(t, version.NewV003Detector(ctx))
	assertRule(t, findings, "V003", 0)
}

// --- V004: vendored-third-party ---

func TestV004_DetectsThirdPartyCQRS(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"third_party/go-cqrs-lite-eventtest/eventtest.go": `package eventtest

import "github.com/larsartmann/go-cqrs-lite/event/v4"

var _ = event.New
`,
	})
	findings := runDetector(t, version.NewV004Detector(ctx))
	assertRule(t, findings, "V004", 1)
}

func TestV004_NoFindingForRegularPath(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import "github.com/larsartmann/go-cqrs-lite/event/v4"

var _ = event.New
`,
	})
	findings := runDetector(t, version.NewV004Detector(ctx))
	assertRule(t, findings, "V004", 0)
}

func TestV004_NoFindingForThirdPartyWithoutCQRS(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"third_party/somelib/lib.go": `package somelib

import "fmt"

var _ = fmt.Println
`,
	})
	findings := runDetector(t, version.NewV004Detector(ctx))
	assertRule(t, findings, "V004", 0)
}

// --- V005: eventtest-vendored-mismatch ---

func TestV005_DetectsVendoredEventtest(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import "github.com/larsartmann/go-cqrs-lite/event/v4"

var _ = event.New
`,
		"third_party/eventtest/fake.go": `package eventtest

type FakeStore struct{}
`,
	})
	findings := runDetector(t, version.NewV005Detector(ctx))
	assertRule(t, findings, "V005", 1)
}

func TestV005_NoFindingWithoutCQRSImports(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import "fmt"

var _ = fmt.Println
`,
		"third_party/eventtest/fake.go": `package eventtest

type FakeStore struct{}
`,
	})
	findings := runDetector(t, version.NewV005Detector(ctx))
	assertRule(t, findings, "V005", 0)
}

func TestV005_NoFindingForRegularEventtest(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import "github.com/larsartmann/go-cqrs-lite/event/v4"

var _ = event.New
`,
		"eventtest/fake.go": `package eventtest

type FakeStore struct{}
`,
	})
	findings := runDetector(t, version.NewV005Detector(ctx))
	assertRule(t, findings, "V005", 0)
}

// --- V006: mixed-version-pins ---

func TestV006_DetectsMixedVersions(t *testing.T) {
	t.Parallel()

	ctx := ctxWithGoMod(t, `module example.com/app

go 1.26.4

require (
	github.com/larsartmann/go-cqrs-lite/event/v4 v4.2.0
	github.com/larsartmann/go-cqrs-lite/decider/v4 v4.1.0
)
`)
	findings := runDetector(t, version.NewV006Detector(ctx))
	assertRule(t, findings, "V006", 1)
}

func TestV006_NoFindingForConsistentVersions(t *testing.T) {
	t.Parallel()

	ctx := ctxWithGoMod(t, `module example.com/app

go 1.26.4

require (
	github.com/larsartmann/go-cqrs-lite/event/v4 v4.2.0
	github.com/larsartmann/go-cqrs-lite/decider/v4 v4.2.0
)
`)
	findings := runDetector(t, version.NewV006Detector(ctx))
	assertRule(t, findings, "V006", 0)
}

func TestV006_NoFindingForSingleModule(t *testing.T) {
	t.Parallel()

	ctx := ctxWithGoMod(t, `module example.com/app

go 1.26.4

require (
	github.com/larsartmann/go-cqrs-lite/event/v4 v4.2.0
)
`)
	findings := runDetector(t, version.NewV006Detector(ctx))
	assertRule(t, findings, "V006", 0)
}

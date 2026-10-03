package version_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/version"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/ruletest"
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
	findings := ruletest.RunDetector(t, version.NewV002Detector(ctx))
	ruletest.AssertRule(t, findings, "V002", 1)
}

func TestV002_NoFindingForTaggedRelease(t *testing.T) {
	t.Parallel()

	ctx := ctxWithGoMod(t, `module example.com/app

go 1.26.4

require (
	github.com/larsartmann/go-cqrs-lite/event/v4 v4.2.0
)
`)
	findings := ruletest.RunDetector(t, version.NewV002Detector(ctx))
	ruletest.AssertRule(t, findings, "V002", 0)
}

func TestV002_NoProjectRoot(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"a.go": `package main

import "github.com/larsartmann/go-cqrs-lite/event/v4"
`,
	})
	findings := ruletest.RunDetector(t, version.NewV002Detector(ctx))
	ruletest.AssertRule(t, findings, "V002", 0)
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
	findings := ruletest.RunDetector(t, version.NewV003Detector(ctx))
	ruletest.AssertRule(t, findings, "V003", 1)
}

func TestV003_NoFindingForRecentVersion(t *testing.T) {
	t.Parallel()

	ctx := ctxWithGoMod(t, `module example.com/app

go 1.26.4

require (
	github.com/larsartmann/go-cqrs-lite/event/v4 v4.3.0
)
`)
	findings := ruletest.RunDetector(t, version.NewV003Detector(ctx))
	ruletest.AssertRule(t, findings, "V003", 0)
}

func TestV003_NoFindingForIndirect(t *testing.T) {
	t.Parallel()

	ctx := ctxWithGoMod(t, `module example.com/app

go 1.26.4

require (
	github.com/larsartmann/go-cqrs-lite/event/v4 v4.0.0 // indirect
)
`)
	findings := ruletest.RunDetector(t, version.NewV003Detector(ctx))
	ruletest.AssertRule(t, findings, "V003", 0)
}

// TestV003_BoundaryLagTwoStaysSilent pins the lag threshold boundary: a
// one-minor gap below the threshold must not fire (lag == 2 is tolerated;
// lag == 3 fires in TestV003_DetectsLag).
func TestV003_BoundaryLagTwoStaysSilent(t *testing.T) {
	t.Parallel()

	ctx := ctxWithGoMod(t, `module example.com/app

go 1.26.4

require (
	github.com/larsartmann/go-cqrs-lite/event/v4 v4.1.0
)
`)
	findings := ruletest.RunDetector(t, version.NewV003Detector(ctx))
	ruletest.AssertRule(t, findings, "V003", 0)
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
	findings := ruletest.RunDetector(t, version.NewV004Detector(ctx))
	ruletest.AssertRule(t, findings, "V004", 1)
}

func TestV004_NoFindingForRegularPath(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import "github.com/larsartmann/go-cqrs-lite/event/v4"

var _ = event.New
`,
	})
	findings := ruletest.RunDetector(t, version.NewV004Detector(ctx))
	ruletest.AssertRule(t, findings, "V004", 0)
}

func TestV004_NoFindingForThirdPartyWithoutCQRS(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"third_party/somelib/lib.go": `package somelib

import "fmt"

var _ = fmt.Println
`,
	})
	findings := ruletest.RunDetector(t, version.NewV004Detector(ctx))
	ruletest.AssertRule(t, findings, "V004", 0)
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
	findings := ruletest.RunDetector(t, version.NewV005Detector(ctx))
	ruletest.AssertRule(t, findings, "V005", 1)
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
	findings := ruletest.RunDetector(t, version.NewV005Detector(ctx))
	ruletest.AssertRule(t, findings, "V005", 0)
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
	findings := ruletest.RunDetector(t, version.NewV005Detector(ctx))
	ruletest.AssertRule(t, findings, "V005", 0)
}

// --- V006: divergent-module-pins ---

// ctxWithGoMods writes a temp project root with the given files (path →
// content, relative to the root) and returns an AnalysisContext over it.
func ctxWithGoMods(t *testing.T, files map[string]string) *analyzer.AnalysisContext {
	t.Helper()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import "github.com/larsartmann/go-cqrs-lite/event/v4"

var _ = event.New
`,
	})

	root := t.TempDir()
	for rel, content := range files {
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", rel, err)
		}

		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}

	ctx.ProjectRoot = root

	return ctx
}

// TestV006_NoFindingForPerModuleVersionDifferences pins the per-module-tags
// semantics: different minor/patch versions ACROSS modules in one go.mod are
// the ecosystem's normal shape (tags are per-module; no unified release
// exists) and must NOT fire (CV feedback, 2026-10-03).
func TestV006_NoFindingForPerModuleVersionDifferences(t *testing.T) {
	t.Parallel()

	ctx := ctxWithGoMod(t, `module example.com/app

go 1.26.4

require (
	github.com/larsartmann/go-cqrs-lite/event/v4 v4.12.0
	github.com/larsartmann/go-cqrs-lite/decider/v4 v4.7.0
	github.com/larsartmann/go-cqrs-lite/system/v4 v4.10.0
)
`)
	findings := ruletest.RunDetector(t, version.NewV006Detector(ctx))
	ruletest.AssertRule(t, findings, "V006", 0)
}

// TestV006_DetectsCrossModuleDivergentPin pins the workspace-lockstep class:
// the SAME module path pinned at different versions in different go.mod
// files must fire, anchored on the stale (lowest) pin.
func TestV006_DetectsCrossModuleDivergentPin(t *testing.T) {
	t.Parallel()

	ctx := ctxWithGoMods(t, map[string]string{
		"go.mod": `module example.com/app

go 1.26.4

require github.com/larsartmann/go-cqrs-lite/event/v4 v4.10.0
`,
		"sub/go.mod": `module example.com/app/sub

go 1.26.4

require github.com/larsartmann/go-cqrs-lite/event/v4 v4.9.0
`,
	})
	findings := ruletest.RunDetector(t, version.NewV006Detector(ctx))
	ruletest.AssertRule(t, findings, "V006", 1)

	for _, f := range findings {
		if !strings.Contains(f.Message, "event/v4 is pinned at v4.9.0") {
			t.Errorf("finding must anchor on the stale pin v4.9.0, got: %s", f.Message)
		}

		if !strings.Contains(f.Message, "sub/go.mod") {
			t.Errorf("finding must name the divergent file, got: %s", f.Message)
		}

		if !strings.Contains(f.Suggestion, "@v4.10.0") {
			t.Errorf("align suggestion must target v4.10.0, got: %s", f.Suggestion)
		}
	}
}

// TestV006_NoFindingForConsistentVersions: the same module pinned at the
// same version across two go.mod files is the aligned (healthy) shape.
func TestV006_NoFindingForConsistentVersions(t *testing.T) {
	t.Parallel()

	ctx := ctxWithGoMods(t, map[string]string{
		"go.mod": `module example.com/app

go 1.26.4

require github.com/larsartmann/go-cqrs-lite/event/v4 v4.2.0
`,
		"sub/go.mod": `module example.com/app/sub

go 1.26.4

require github.com/larsartmann/go-cqrs-lite/event/v4 v4.2.0
`,
	})
	findings := ruletest.RunDetector(t, version.NewV006Detector(ctx))
	ruletest.AssertRule(t, findings, "V006", 0)
}

func TestV006_NoFindingForSingleModule(t *testing.T) {
	t.Parallel()

	ctx := ctxWithGoMod(t, `module example.com/app

go 1.26.4

require (
	github.com/larsartmann/go-cqrs-lite/event/v4 v4.2.0
)
`)
	findings := ruletest.RunDetector(t, version.NewV006Detector(ctx))
	ruletest.AssertRule(t, findings, "V006", 0)
}

// TestV006_MultiDigitMinorAnchorsLowestVersion is the semver-ordering
// regression: lexicographic sorting ranks v4.10.0 BELOW v4.9.0. With numeric
// ordering the cross-file finding must anchor on v4.9.0 and suggest v4.10.0.
func TestV006_MultiDigitMinorAnchorsLowestVersion(t *testing.T) {
	t.Parallel()

	ctx := ctxWithGoMods(t, map[string]string{
		"go.mod": `module example.com/app

go 1.26.4

require github.com/larsartmann/go-cqrs-lite/event/v4 v4.10.0
`,
		"sub/go.mod": `module example.com/app/sub

go 1.26.4

require github.com/larsartmann/go-cqrs-lite/event/v4 v4.9.0
`,
	})
	findings := ruletest.RunDetector(t, version.NewV006Detector(ctx))
	ruletest.AssertRule(t, findings, "V006", 1)

	for _, f := range findings {
		if !strings.Contains(f.Message, "event/v4 is pinned at v4.9.0") {
			t.Errorf("finding must anchor on the lowest version v4.9.0, got: %s", f.Message)
		}

		if !strings.Contains(f.Suggestion, "@v4.10.0") {
			t.Errorf("align suggestion must target v4.10.0, got: %s", f.Suggestion)
		}
	}
}

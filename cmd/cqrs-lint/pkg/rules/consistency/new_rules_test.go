package consistency_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/consistency"
)

// --- D001: Inconsistent event naming ---

func TestD001_DetectsMixedNaming(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

func emitEvents() {
	_ = event.New("user.created", id, "User", 1, payload)
	_ = event.New("UserDeleted", id, "User", 2, payload)
}
`,
	})
	findings := runDetector(t, consistency.NewD001Detector(ctx))
	assertRule(t, findings, "D001", 1)
}

func TestD001_NoFindingForConsistentNaming(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

func emitEvents() {
	_ = event.New("user.created", id, "User", 1, payload)
	_ = event.New("user.deleted", id, "User", 2, payload)
}
`,
	})
	findings := runDetector(t, consistency.NewD001Detector(ctx))
	assertRule(t, findings, "D001", 0)
}

// --- D003: Inconsistent logging library ---

func TestD003_NoCrashOnEmptyContext(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	findings := runDetector(t, consistency.NewD003Detector(ctx))
	assertRule(t, findings, "D003", 0)
}

// --- D003: Positive test — mixed logging libraries ---

func TestD003_DetectsMixedLogging(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"a.go": `package main

import (
	"log/slog"
)

var _ = slog.Default
`,
		"b.go": `package main

import (
	"go.uber.org/zap"
)

var _ = zap.New
`,
	})
	findings := runDetector(t, consistency.NewD003Detector(ctx))
	assertRule(t, findings, "D003", 1)
}

// --- D005: Positive test — stale documentation version ---

func TestD005_DetectsStaleDocVersion(t *testing.T) {
	tmpDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(
		"module example.com/app\n\nrequire github.com/larsartmann/go-cqrs-lite v4.2.0\n",
	),
		0o644)
	_ = os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte(
		"# My App\n\nUses go-cqrs-lite v3.1.0\n",
	),
		0o644)

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	ctx.ProjectRoot = tmpDir

	findings := runDetector(t, consistency.NewD005Detector(ctx))
	assertRule(t, findings, "D005", 1)
}

// --- D005: Wildcard version (v4.0.x) is compatible with v4.0.0

func TestD005_NoFindingForWildcardVersion(t *testing.T) {
	tmpDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(
		"module example.com/app\n\nrequire github.com/larsartmann/go-cqrs-lite v4.0.0\n",
	),
		0o644)
	_ = os.WriteFile(filepath.Join(tmpDir, "AGENTS.md"), []byte(
		"# Project\n\nUses go-cqrs-lite v4.0.x\n",
	),
		0o644)

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	ctx.ProjectRoot = tmpDir

	findings := runDetector(t, consistency.NewD005Detector(ctx))
	assertRule(t, findings, "D005", 0)
}

// --- D005: Migration arrow (v2→v3) in ADR context is NOT flagged

func TestD005_NoFindingForMigrationArrow(t *testing.T) {
	tmpDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(
		"module example.com/app\n\nrequire github.com/larsartmann/go-cqrs-lite v4.0.0\n",
	),
		0o644)
	_ = os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte(
		"| [007](docs/adr/007.md) | go-cqrs-lite v3 Migration | v2→v3, memory → watermill |\n",
	),
		0o644)

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	ctx.ProjectRoot = tmpDir

	findings := runDetector(t, consistency.NewD005Detector(ctx))
	assertRule(t, findings, "D005", 0)
}

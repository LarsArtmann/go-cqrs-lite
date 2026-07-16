package consistency_test

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/tools/go/packages"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/consistency"
)

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
		"main.go": `package main`,
	})
	ctx.Packages = []*packages.Package{
		{
			PkgPath: "example.com/app",
			Imports: map[string]*packages.Package{
				"log/slog":        {PkgPath: "log/slog"},
				"go.uber.org/zap": {PkgPath: "go.uber.org/zap"},
			},
		},
	}
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

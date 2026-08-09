package main

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules"
)

// TestLintExampleTaskmanager runs all cqrs-lint detectors against the
// example/taskmanager reference project. This verifies end-to-end behavior
// on a real CQRS/ES application with HTTP server, projections, middleware,
// signing, and metaengine integration.
//
// The test asserts that the taskmanager produces no critical findings — it
// is the canonical reference that consumers copy from, so it must be clean.
//
// Update the golden finding list with CQRS_LINT_UPDATE_GOLDEN=1.
func TestLintExampleTaskmanager(t *testing.T) {
	t.Parallel()

	tmDir := filepath.Join("..", "..", "example", "taskmanager")

	if _, err := os.Stat(filepath.Join(tmDir, "go.mod")); err != nil {
		t.Skipf("example/taskmanager not found at %s: %v", tmDir, err)
	}

	ctx, err := analyzer.BuildContext(tmDir)
	if err != nil {
		t.Fatalf("BuildContext(%s): %v", tmDir, err)
	}

	detectors := rules.RegisterAll(ctx)

	var allFindings []string

	for _, det := range detectors {
		findings, err := det.Detect(context.Background())
		if err != nil {
			t.Errorf("detector %s: %v", det.Name(), err)
			continue
		}

		for _, f := range findings {
			line := string(f.Rule) + " " + f.Message
			allFindings = append(allFindings, line)
		}
	}

	sort.Strings(allFindings)
	got := strings.Join(allFindings, "\n")

	goldenPath := filepath.Join("testdata", "taskmanager_golden.txt")

	if os.Getenv("CQRS_LINT_UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(goldenPath, []byte(got+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		t.Logf("golden updated: %s (%d findings)", goldenPath, len(allFindings))

		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		if os.IsNotExist(err) {
			t.Skipf("golden file not found at %s — run with CQRS_LINT_UPDATE_GOLDEN=1 to create", goldenPath)
		}

		t.Fatalf("read golden: %v", err)
	}

	if got != strings.TrimRight(string(want), "\n") {
		t.Errorf("findings mismatch (got %d, want different count).\nDiff:\n--- got ---\n%s\n--- want ---\n%s",
			len(allFindings), got, string(want))
	}

	// Assert no critical findings in the reference project.
	for _, f := range allFindings {
		if strings.HasPrefix(f, "C0") && !strings.Contains(f, "C008") {
			t.Errorf("unexpected correctness finding in reference project: %s", f)
		}
	}
}

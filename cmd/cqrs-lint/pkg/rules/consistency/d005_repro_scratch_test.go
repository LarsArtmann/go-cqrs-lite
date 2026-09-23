package consistency

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/ruletest"
)

func TestD005_Repro_FirstTokenOnMixedLine(t *testing.T) {
	tmpDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(
		"module example.com/app\n\nrequire (\n\tgithub.com/larsartmann/go-finding v1.12.0\n\tgithub.com/larsartmann/go-cqrs-lite v4.12.1\n)\n",
	), 0o644)
	_ = os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte(
		"# App\n\nThis project pins go-finding v1.12.0 and go-cqrs-lite v4.12.1 for storage.\n",
	), 0o644)

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	ctx.ProjectRoot = tmpDir

	findings := ruletest.RunDetector(t, NewD005Detector(ctx))
	t.Logf("MIXED-LINE repro: %d D005 findings (expected 0 — doc matches go.mod)", len(findings))
	for _, f := range findings {
		t.Logf("  finding: %+v", f)
	}
	if len(findings) != 0 {
		t.Logf("REPRO CONFIRMED: mixed line fired D005; versions[0] was the OTHER project's token")
	}
}

func TestD005_Repro_HistoricalSentenceMention(t *testing.T) {
	tmpDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(
		"module example.com/app\n\nrequire github.com/larsartmann/go-cqrs-lite v4.12.1\n",
	), 0o644)
	_ = os.WriteFile(filepath.Join(tmpDir, "CHANGELOG.md"), []byte(
		"# Changelog\n\n- 2026-08: upgraded from go-cqrs-lite v4.11.1 to pick up storage fixes.\n",
	), 0o644)

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
	})
	ctx.ProjectRoot = tmpDir

	findings := ruletest.RunDetector(t, NewD005Detector(ctx))
	t.Logf("HISTORICAL repro: %d D005 findings (expected 0 — historical mention, not a current claim)", len(findings))
	for _, f := range findings {
		t.Logf("  finding: %+v", f)
	}
}

